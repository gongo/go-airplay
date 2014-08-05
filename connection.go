package airplay

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type connection struct {
	device   Device
	endpoint string
}

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
	return http.Get(c.endpoint + path)
}

func (c *connection) post(path, bodyType, body string) (*http.Response, error) {
	return http.Post(c.endpoint+path, bodyType, strings.NewReader(body))
}
