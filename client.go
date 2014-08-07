package airplay

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/DHowett/go-plist"
)

// A PlayBackInfo is a playback information of playing content.
type PlayBackInfo struct {
	IsReadyToPlay bool    `plist:"readyToPlay"`
	Duration      float32 `plist:"duration"`
	Position      float32 `plist:"position"`
}

type Client struct {
	connection *connection
}

type SlideTransition string

const (
	SlideNone     SlideTransition = "None"
	SlideDissolve SlideTransition = "Dissolve"
	SlideLeft     SlideTransition = "SlideLeft"
	SlideRight    SlideTransition = "SlideRight"
)

func NewClient() (*Client, error) {
	devices := Devices()
	if len(devices) == 0 {
		return nil, errors.New("AirPlay devices not found")
	}

	return &Client{connection: newConnection(devices[0])}, nil
}

func NewClientHasDevice(device Device) (*Client, error) {
	// TODO validation of device
	return &Client{connection: newConnection(device)}, nil
}

func (c *Client) Play(url string) <-chan error {
	return c.PlayAt(url, 0.0)
}

func (c *Client) PlayAt(url string, position float32) <-chan error {
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

		interval := time.Tick(time.Second)

		for {
			info, err := c.GetPlayBackInfo()

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

func (c *Client) Stop() {
	c.connection.post("stop", nil)
}

func (c *Client) Scrub(position float64) {
	query := fmt.Sprintf("?position=%f", position)
	c.connection.post("scrub"+query, nil)
}

func (c *Client) Photo(path string) {
	c.PhotoWithSlide(path, SlideNone)
}

func (c *Client) PhotoWithSlide(path string, transition SlideTransition) {
	url, err := url.Parse(path)
	if err != nil {
		log.Fatal(err)
	}

	var image *bytes.Reader

	if url.Scheme == "http" || url.Scheme == "https" {
		image, err = c.remoteImageReader(path)
	} else {
		image, err = c.localImageReader(path)
	}
	if err != nil {
		log.Fatal(err)
	}

	header := requestHeader{
		"X-Apple-Transition": {string(transition)},
	}
	c.connection.postWithHeader("photo", image, header)
}

func (c *Client) GetPlayBackInfo() (*PlayBackInfo, error) {
	response, err := c.connection.get("playback-info")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	decoder := plist.NewDecoder(bytes.NewReader(body))
	info := &PlayBackInfo{}
	if err := decoder.Decode(info); err != nil {
		return nil, err
	}

	return info, nil
}

func (c *Client) waitForReadyToPlay() error {
	interval := time.Tick(time.Second)
	timeout := time.After(10 * time.Second)

	for {
		select {
		case <-timeout:
			return errors.New("timeout while waiting for ready to play")
		case <-interval:
			info, err := c.GetPlayBackInfo()

			if err != nil {
				return err
			}

			if info.IsReadyToPlay {
				return nil
			}
		}
	}
}

func (c *Client) localImageReader(path string) (*bytes.Reader, error) {
	fn, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fn.Close()

	b, err := ioutil.ReadAll(fn)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(b), nil
}

func (c *Client) remoteImageReader(url string) (*bytes.Reader, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(body), nil
}
