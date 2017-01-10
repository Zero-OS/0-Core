package g8ufs

import (
	"fmt"
	"github.com/g8os/g8ufs/meta"
	"github.com/g8os/g8ufs/rofs"
	"github.com/g8os/g8ufs/storage"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
	"github.com/op/go-logging"
	"os"
	"os/exec"
	"path"
	"syscall"
	"time"
)

var (
	log = logging.MustGetLogger("g8ufs")
)

type Starter interface {
	Start() error
	Wait() error
}

type Exec func(name string, arg ...string) Starter

type Options struct {
	//PList (required) to mount
	PList string
	//PListTrim (optional) trim prefix of file paths
	PListTrim string
	//Backend (required) working directory where the filesystem keeps it's cache and others
	//will be created if doesn't exist
	Backend string
	//Mount (required) is the mount point
	Target string
	//MetaStore (optional) will use meta.NewMemoryMeta if not provided
	MetaStore meta.MetaStore
	//Storage (required) storage to download files from
	Storage storage.Storage
	//Reset if set, will wipe up the backend clean before mounting.
	Reset bool

	Exec Exec
}

type G8ufs struct {
	target string
	server *fuse.Server
	cmd    Starter
}

func DefaultExec(name string, arg ...string) Starter {
	return exec.Command(name, arg...)
}

//Mount mounts fuse with given options, it blocks forever until unmount is called on the given mount point
func Mount(opt *Options) (*G8ufs, error) {
	backend := opt.Backend
	ro := path.Join(backend, "ro")
	rw := path.Join(backend, "rw")
	ca := path.Join(backend, "ca")

	for _, name := range []string{ro, rw, ca} {
		if opt.Reset {
			os.RemoveAll(name)
		}
		os.MkdirAll(name, 0755)
	}

	metaStore := opt.MetaStore
	if metaStore == nil {
		metaStore = meta.NewMemoryMetaStore()
	}

	if err := meta.Populate(metaStore, opt.PList, rw, opt.PListTrim); err != nil {
		return nil, err
	}

	fs := rofs.New(opt.Storage, metaStore, ca)

	server, err := fuse.NewServer(
		nodefs.NewFileSystemConnector(
			pathfs.NewPathNodeFs(fs, nil).Root(),
			nil,
		).RawFS(), ro, &fuse.MountOptions{
			AllowOther: true,
			Options:    []string{"ro"},
		})

	if err != nil {
		return nil, err
	}

	go server.Serve()
	log.Debugf("Waiting for fuse mount")
	server.WaitMount()

	log.Debugf("Fuse mount is complete")

	branch := fmt.Sprintf("%s=RW:%s=RO", rw, ro)

	ex := DefaultExec
	if opt.Exec != nil {
		ex = opt.Exec
	}

	cmd := ex("unionfs", "-f",
		"-o", "cow",
		"-o", "allow_other",
		"-o", "default_permissions",
		"-o", "attr_timeout=0",
		"-o", "entry_timeout=0",
		branch, opt.Target)

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	//TODO: find another way to grantee mount is done.
	time.Sleep(time.Second)

	return &G8ufs{
		target: opt.Target,
		server: server,
		cmd:    cmd,
	}, nil
}

//Wait filesystem until it's unmounted.
func (fs *G8ufs) Wait() error {
	defer fs.server.Unmount()
	return fs.cmd.Wait()
}

type errors []interface{}

func (e errors) Error() string {
	return fmt.Sprint(e...)
}

func (fs *G8ufs) Unmount() error {
	var errs errors

	if err := syscall.Unmount(fs.target, syscall.MNT_FORCE|syscall.MNT_DETACH); err != nil {
		errs = append(errs, err)
	}

	if err := fs.server.Unmount(); err != nil {
		errs = append(errs, err)
	}
	return errs
}
