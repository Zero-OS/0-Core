package builtin

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"time"

	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"github.com/patrickmn/go-cache"
	"github.com/pborman/uuid"
)

const (
	cmdFilesystemOpen   = "filesystem.open"
	cmdFilesystemRead   = "filesystem.read"
	cmdFilesystemWrite  = "filesystem.write"
	cmdFilesystemClose  = "filesystem.close"
	cmdFilesystemMkDir  = "filesystem.mkdir"
	cmdFilesystemRemove = "filesystem.remove"
	cmdFilesystemChmod  = "filesystem.chmod"
	cmdFilesystemChown  = "filesystem.chown"
	cmdFilesystemExists = "filesystem.exists"
	cmdFilesystemList   = "filesystem.list"
	cmdFilesystemMove   = "filesystem.move"

	fsReadBS = 512 * 1024 //512K
)

type filesystem struct {
	cache *cache.Cache
}

type FSOpenArgs struct {
	File string `json:"file"`
	Mode string `json:"mode"`
	Perm uint32 `json:"perm"`
}

type FSFileDescriptorArgs struct {
	FD string `json:"fd"`
}

type FSWriteArgs struct {
	FSFileDescriptorArgs
	Block string `json:"block"`
}

type FSPathArgs struct {
	Path string `json:"path"`
}

type FSChmodArgs struct {
	FSPathArgs
	Mode      os.FileMode `json:"mode"`
	Recursive bool        `json:"recursive"`
}

type FSChownArgs struct {
	FSPathArgs
	User      string `json:"user"`
	Group     string `json:"group"`
	Recursive bool   `json:"recursive"`
}

type FSMoveArgs struct {
	FSPathArgs
	Destination string `json:"destination"`
}

type FSEntry struct {
	Name  string      `json:"name"`   // base name of the file
	Size  int64       `json:"size"`   // length in bytes for regular files; system-dependent for others
	Mode  os.FileMode `json:"mode"`   // file mode bits
	IsDir bool        `json:"is_dir"` // abbreviation for Mode().IsDir()
}

func init() {
	fs := filesystem{
		cache: cache.New(5*time.Minute, 30*time.Second),
	}

	fs.cache.OnEvicted(fs.evicted)

	pm.CmdMap[cmdFilesystemOpen] = process.NewInternalProcessFactory(fs.open)
	pm.CmdMap[cmdFilesystemRead] = process.NewInternalProcessFactory(fs.read)
	pm.CmdMap[cmdFilesystemWrite] = process.NewInternalProcessFactory(fs.write)
	pm.CmdMap[cmdFilesystemClose] = process.NewInternalProcessFactory(fs.close)
	pm.CmdMap[cmdFilesystemMkDir] = process.NewInternalProcessFactory(fs.mkdir)
	pm.CmdMap[cmdFilesystemRemove] = process.NewInternalProcessFactory(fs.remove)
	pm.CmdMap[cmdFilesystemChmod] = process.NewInternalProcessFactory(fs.chmod)
	pm.CmdMap[cmdFilesystemChown] = process.NewInternalProcessFactory(fs.chown)
	pm.CmdMap[cmdFilesystemExists] = process.NewInternalProcessFactory(fs.exists)
	pm.CmdMap[cmdFilesystemList] = process.NewInternalProcessFactory(fs.list)
	pm.CmdMap[cmdFilesystemMove] = process.NewInternalProcessFactory(fs.move)
}

func (fs *filesystem) evicted(_ string, f interface{}) {
	if fd, ok := f.(*os.File); ok {
		fd.Close()
	}
}

//mode parses python open file modes (
func (fs *filesystem) mode(m string) (int, error) {
	var mode int
	rwax := 0
	readable := false
	writable := false

	for _, chr := range m {
		switch chr {
		case 'r':
			rwax += 1
			readable = true
		case 'x':
			rwax += 1
			writable = true
			mode |= os.O_CREATE | os.O_EXCL
		case 'w':
			rwax += 1
			writable = true
			mode |= os.O_CREATE | os.O_TRUNC
		case 'a':
			rwax += 1
			writable = true
			mode |= os.O_CREATE | os.O_APPEND
		case '+':
			readable = true
			writable = true
		default:
			return 0, fmt.Errorf("unknown mode '%c'", chr)
		}
	}

	if rwax != 1 {
		return 0, fmt.Errorf("rwax modes has to be used once and only once")
	}

	if readable && writable {
		mode |= os.O_RDWR
	} else if writable {
		mode |= os.O_WRONLY
	} else {
		mode |= os.O_RDONLY
	}

	return mode, nil
}

func (fs *filesystem) open(cmd *core.Command) (interface{}, error) {
	var args FSOpenArgs
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	mode, err := fs.mode(args.Mode)
	if err != nil {
		return nil, err
	}
	fd, err := os.OpenFile(args.File, mode, os.ModePerm&os.FileMode(args.Perm))

	if err != nil {
		return nil, err
	}

	id := uuid.New()
	fs.cache.Set(id, fd, cache.DefaultExpiration)

	return id, nil
}

func (fs *filesystem) close(cmd *core.Command) (interface{}, error) {
	var args FSFileDescriptorArgs
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	//this will call the Evict function, hence closing the file.
	fs.cache.Delete(args.FD)

	return nil, nil
}

func (fs *filesystem) read(cmd *core.Command) (interface{}, error) {
	var args FSFileDescriptorArgs
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	f, ok := fs.cache.Get(args.FD)
	if !ok {
		return nil, fmt.Errorf("unknown file description '%s'", args.FD)
	}
	// refresh cache expiration
	fs.cache.Set(args.FD, f, cache.DefaultExpiration)

	fd, ok := f.(*os.File)
	if !ok {
		return nil, fmt.Errorf("internal server error (invalid file descriptor)")
	}

	buffer := make([]byte, fsReadBS)

	n, err := fd.Read(buffer)
	if err == io.EOF {
		err = nil
	}

	if err != nil {
		return nil, err
	}

	return base64.StdEncoding.EncodeToString(buffer[0:n]), err
}

func (fs *filesystem) write(cmd *core.Command) (interface{}, error) {
	var args FSWriteArgs
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	f, ok := fs.cache.Get(args.FD)
	if !ok {
		return nil, fmt.Errorf("unknown file description '%s'", args.FD)
	}
	// refresh cache expiration
	fs.cache.Set(args.FD, f, cache.DefaultExpiration)

	fd, ok := f.(*os.File)
	if !ok {
		return nil, fmt.Errorf("internal server error (invalid file descriptor)")
	}

	buffer, err := base64.StdEncoding.DecodeString(args.Block)
	if err != nil {
		return nil, err
	}

	if _, err := fd.Write(buffer); err != nil {
		return nil, err
	}

	return nil, nil
}

func (fs *filesystem) mkdir(cmd *core.Command) (interface{}, error) {
	var p FSPathArgs
	if err := json.Unmarshal(*cmd.Arguments, &p); err != nil {
		return nil, err
	}

	return nil, os.MkdirAll(p.Path, 0755)
}

func (fs *filesystem) remove(cmd *core.Command) (interface{}, error) {
	var p FSPathArgs
	if err := json.Unmarshal(*cmd.Arguments, &p); err != nil {
		return nil, err
	}

	return nil, os.RemoveAll(p.Path)
}

func (fs *filesystem) exists(cmd *core.Command) (interface{}, error) {
	var p FSPathArgs
	if err := json.Unmarshal(*cmd.Arguments, &p); err != nil {
		return nil, err
	}

	_, err := os.Stat(p.Path)
	return !os.IsNotExist(err), nil
}

func (fs *filesystem) list(cmd *core.Command) (interface{}, error) {
	var p FSPathArgs
	if err := json.Unmarshal(*cmd.Arguments, &p); err != nil {
		return nil, err
	}

	entries, err := ioutil.ReadDir(p.Path)
	if err != nil {
		return nil, err
	}

	results := make([]FSEntry, 0, len(entries))
	for _, entry := range entries {
		results = append(results,
			FSEntry{
				Name:  entry.Name(),
				Size:  entry.Size(),
				Mode:  entry.Mode() & os.ModePerm,
				IsDir: entry.IsDir(),
			},
		)
	}

	return results, nil
}

func (fs *filesystem) chmod(cmd *core.Command) (interface{}, error) {
	var p FSChmodArgs
	if err := json.Unmarshal(*cmd.Arguments, &p); err != nil {
		return nil, err
	}

	if !p.Recursive {
		return nil, os.Chmod(p.Path, os.ModePerm&p.Mode)
	}

	walk := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			//skip files with problems
			return nil
		}
		os.Chmod(path, os.ModePerm&p.Mode)
		return nil
	}

	//recursive chmod
	return nil, filepath.Walk(p.Path, walk)
}

func (fs *filesystem) chown(cmd *core.Command) (interface{}, error) {
	var args FSChownArgs
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	u, err := user.Lookup(args.User)
	if err != nil {
		return nil, err
	}
	g, err := user.LookupGroup(args.Group)
	if err != nil {
		return nil, err
	}

	//unix
	uid, _ := strconv.ParseInt(u.Uid, 10, 64)
	gid, _ := strconv.ParseInt(g.Gid, 10, 64)

	if !args.Recursive {
		return nil, os.Chown(args.Path, int(uid), int(gid))
	}

	walk := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			//skip files with problems
			return nil
		}
		os.Chown(path, int(uid), int(gid))
		return nil
	}

	//recursive chown
	return nil, filepath.Walk(args.Path, walk)
}

func (fs *filesystem) move(cmd *core.Command) (interface{}, error) {
	var p FSMoveArgs
	if err := json.Unmarshal(*cmd.Arguments, &p); err != nil {
		return nil, err
	}

	return nil, os.Rename(p.Path, p.Destination)
}
