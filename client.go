package airplay

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
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

func NewClient() (*Client, error) {
	connection, err := newConnection()
	if err != nil {
		return nil, err
	}
	return &Client{connection: connection}, nil
}

func (c *Client) Play(url string) <-chan error {
	return c.PlayAt(url, 0.0)
}

func (c *Client) PlayAt(url string, position float32) <-chan error {
	ch := make(chan error, 1)
	body := fmt.Sprintf("Content-Location: %s\nStart-Position: %f\n", url, position)

	go func() {
		if _, err := c.connection.post("play", "text/parameters", body); err != nil {
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
	c.connection.post("stop", "", "")
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
