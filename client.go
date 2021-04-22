package airplay

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"howett.net/plist"
)

// A PlaybackInfo is a playback information of playing content.
type PlaybackInfo struct {
	// IsReadyToPlay, if true, content is currently playing or ready to play.
	IsReadyToPlay bool `plist:"readyToPlay"`

	// ReadyToPlayValue represents the information on whether content is currently playing, ready to play or not.
	ReadyToPlayValue interface{} `plist:"readyToPlay"`

	// Duration represents playback duration in seconds.
	Duration float64 `plist:"duration"`

	// Position represents playback position in seconds.
	Position float64 `plist:"position"`
}

type Client struct {
	connection *connection
}

// SlideTransition represents transition that used when show the picture.
type SlideTransition string

const (
	SlideNone     SlideTransition = "None"
	SlideDissolve SlideTransition = "Dissolve"
	SlideLeft     SlideTransition = "SlideLeft"
	SlideRight    SlideTransition = "SlideRight"
)

var (
	requestInverval = time.Second
)

type ClientParam struct {
	Addr     string
	Port     int
	Password string
}

// FirstClient return the AirPlay Client that has the first found AirPlay device in LAN
func FirstClient() (*Client, error) {
	device := FirstDevice()
	if device.Name == "" {
		return nil, errors.New("AirPlay devices not found")
	}

	return &Client{connection: newConnection(device)}, nil
}

func NewClient(params *ClientParam) (*Client, error) {
	if params.Addr == "" {
		return nil, errors.New("airplay: [ERR] Address is required to NewClient()")
	}

	if params.Port <= 0 {
		params.Port = 7000
	}

	client := &Client{}
	device := Device{Addr: params.Addr, Port: params.Port}
	client.connection = newConnection(device)

	if params.Password != "" {
		client.SetPassword(params.Password)
	}

	return client, nil
}

func (c Client) SetPassword(password string) {
	c.connection.setPassword(password)
}

// Play start content playback.
//
// When playback is finished, sends termination status on the returned channel.
// If non-nil, not a successful termination.
func (c *Client) Play(url string) <-chan error {
	return c.PlayAt(url, 0.0)
}

// PlayAt start content playback by specifying the start position.
//
// Returned channel is the same as Play().
func (c *Client) PlayAt(url string, position float64) <-chan error {
	ch := make(chan error, 1)
	body := fmt.Sprintf("Content-Location: %s\nStart-Position: %f\n", url, position)

	go func() {
		if _, err := c.connection.post("play", strings.NewReader(body)); err != nil {
			ch <- err
			return
		}

		if err := c.waitForReadyToPlay(); err != nil {
			ch <- err
			return
		}

		interval := time.Tick(requestInverval)

		for {
			info, err := c.GetPlaybackInfo()

			if err != nil {
				ch <- err
				return
			}

			if !info.IsReadyToPlay {
				break
			}

			<-interval
		}

		ch <- nil
	}()

	return ch
}

// Stop exits content playback.
func (c *Client) Stop() {
	c.connection.post("stop", nil)
}

// Scrub seeks at position seconds in playing content.
func (c *Client) Scrub(position float64) {
	query := fmt.Sprintf("?position=%f", position)
	c.connection.post("scrub"+query, nil)
}

// Rate change the playback rate in playing content.
//
// If rate is 0, content is paused.
// if rate is 1, content playing at the normal speed.
func (c *Client) Rate(rate float64) {
	query := fmt.Sprintf("?value=%f", rate)
	c.connection.post("rate"+query, nil)
}

// Photo show a JPEG picture. It can specify both remote or local file.
//
// A trivial example:
//
//     // local file
//     client.Photo("/path/to/gopher.jpg")
//
//     // remote file
//     client.Photo("http://blog.golang.org/gopher/plush.jpg")
//
func (c *Client) Photo(path string) {
	c.PhotoWithSlide(path, SlideNone)
}

// PhotoWithSlide show a JPEG picture in the transition specified.
func (c *Client) PhotoWithSlide(path string, transition SlideTransition) {
	url, err := url.Parse(path)
	if err != nil {
		log.Fatal(err)
	}

	var image *bytes.Reader

	if url.Scheme == "http" || url.Scheme == "https" {
		image, err = remoteImageReader(path)
	} else {
		image, err = localImageReader(path)
	}
	if err != nil {
		log.Fatal(err)
	}

	header := http.Header{
		"X-Apple-Transition": {string(transition)},
	}
	c.connection.postWithHeader("photo", image, header)
}

// GetPlaybackInfo retrieves playback informations.
func (c *Client) GetPlaybackInfo() (*PlaybackInfo, error) {
	response, err := c.connection.get("playback-info")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := convertBytesReader(response.Body)
	if err != nil {
		return nil, err
	}

	decoder := plist.NewDecoder(body)
	info := &PlaybackInfo{}
	if err := decoder.Decode(info); err != nil {
		return nil, err
	}

	switch t := info.ReadyToPlayValue.(type) {
	case uint64: // AppleTV 4G
		info.IsReadyToPlay = (t == 1)
	case bool: // AppleTV 2G, 3G
		info.IsReadyToPlay = t
	}

	return info, nil
}

func (c *Client) waitForReadyToPlay() error {
	interval := time.Tick(requestInverval)
	timeout := time.After(10 * time.Second)

	for {
		select {
		case <-timeout:
			return errors.New("timeout while waiting for ready to play")
		case <-interval:
			info, err := c.GetPlaybackInfo()

			if err != nil {
				return err
			}

			if info.IsReadyToPlay {
				return nil
			}
		}
	}
}

func localImageReader(path string) (*bytes.Reader, error) {
	fn, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fn.Close()

	return convertBytesReader(fn)
}

func remoteImageReader(url string) (*bytes.Reader, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	return convertBytesReader(response.Body)
}

func convertBytesReader(r io.Reader) (*bytes.Reader, error) {
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(body), nil
}
