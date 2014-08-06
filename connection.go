package airplay

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

type connection struct {
	device   Device
	endpoint string
}

type requestHeader http.Header

func newConnection() (*connection, error) {
	devices := Devices()

	if len(devices) == 0 {
		return nil, errors.New("AirPlay devices not found")
	}

	device := devices[0]
	endpoint := fmt.Sprintf("http://%s:%d/", device.Addr, device.Port)

	return &connection{device: device, endpoint: endpoint}, nil
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
	req, err := http.NewRequest(method, c.endpoint+path, body)
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
