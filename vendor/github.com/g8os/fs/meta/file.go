package meta

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/g8os/fs/utils"
	"io"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
)

const (
	MetaSuffix = ".meta"
)

type metaFile string

func (m metaFile) Stat() MetaState {
	stat, err := os.Stat(string(m))
	if err != nil {
		return MetaInitial
	}

	//mask out everything except the USER perm bits
	return MetaState(stat.Mode()) & MetaStateMask
}

func (m metaFile) SetStat(state MetaState) {
	os.Chmod(string(m), os.FileMode(state))
}

func (m metaFile) String() string {
	return string(m)
}

func (m metaFile) Name() string {
	return strings.TrimSuffix(path.Base(string(m)), MetaSuffix)
}

func (m metaFile) GetEffectiveFilePath() string {
	return strings.TrimSuffix(string(m), MetaSuffix)
}

func (m metaFile) Load() (*MetaData, error) {
	meta := MetaData{}
	_, err := toml.DecodeFile(string(m), &meta)
	if err != nil {
		return nil, err
	}

	return &meta, nil
}

func (m metaFile) Save(meta *MetaData) error {
	p := string(m)
	dir := path.Dir(p)
	os.MkdirAll(dir, os.ModePerm)
	file, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(MetaInitial))
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := toml.NewEncoder(file)
	return encoder.Encode(meta)
}

func (m metaFile) Children() <-chan Meta {
	return nil
}

type metaDir string

func (m metaDir) Stat() MetaState {
	return MetaInitial
}

func (m metaDir) SetStat(state MetaState) {
	//NOOP
}

func (m metaDir) String() string {
	return string(m)
}

func (m metaDir) Name() string {
	return path.Base(string(m))
}

func (m metaDir) GetEffectiveFilePath() string {
	return strings.TrimSuffix(string(m), MetaSuffix)
}

func (m metaDir) Load() (*MetaData, error) {
	var st syscall.Stat_t
	if err := syscall.Stat(string(m), &st); err != nil {
		return nil, err
	}

	return &MetaData{
		Filetype:    syscall.S_IFDIR,
		Inode:       st.Ino,
		Ctime:       uint64(st.Ctim.Sec),
		Mtime:       uint64(st.Mtim.Sec),
		Uid:         st.Uid,
		Gid:         st.Gid,
		Permissions: st.Mode & uint32(os.ModePerm),
	}, nil
}

func (m metaDir) Save(meta *MetaData) error {
	return fmt.Errorf("not supported")
}

func (m metaDir) Children() <-chan Meta {
	d, err := os.Open(string(m))
	if err != nil {
		return nil
	}
	ch := make(chan Meta)
	go func() {
		defer close(ch)
		defer d.Close()

		for {
			entries, err := d.Readdir(100)
			if err != nil && err != io.EOF {
				log.Debugf("directory listing err: %s", err)
				return
			}

			for _, entry := range entries {
				fullname := path.Join(string(m), entry.Name())
				log.Debugf("child: %s", fullname)
				if entry.IsDir() {
					ch <- metaDir(fullname)
				} else {
					ch <- metaFile(fullname)
				}
			}

			if err == io.EOF {
				return
			}
		}

	}()

	return ch
}

type fileMetaStore struct {
	base string
	ino  uint64
}

func NewFileMetaStore(base string) MetaStore {
	return &fileMetaStore{
		base: base,
	}
}

func (s *fileMetaStore) Get(name string) (Meta, bool) {
	base := path.Join(s.base, name)
	st, err := os.Stat(base)
	if err == nil && st.IsDir() {
		log.Debugf("Get meta for '%s'", base)
		return metaDir(base), true
	}

	fullname := fmt.Sprintf("%s%s", base, MetaSuffix)
	if !utils.Exists(fullname) {
		return nil, false
	}

	return metaFile(fullname), true
}

func (s *fileMetaStore) CreateFile(name string) (Meta, error) {
	fullname := path.Join(s.base, fmt.Sprintf("%s%s", name, MetaSuffix))
	m := metaFile(fullname)
	if utils.Exists(fullname) {
		return m, nil
	}

	dir := path.Dir(fullname)
	os.MkdirAll(dir, 0755)

	if err := m.Save(&MetaData{
		Inode: atomic.AddUint64(&s.ino, 1),
	}); err != nil {
		return nil, err
	}

	return m, nil
}

func (s *fileMetaStore) CreateDir(name string) (Meta, error) {
	fullname := path.Join(s.base, name)
	m := metaDir(fullname)
	if err := os.MkdirAll(fullname, 0755); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *fileMetaStore) Delete(meta Meta) error {
	return os.Remove(meta.String())
}

func (s *fileMetaStore) Populate(plist string, trim string) error {
	var parsed = 0

	iter, err := utils.IterFlistFile(plist)
	if err != nil {
		return err
	}

	log.Infof("Populating mountpoint...")

	for line := range iter {
		entity, err := ParseLine(line, trim)
		if err != nil {
			return err
		}

		if entity.Filetype == syscall.S_IFDIR {
			s.CreateDir(entity.Filepath)
			continue
		}

		m, err := s.CreateFile(entity.Filepath)
		if err != nil {
			return err
		}

		// user and group id
		uid := 0
		u, err := user.Lookup(entity.Uname)
		if err == nil {
			uid, _ = strconv.Atoi(u.Uid)
		}

		gid := 0
		g, err := user.LookupGroup(entity.Gname)
		if err == nil {
			gid, _ = strconv.Atoi(g.Gid)
		}

		ino := atomic.AddUint64(&s.ino, 1)

		data := &MetaData{
			Hash:        entity.Hash,
			Size:        uint64(entity.Filesize),
			Uname:       entity.Uname,
			Uid:         uint32(uid),
			Gname:       entity.Gname,
			Gid:         uint32(gid),
			Permissions: uint32(entity.Permissions),
			Filetype:    entity.Filetype,
			Ctime:       uint64(entity.Ctime.Unix()),
			Mtime:       uint64(entity.Mtime.Unix()),
			Extended:    entity.Extended,
			DevMajor:    entity.DevMajor,
			DevMinor:    entity.DevMinor,
			Inode:       ino,
		}

		if !m.Stat().Modified() {
			//both meta and file exists. This file wasn't modified we can
			//just now place the meta and delete the file ONLY if file was changed.

			oldMeta, err := m.Load()
			if err != nil && !os.IsNotExist(err) {
				return err
			}

			if oldMeta.Hash != entity.Hash {
				//TODO: remove old data file
				//os.Remove(file)
			}
		}

		if err := m.Save(data); err != nil {
			return err
		}

		parsed += 1
	}

	log.Infof("Mountpoint populated: %v items parsed", parsed)

	return nil
}
