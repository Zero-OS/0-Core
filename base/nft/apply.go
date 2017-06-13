package nft

import (
	"github.com/zero-os/0-core/base/pm"
	"io/ioutil"
	"os"
)

func ApplyFromFile(cfg string) error {
	_, err := pm.GetManager().System("nft", "-f", cfg)
	return err
}

func Apply(nft *Nft) error {
	data, err := nft.MarshalText()
	if err != nil {
		return err
	}
	f, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		os.RemoveAll(f.Name())
	}()

	if _, err := f.Write(data); err != nil {
		return err
	}
	f.Close()

	return ApplyFromFile(f.Name())
}
