package g8ufs

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"github.com/op/go-logging"
	"github.com/pborman/uuid"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
	"github.com/zero-os/0-core/base/pm/stream"
	"github.com/zero-os/0-core/base/settings"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"
)

var (
	log = logging.MustGetLogger("g8ufs")
)

//a helper to close all under laying readers in a plist file stream since decompression doesn't
//auto close the under laying layer.
type underLayingCloser struct {
	readers []io.Reader
}

//close all layers.
func (u *underLayingCloser) Close() error {
	for i := len(u.readers) - 1; i >= 0; i-- {
		r := u.readers[i]
		if c, ok := r.(io.Closer); ok {
			c.Close()
		}
	}

	return nil
}

//read only from the last layer.
func (u *underLayingCloser) Read(p []byte) (int, error) {
	return u.readers[len(u.readers)-1].Read(p)
}

func getMetaDBTar(src string) (io.ReadCloser, error) {
	u, err := url.Parse(src)
	if err != nil {
		return nil, err
	}

	var reader io.ReadCloser
	base := path.Base(u.Path)

	if u.Scheme == "file" || u.Scheme == "" {
		// check file exists
		_, err := os.Stat(u.Path)
		if err != nil {
			return nil, err
		}
		reader, err = os.Open(u.Path)
		if err != nil {
			return nil, err
		}
	} else if u.Scheme == "http" || u.Scheme == "https" {
		response, err := http.Get(src)
		if err != nil {
			return nil, err
		}

		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to download flist: %s", response.Status)
		}

		reader = response.Body
	} else {
		return nil, fmt.Errorf("invalid plist url (%s)", src)
	}

	var closer underLayingCloser
	closer.readers = append(closer.readers, reader)

	ext := path.Ext(base)
	switch ext {
	case ".tgz":
		fallthrough
	case ".flist":
		fallthrough
	case ".gz":
		if r, err := gzip.NewReader(reader); err != nil {
			closer.Close()
			return nil, err
		} else {
			closer.readers = append(closer.readers, r)
		}
		return &closer, nil
	case ".tbz2":
		fallthrough
	case ".bz2":
		closer.readers = append(closer.readers, bzip2.NewReader(reader))
		return &closer, err
	case ".tar":
		return &closer, nil
	}

	return nil, fmt.Errorf("unknown plist format %s", ext)
}

func getMetaDB(source, backend string) (string, error) {
	reader, err := getMetaDBTar(source)
	if err != nil {
		return "", err
	}

	defer reader.Close()

	archive := tar.NewReader(reader)
	db := path.Join(backend, "db")
	log.Debugf("Extracting meta to %s", db)
	if err := os.MkdirAll(db, 0755); err != nil {
		return "", err
	}

	for {
		header, err := archive.Next()
		if err != nil && err != io.EOF {
			return "", err
		} else if err == io.EOF {
			break
		}

		if header.FileInfo().IsDir() {
			continue
		}

		base := path.Join(db, path.Dir(header.Name))
		log.Debugf("extracting: %s", header.Name)
		if err := os.MkdirAll(base, 0755); err != nil {
			return "", err
		}

		file, err := os.Create(path.Join(db, header.Name))
		if err != nil {
			return "", err
		}

		if _, err := io.Copy(file, archive); err != nil {
			file.Close()
			return "", err
		}

		file.Close()
	}

	return db, nil
}

func Mount(source, target, backend, storage string, cache string, hooks ...pm.RunnerHook) error {
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}

	os.RemoveAll(backend)
	os.MkdirAll(backend, 0755)

	db, err := getMetaDB(source, backend)
	if err != nil {
		return err
	}

	if storage == "" {
		storage = settings.Settings.Globals.Get("storage", "ardb://hub.gig.tech:16379")
	}

	cmd := &core.Command{
		ID:      uuid.New(),
		Command: process.CommandSystem,
		Arguments: core.MustArguments(process.SystemCommandArguments{
			Name: "g8ufs",
			Args: []string{
				"-reset",
				"-backend", backend,
				"-cache", cache,
				"-meta", db,
				"-storage-url", storage,
				target},
			NoOutput: false, //this can't be set to true other wise the MatchHook below won't work
		}),
	}

	var o sync.Once
	var wg sync.WaitGroup
	wg.Add(1)

	hooks = append(hooks, &pm.MatchHook{
		Match: "mount starts",
		Action: func(_ *stream.Message) {
			o.Do(wg.Done)
		},
	}, &pm.ExitHook{
		Action: func(s bool) {
			log.Debugf("mount point '%s' exited with '%v'", target, s)
			o.Do(func() {
				if !s {
					err = fmt.Errorf("upnormal exit of filesystem mount at '%s'", target)
				}
				wg.Done()
			})
		},
	})

	pm.GetManager().RunCmd(cmd, hooks...)

	//wait for either of the hooks (ready or exit)
	wg.Wait()
	return err
}
