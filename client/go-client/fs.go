package client

import (
	"encoding/base64"
	"io"
)

type FilesystemManager interface {
	Upload(reader io.Reader, p string) error
	Download(p string, writer io.Writer) error
	Remove(p string) error
	Exists(p string) (bool, error)
}

type File string

type fsMgr struct {
	cl Client
}

func Filesystem(client Client) FilesystemManager {
	return &fsMgr{client}
}

func (f *fsMgr) Open(p string, mode string, perm int) (File, error) {
	var fl File
	res, err := sync(f.cl, "filesystem.open", A{
		"file": p,
		"mode": mode,
		"perm": perm,
	})

	if err != nil {
		return fl, err
	}

	if err := res.Json(&fl); err != nil {
		return fl, err
	}

	return fl, nil
}

func (f *fsMgr) Close(file File) error {
	_, err := sync(f.cl, "filesystem.close", A{
		"fd": file,
	})
	return err
}

func (f *fsMgr) Read(file File) ([]byte, error) {
	res, err := sync(f.cl, "filesystem.read", A{
		"fd": file,
	})
	if err != nil {
		return nil, err
	}

	var buffer string
	if err := res.Json(&buffer); err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(buffer)
}

func (f *fsMgr) Write(file File, data []byte) error {
	_, err := sync(f.cl, "filesystem.write", A{
		"fd":    file,
		"block": base64.StdEncoding.EncodeToString(data),
	})

	return err
}

func (f *fsMgr) Upload(reader io.Reader, p string) error {
	fd, err := f.Open(p, "w", 0644)
	if err != nil {
		return err
	}

	defer f.Close(fd)

	buffer := make([]byte, 512*1024)
	for {
		n, err := reader.Read(buffer)
		if err != nil && err != io.EOF {
			return err
		}

		f.Write(fd, buffer[:n])

		if err == io.EOF {
			break
		}
	}

	return nil
}

func (f *fsMgr) Download(p string, writer io.Writer) error {
	fd, err := f.Open(p, "r", 0644)
	if err != nil {
		return err
	}

	defer f.Close(fd)

	for {
		block, err := f.Read(fd)
		if err != nil {
			return err
		}
		if len(block) == 0 {
			break
		}

		if _, err := writer.Write(block); err != nil {
			return err
		}
	}

	return nil
}

func (f *fsMgr) Remove(p string) error {
	_, err := sync(f.cl, "filesystem.remove", A{
		"path": p,
	})

	return err
}

func (f *fsMgr) Exists(p string) (bool, error) {
	var exist bool
	res, _ := sync(f.cl, "filesystem.exists", A{
		"path": p,
	})
	if err := res.Json(&exist); err != nil {
		return exist, err
	}

	return exist, nil
}
