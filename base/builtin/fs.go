package builtin

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"github.com/patrickmn/go-cache"
	"github.com/pborman/uuid"
	"io"
	"os"
	"time"
)

const (
	cmdFilesystemOpen  = "filesystem.open"
	cmdFilesystemRead  = "filesystem.read"
	cmdFilesystemWrite = "filesystem.write"
	cmdFilesystemClose = "filesystem.close"

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

func init() {
	fs := filesystem{
		cache: cache.New(5*time.Minute, 30*time.Second),
	}

	fs.cache.OnEvicted(fs.evicted)

	pm.CmdMap[cmdFilesystemOpen] = process.NewInternalProcessFactory(fs.open)
	pm.CmdMap[cmdFilesystemRead] = process.NewInternalProcessFactory(fs.read)
	pm.CmdMap[cmdFilesystemWrite] = process.NewInternalProcessFactory(fs.write)
	pm.CmdMap[cmdFilesystemClose] = process.NewInternalProcessFactory(fs.close)

}

func (fs *filesystem) evicted(_ string, f interface{}) {
	if fd, ok := f.(*os.File); ok {
		fd.Close()
	}
}

//mode parses python open file modes (
func (fs *filesystem) mode(m string) (int, error) {
	var mode int
	for _, chr := range m {
		switch chr {
		case 'r':
			mode |= os.O_RDONLY
		case 'w':
			mode |= os.O_WRONLY
		case '+':
			mode |= os.O_RDWR
		case 'x':
			mode |= os.O_CREATE
		case 'a':
			mode |= os.O_APPEND
		default:
			return 0, fmt.Errorf("unknown mode '%s'", chr)
		}
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
