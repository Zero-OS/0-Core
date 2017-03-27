package builtin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	fsString = `Label: 'two'  uuid: 5efab9c9-55d8-4f0f-b5b9-4c521b567c70
	Total devices 2 FS bytes used 294912
	devid    1 size 1048576000 used 218103808 path /dev/loop1
	devid    2 size 1048576000 used 218103808 path /dev/loop2
Label: 'single'  uuid: 74595911-0f79-4c2e-925f-105d1279fb48
	Total devices 1 FS bytes used 196608
    devid    1 size 1048576000 used 138412032 path /dev/loop3
`
)

func TestParseFS(t *testing.T) {
	fss, err := btrfsParseList(fsString)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(fss))

	fs := fss[0]
	assert.Nil(t, err)
	assert.Equal(t, "two", fs.Label)
	assert.Equal(t, "5efab9c9-55d8-4f0f-b5b9-4c521b567c70", fs.UUID)
	assert.Equal(t, 2, fs.TotalDevices)
	assert.Equal(t, int64(294912), fs.Used)

	assert.Equal(t, 2, len(fs.Devices))
	dev := fs.Devices[0]
	assert.Equal(t, 1, dev.DevID)
	assert.Equal(t, int64(1048576000), dev.Size)
	assert.Equal(t, int64(218103808), dev.Used)
	assert.Equal(t, "/dev/loop1", dev.Path)
}

var (
	dfString = `Data, single: total=8388608, used=65536
System, single: total=4194304, used=16384
Metadata, single: total=276824064, used=163840
GlobalReserve, single: total=16777216, used=0
`
)

func TestParseDF(t *testing.T) {
	fsinfo := btrfsFSInfo{}
	btrfsParserFilesystemDF(dfString, &fsinfo)
	assert.Equal(t, "single", fsinfo.Data.Profile)
	assert.Equal(t, int64(8388608), fsinfo.Data.Total)
	assert.Equal(t, int64(65536), fsinfo.Data.Used)
	assert.Equal(t, "single", fsinfo.System.Profile)
	assert.Equal(t, int64(4194304), fsinfo.System.Total)
	assert.Equal(t, int64(16384), fsinfo.System.Used)
}

var (
	subvolStr = `ID 259 gen 14 top level 5 path svol
ID 262 gen 21 top level 5 path cobavol

`
)

func TestParseSubvolume(t *testing.T) {
	svs, err := btrfsParseSubvolList(subvolStr)

	assert.Nil(t, err)
	assert.Equal(t, 2, len(svs))

	sv := svs[0]
	assert.Equal(t, sv.ID, 259)
	assert.Equal(t, sv.Gen, 14)
	assert.Equal(t, sv.TopLevel, 5)
	assert.Equal(t, sv.Path, "svol")
}
