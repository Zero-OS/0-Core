package builtin

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWebDownload(t *testing.T) {
	req := require.New(t)

	tt := []struct {
		url  string
		dest string
		err  error
	}{
		{
			url:  "http://google.com",
			dest: "/tmp/google.html",
			err:  nil,
		},
		{
			url:  "",
			dest: "/tmp/google.html",
			err:  errBadArgument,
		},
		{
			url:  "http://google.com",
			dest: "",
			err:  errBadArgument,
		},
		{
			url:  "wrong_format",
			dest: "/tmp/google.html",
			err:  fmt.Errorf("can't download"),
		},
	}

	for _, tc := range tt {
		t.Run(fmt.Sprintf("url:%s - dest %s", tc.url, tc.dest), func(t *testing.T) {
			_, err := download(tc.url, tc.dest)
			if tc.err != nil {
				req.Error(err)
			} else {
				info, err := os.Stat(tc.dest)
				req.NoError(err)
				req.True(info.Size() > 0)
			}
		})
	}
}
