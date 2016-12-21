package files

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/hanwen/go-fuse/fuse"
)

// Mkdir creates a directory
func (fs *fileSystem) Mkdir(path string, mode uint32, context *fuse.Context) fuse.Status {
	fullPath := fs.GetPath(path)

	backendFn := func() fuse.Status {
		return fuse.ToStatus(os.Mkdir(fullPath, os.FileMode(mode)))
	}

	metaFn := func() { fs.meta.CreateDir(path) }

	if st := backendFn(); st != fuse.ENOENT {
		metaFn()
		return st
	}

	// only populate directories above it.
	fs.populateDirFile(filepath.Dir(strings.TrimSuffix(path, "/")))

	if st := backendFn(); st != fuse.OK {
		return st
	}

	metaFn()
	// This line break mkdir on OL
	// fs.tracker.Touch(fullPath)
	return fuse.OK
}

// Rmdir deletes a directory
func (fs *fileSystem) Rmdir(name string, context *fuse.Context) (code fuse.Status) {
	fullPath := fs.GetPath(name)
	log.Debugf("Rmdir %v", fullPath)
	m, exists := fs.meta.Get(name)
	if !exists {
		return fuse.ENOENT
	}

	if err := os.Remove(fullPath); err != nil {
		return fuse.ToStatus(err)
	}

	return fuse.ToStatus(fs.meta.Delete(m))
}

// OpenDir opens a directory and return all files/dir in the directory.
// If it finds .meta file, it shows the file represented by that meta
func (fs *fileSystem) OpenDir(name string, context *fuse.Context) (stream []fuse.DirEntry, status fuse.Status) {
	log.Debugf("OpenDir %v", fs.GetPath(name))
	m, exists := fs.meta.Get(name)
	if !exists {
		return nil, fuse.ENOENT
	}
	var output []fuse.DirEntry
	log.Debugf("Listing children in directory %s", name)

	for child := range m.Children() {
		data, err := child.Load()
		if err != nil {
			return nil, fuse.ToStatus(err)
		}
		output = append(output, fuse.DirEntry{
			Name: child.Name(),
			Mode: data.Filetype,
		})
	}

	return output, fuse.OK
}
