package airplay

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

const (
	digestAuthUsername = "AirPlay"
	digestAuthRealm    = "AirPlay"
)

type connection struct {
	device       Device
	passwordHash string
}

func newConnection(device Device) *connection {
	return &connection{device: device}
}

func (c *connection) setPassword(password string) {
	c.passwordHash = fmt.Sprintf("%x", md5.Sum([]byte(digestAuthUsername+":"+digestAuthRealm+":"+password)))
}

func (c *connection) get(path string) (*http.Response, error) {
	return c.getWithHeader(path, http.Header{})
}

func (c *connection) getWithHeader(path string, header http.Header) (*http.Response, error) {
	return c.request("GET", path, nil, header)
}

func (c *connection) post(path string, body io.ReadSeeker) (*http.Response, error) {
	return c.postWithHeader(path, body, http.Header{})
}

func (c *connection) postWithHeader(path string, body io.ReadSeeker, header http.Header) (*http.Response, error) {
	return c.request("POST", path, body, header)
}

func (c *connection) request(method, path string, body io.ReadSeeker, header http.Header) (*http.Response, error) {
	response, err := c.do(method, path, body, header)
	if err != nil {
		return nil, err
	}

	if response.StatusCode == http.StatusUnauthorized {
		if c.passwordHash == "" {
			msg := fmt.Sprintf(
				"airplay: [ERR] Device %s:%d is required password",
				c.device.Addr,
				c.device.Port,
			)
			return nil, errors.New(msg)
		}

		token := c.authorizationHeader(response, method, path, header)

		// body is closed first c.do().
		body.Seek(0, 0)
		header.Add("Authorization", token)
		response, err = c.do(method, path, body, header)
		if err != nil {
			return nil, err
		}

		if response.StatusCode == http.StatusUnauthorized {
			msg := fmt.Sprintf(
				"airplay: [ERR] Wrong password to %s:%d",
				c.device.Addr,
				c.device.Port,
			)
			return nil, errors.New(msg)
		}
	}

	return response, nil
}

func (c *connection) do(method, path string, body io.ReadSeeker, header http.Header) (*http.Response, error) {
	req, err := http.NewRequest(method, c.endpoint()+path, body)
	if err != nil {
		return nil, err
	}

	req.Header = header
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (c *connection) endpoint() string {
	return fmt.Sprintf("http://%s:%d/", c.device.Addr, c.device.Port)
}

func (c *connection) authorizationHeader(response *http.Response, method, path string, header http.Header) string {
	pattern := regexp.MustCompile("^Digest .*nonce=\"([^\"]+)\"")
	results := pattern.FindStringSubmatch(response.Header.Get("Www-Authenticate"))
	if results == nil {
		return ""
	}

	nonce := results[1]
	a1 := c.passwordHash
	a2 := fmt.Sprintf("%x", md5.Sum([]byte(method+":/"+path)))
	resp := fmt.Sprintf("%x", md5.Sum([]byte(a1+":"+nonce+":"+a2)))

	return fmt.Sprintf(
		"Digest username=\"%s\", realm=\"%s\", uri=\"/%s\", nonce=\"%s\", response=\"%s\"",
		digestAuthUsername,
		digestAuthRealm,
		path,
		nonce,
		resp,
	)
}
