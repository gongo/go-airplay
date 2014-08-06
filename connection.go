package airplay

import (
	"fmt"
	"io"
	"net/http"
)

type connection struct {
	device Device
}

type requestHeader http.Header

func newConnection(device Device) *connection {
	return &connection{device: device}
}

func (c *connection) get(path string) (*http.Response, error) {
	return c.getWithHeader(path, make(requestHeader))
}

func (c *connection) getWithHeader(path string, header requestHeader) (*http.Response, error) {
	return c.request("GET", path, nil, header)
}

func (c *connection) post(path string, body io.Reader) (*http.Response, error) {
	return c.postWithHeader(path, body, make(requestHeader))
}

func (c *connection) postWithHeader(path string, body io.Reader, header requestHeader) (*http.Response, error) {
	return c.request("POST", path, body, header)
}

func (c *connection) request(method, path string, body io.Reader, header requestHeader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.endpoint()+path, body)
	if err != nil {
		return nil, err
	}

	for key, values := range header {
		for _, vv := range values {
			req.Header.Add(key, vv)
		}
	}

	client := &http.Client{}
	return client.Do(req)
}

func (c *connection) endpoint() string {
	return fmt.Sprintf("http://%s:%d/", c.device.Addr, c.device.Port)
}
