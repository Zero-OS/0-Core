package filesystem

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/stream"
	"github.com/zero-os/0-core/base/settings"
)

const (
	CacheBaseDir = "/var/cache"
)

func Hash(s string) string {
	m := md5.New()
	io.WriteString(m, s)
	return fmt.Sprintf("%x", m.Sum(nil))
}

//a helper to close all under laying readers in a flist file stream since decompression doesn't
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
		return nil, fmt.Errorf("invalid flist url (%s)", src)
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

	return nil, fmt.Errorf("unknown flist format %s", ext)
}

func getMetaDB(location, src string) (string, error) {
	reader, err := getMetaDBTar(src)
	if err != nil {
		return "", err
	}

	defer reader.Close()

	archive := tar.NewReader(reader)
	db := fmt.Sprintf("%s.db", location)
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

func MountFList(namespace, storage, src string, target string, hooks ...pm.RunnerHook) error {
	//check
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}

	hash := Hash(src)
	backend := path.Join(CacheBaseDir, namespace, hash)

	os.RemoveAll(backend)
	os.MkdirAll(backend, 0755)

	cache := settings.Settings.Globals.Get("cache", path.Join(CacheBaseDir, "zerofs"))
	g8ufs := []string{
		"-reset",
		"-backend", backend,
		"-cache", cache,
	}

	if strings.HasPrefix(src, "restic:") {
		if err := RestoreRepo(
			strings.TrimPrefix(src, "restic:"),
			path.Join(backend, "ro"),
		); err != nil {
			return err
		}
	} else {
		//assume an flist, an flist requires the meat and storage url
		db, err := getMetaDB(backend, src)
		if err != nil {
			return err
		}

		g8ufs = append(g8ufs,
			"-meta", db,
			"-storage-url", storage,
		)
	}

	g8ufs = append(g8ufs, target)
	cmd := &pm.Command{
		ID:      fmt.Sprintf("%s-g8ufs-%s", namespace, target),
		Command: pm.CommandSystem,
		Arguments: pm.MustArguments(pm.SystemCommandArguments{
			Name: "g8ufs",
			Args: g8ufs,
		}),
	}

	var err error
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
			o.Do(func() {
				if !s {
					err = fmt.Errorf("upnormal exit of filesystem mount at '%s'", target)
				}
				wg.Done()
			})
		},
	})

	pm.Run(cmd, hooks...)

	//wait for either of the hooks (ready or exit)
	wg.Wait()
	return err
}
