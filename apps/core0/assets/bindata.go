// Code generated by go-bindata.
// sources:
// text/hub.pgp.txt
// text/logo.txt
// DO NOT EDIT!

package assets

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _textHubPgpTxt = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x74\xd5\xb9\x0e\xb3\xd8\x96\x86\xe1\x9c\xab\xf8\x73\xd4\x02\x33\x13\x74\xb0\xcd\x3c\xb3\x99\x21\x03\xdb\xcc\x33\x06\x0c\x57\xdf\xaa\x52\xa9\xa4\x56\xf7\x59\xe9\xa7\x15\x3e\x7a\xff\xeb\xaf\x7b\x4a\x8a\x66\xff\x71\x15\xf7\x8f\x1b\x3e\x4d\x4d\xf8\x63\x48\xe9\x9f\xa7\xe9\x08\xc6\xdf\x33\x82\x0c\x50\xb2\x9f\xf2\x52\x0c\xae\xf4\x14\x80\x98\x59\xb2\xac\x5c\x96\xd6\xbd\xbd\x35\xe5\x6e\xda\xe5\xf1\x53\x40\x9f\x7c\x4c\xfe\x3e\xa2\xee\x2e\xec\xb4\x85\x4b\xe2\xc0\xdb\x77\x74\x06\xae\x5f\x0f\x09\xd5\x2f\x76\xd4\x66\xd3\x55\x6a\x6d\x3d\x19\xad\xa0\xa4\xab\xa9\x89\xaf\x38\xb3\xad\x76\x9f\x09\x15\x59\x54\xeb\x9d\x2a\xe6\xd2\x23\xd6\x95\xcd\x01\x5f\x27\x9e\x95\xaa\x1e\x9e\x1a\x55\xbc\x6f\x24\x09\xca\x80\x9a\x92\xd3\x2c\x09\x32\x53\x92\x12\x27\x13\xf6\xa4\xd3\xea\x3a\x66\x5a\xbb\x4a\xf6\xd8\x72\x63\x7f\x38\x1b\x73\x30\xd7\x12\x9b\x98\x20\x2c\x50\x49\x67\xfe\x38\xd4\x51\xbc\xf7\x37\x32\x51\xa1\xd3\x6a\x3b\x2f\x3e\xcf\x49\x66\x07\xf7\xa5\x7e\x18\x4c\xcd\xcc\xcd\x2a\xc3\xd0\xdf\x36\x87\x92\x16\xff\xb2\xe3\xe8\xca\x64\x6b\xe4\x5c\xbd\xa0\x00\xcd\x80\xcb\xba\x7b\x79\x6b\x9f\x2c\x32\x1b\x69\xe3\x87\xf7\x51\x71\xed\xd5\x87\x3b\x88\x75\x5a\x7e\x28\x57\x7c\xe8\x84\x62\x43\xb8\x5a\xf9\xfd\x20\xcf\x31\xa2\x4a\xc0\x13\x30\x21\xbc\x72\x95\x3a\x16\x63\x7b\x83\x98\x43\x25\xcf\x90\xed\xc5\xcc\xea\xc5\x67\x7a\xb5\xfd\x5e\x18\xdf\xf7\xb4\xc3\x94\x8e\xe1\x2b\xd9\xa0\x2a\x31\xc7\x87\xa8\x69\x83\xa7\xf4\x04\x40\xc1\x0d\xe9\xf1\xc8\x15\xf9\x9b\xc6\xb0\x82\xc9\xfc\xfd\xf8\x80\x43\x8a\x24\x9a\xd2\xf8\x51\x67\x8a\x0c\x32\x52\xef\xb3\x98\x9e\xdf\xca\x7b\x2b\x08\xbd\x37\x07\xfb\x28\x02\x46\x07\x21\x25\x49\xa7\xa4\x01\xb1\x8a\x35\xe8\x7d\x31\x3b\x1d\x23\xfc\x84\x71\x2e\xe3\x08\x11\xb0\x0a\xe0\xa5\x40\x8d\x6f\x6f\xaf\x42\x21\x9e\xf7\x94\x83\x5a\x01\xce\xd0\x14\x60\xa5\x82\x2a\x8d\x84\xaa\xd3\x84\x53\x93\xe4\x4a\x13\x01\xd4\x3e\x00\x6a\x49\x05\x80\x21\x78\xa2\x82\xfc\xfb\xbc\xfd\x24\x01\x08\x47\x7c\xc4\x8b\x05\xfb\x7c\x5c\x99\xaf\xd7\x60\xa3\x45\x62\xb6\x7c\xb6\x29\xd0\x53\xea\xa1\x4d\x82\x6f\x04\x02\xf1\x89\x5c\x25\xce\x28\xed\x9b\xdf\x48\xf4\xf5\x89\x66\x9a\xc4\xbb\x54\xa2\xf3\xb5\x74\x9c\x0d\x32\x3b\xe2\xcc\x2b\x51\x12\xba\x90\xd1\xb6\xa5\x04\x34\xb1\x07\x1d\x9d\xb7\x9b\xb2\x1c\xdc\xc8\x97\x97\xd4\xb2\xad\x27\x26\xce\x29\x75\x48\xf2\x0a\x4f\x30\x96\x14\x27\xeb\x73\xff\xfe\xbc\xac\x98\x0c\xc5\x80\x1d\xcc\x49\x58\x24\x52\xdd\xeb\x4a\x0d\xa2\xa2\x12\x64\xc9\xed\x72\xe1\xa9\xba\x55\x66\xf8\x22\xf6\x71\xa1\x8a\x07\x02\x4c\x10\xc3\x36\xb6\x1a\x4f\x02\x96\x33\xe2\x70\xb2\xb3\x90\x82\x09\x03\x3e\x4f\xf0\x45\x29\xdd\xa8\x20\xc1\x7b\x14\xd7\x94\x4d\xf1\xbb\x9e\x56\x35\x7e\x86\xfc\xf5\xb2\x4e\x97\x55\x16\x11\x8d\xdc\x19\x43\x5a\xd9\x01\x2f\x18\xf0\xdf\xfe\x29\x57\x0d\x75\x65\xab\xff\x7d\xa0\xb3\x77\x9a\x03\x14\x61\x21\x35\xa1\x00\x7a\xa1\x14\x7d\x3b\xd7\x67\x45\x85\x80\xe2\x78\x2d\xac\xcf\xaf\x58\x91\xa6\xaf\x30\x48\xea\x7d\xf2\x52\x4f\x98\x47\x36\xb8\x09\x76\x6c\x02\xad\xdf\xfb\x67\x99\xde\x57\xed\xe3\xb8\x9d\x39\x49\xe8\xec\xff\x0b\x9c\xb0\x27\x2d\xda\xed\x3b\x6e\xa2\x1c\x23\x21\xcb\xd6\x3f\xac\x4b\x14\x7e\xe8\x84\xad\x0b\x94\xb6\xad\xb2\x0b\x5c\xd1\x4c\x07\xd3\x3c\x48\xe6\xef\x56\x6d\x93\x1b\x92\x4b\x3f\x65\x55\xe3\xd1\x73\x88\x8a\xf0\xd0\xbb\x98\x24\x74\x66\xcb\x16\x24\xaa\xaa\xf7\x78\x2d\x3b\x49\x1c\xa4\x65\xa0\x54\x17\x5d\xbb\x5f\x1e\x0a\x33\xa2\xfb\x25\xf9\xfa\xd2\x04\x7c\xe6\x99\xc6\x6c\x8f\x1d\xce\x8b\xc4\x24\xae\xd9\x36\x5e\x34\x40\x2b\x1e\x38\x29\x85\xd0\x8e\x3e\x5d\x8f\x3c\x4d\xb3\xac\x4d\xb3\x80\x41\xcb\xe8\xa6\x3c\xb0\x26\x7b\xce\xf7\xe8\xda\x76\xda\x24\xac\x7e\x9c\x4a\x35\x14\x0d\xa5\x66\xda\x9f\xff\x3d\xb0\x54\x31\x71\x27\xa4\x16\x3d\x47\x50\xa9\x3e\x9e\x69\x61\x6f\x10\x55\x96\xbd\x9b\x7c\xa5\x67\xbd\x70\x74\x07\x23\x08\x18\xe2\x57\x0b\x54\xe0\x6d\xdb\x64\x07\xed\xc2\x3d\xea\x45\x0f\xc2\x31\x3f\xfb\xf8\xae\x4d\xcf\x14\xc1\xbb\x44\xb4\x2c\x4f\x88\xce\xc1\x5b\x65\xc5\x9d\x6f\x72\xaa\xc2\xc0\x7a\x82\x0c\xd7\xf2\x7d\x84\x36\x95\x1c\xf4\x95\x92\x92\x06\x82\xd6\xeb\xd1\xac\x04\xf5\xe1\x32\x51\x42\x0e\x67\xa3\x6a\x31\x5f\x27\x08\x49\x4f\x72\xe1\xd7\xdf\x7f\xc4\xe9\x20\x48\x25\x05\x48\x1a\x10\xc0\xff\x81\xf3\xff\xb9\x41\xfe\x82\x23\xfe\x03\xe3\xdf\xf1\x4b\xe4\x02\x10\x86\x18\xdc\x60\x22\xf2\x28\xad\xec\xfa\x65\x2c\x52\xd1\x6e\xdb\xbe\x28\x3f\xea\x1c\x71\xb4\x66\xf9\x61\xb8\x50\xc4\x34\x25\x3f\xda\x87\xb9\xc7\xcf\x05\x43\xa3\x70\x7d\x63\x62\xe7\x14\x33\x6e\x78\xd2\xf8\x1b\x9f\x29\x17\x0b\x77\xc1\x84\x49\xf4\x03\x4a\xfc\x23\x47\x69\xc5\x93\x43\xbd\x3c\xb2\x9f\x56\xdc\x11\x10\x53\xd3\x04\xfa\x09\xc3\xc2\x0f\x17\x18\x1d\xef\x1d\x70\x19\x41\xe3\xab\xf4\x40\x0d\x9b\x77\xf2\x63\x70\x05\xcb\x63\xbb\x3c\x86\x5a\xcb\x07\x8d\x44\xe2\x67\x9b\xe5\xd9\xdd\x00\x77\x10\x08\x13\xf1\x93\x5e\xe3\x2a\xe8\xf3\x54\xa7\x03\x97\xf7\x02\x0c\x7d\xc7\xfb\x52\x57\x31\x1d\x49\xca\x71\x1c\xdd\xf0\x53\x5a\xad\x6e\x3e\xb7\xde\xbc\x6a\xec\x31\x86\x26\xbc\xcb\x17\x54\x80\x52\x34\x9f\x08\x89\x3f\xac\x37\xff\x9a\xec\x74\x1a\xab\x36\x98\x33\x65\xdf\xe8\xd4\xb2\x67\x15\x54\x62\x5e\x79\x94\x16\x1f\x2a\x3b\x5e\x7a\x38\xac\xf0\x33\x07\xf9\x1c\x72\x24\x71\x15\x93\xba\xe0\x25\x89\xa7\x3c\xd2\x4f\x6c\x72\x05\x44\xc4\x7b\x87\x58\x48\xec\xb0\x54\x06\x9d\x38\x7c\xf9\x43\x0f\x97\xf7\x6c\xd5\x4f\xde\xae\xa5\x7c\xe3\x60\xf1\x96\xba\xe2\x56\xe4\xbf\xf1\x42\x9d\x91\xbf\xb3\x24\xd9\xe2\x7f\x6e\xd6\xff\x04\x00\x00\xff\xff\x51\xf1\xac\x49\xd8\x06\x00\x00")

func textHubPgpTxtBytes() ([]byte, error) {
	return bindataRead(
		_textHubPgpTxt,
		"text/hub.pgp.txt",
	)
}

func textHubPgpTxt() (*asset, error) {
	bytes, err := textHubPgpTxtBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "text/hub.pgp.txt", size: 1752, mode: os.FileMode(420), modTime: time.Unix(1532761906, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _textLogoTxt = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x3c\x8a\xb1\x0d\xc0\x30\x0c\xc3\xf6\x5c\xc1\xad\xed\xe4\x87\x0c\xf0\x11\x1d\x5f\xb8\x09\xaa\x41\xb4\x09\x81\x3b\xc8\x94\x32\xf9\xa8\x2e\x22\x14\xd2\x5c\xba\x8f\x23\x0a\xcd\x9a\xa7\x66\x5b\x84\x70\xfb\xfc\x6c\xa5\x17\xa5\xb6\x1a\xc3\x38\x8b\xc3\x4c\xbd\x01\x00\x00\xff\xff\x46\x68\x46\x10\x82\x00\x00\x00")

func textLogoTxtBytes() ([]byte, error) {
	return bindataRead(
		_textLogoTxt,
		"text/logo.txt",
	)
}

func textLogoTxt() (*asset, error) {
	bytes, err := textLogoTxtBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "text/logo.txt", size: 130, mode: os.FileMode(420), modTime: time.Unix(1521537949, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"text/hub.pgp.txt": textHubPgpTxt,
	"text/logo.txt": textLogoTxt,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}
var _bintree = &bintree{nil, map[string]*bintree{
	"text": &bintree{nil, map[string]*bintree{
		"hub.pgp.txt": &bintree{textHubPgpTxt, map[string]*bintree{}},
		"logo.txt": &bintree{textLogoTxt, map[string]*bintree{}},
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}

