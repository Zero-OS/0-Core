package storage

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type aydoStor struct {
	baseURL string
	client  *http.Client
}

func NewAydoStorage(u *url.URL) (Storage, error) {
	return &aydoStor{
		client:  &http.Client{},
		baseURL: strings.TrimRight(u.String(), "/"),
	}, nil
}

func (s *aydoStor) Get(hash string) (io.ReadCloser, error) {
	u := fmt.Sprintf("%s/%s", s.baseURL, hash)
	log.Infof("Downloading: %s", u)

	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Accept", "application/brotli")

	response, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		defer response.Body.Close()
		body, _ := ioutil.ReadAll(response.Body)
		return nil, fmt.Errorf("invalid response from stor (%d): %s", response.StatusCode, body)
	}

	return response.Body, nil
}
