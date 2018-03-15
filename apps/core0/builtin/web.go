package builtin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/zero-os/0-core/base/pm"
)

func init() {
	pm.RegisterBuiltIn("web.download", downloadCmd)
}

func downloadCmd(cmd *pm.Command) (interface{}, error) {
	var args struct {
		URL         string `json:"url"`
		Destination string `json:"destination"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	return download(args.URL, args.Destination)
}

var errBadArgument = fmt.Errorf("url and destination argument must be provided")

func download(url, dest string) (interface{}, error) {
	if url == "" || dest == "" {
		return nil, errBadArgument
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0660)
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return nil, err
	}

	return dest, nil
}
