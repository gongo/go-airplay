package airplay

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

type testHundelrFunc func(*testing.T, http.ResponseWriter, *http.Request)

type testExpectRequest struct {
	method string
	path   string
}

func (e testExpectRequest) isMatch(method, path string) bool {
	return (e.method == method && e.path == path)
}

var (
	stopPlaybackInfo = `
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>readyToPlay</key>
	<false/>
	<key>uuid</key>
	<string>AAAAA-BBBBB-CCCCC-DDDDD-EEEEE</string>
</dict>
</plist>`

	playingPlaybackInfo = `
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>duration</key>
	<real>36.00000</real>
	<key>position</key>
	<real>18.00000</real>
	<key>readyToPlay</key>
	<true/>
</dict>
</plist>`
)

func TestPost(t *testing.T) {
	expectRequests := []testExpectRequest{
		{"POST", "/play"},
		{"GET", "/playback-info"},
		{"GET", "/playback-info"},
		{"GET", "/playback-info"},
	}
	responseXMLs := []string{
		stopPlaybackInfo,
		playingPlaybackInfo,
		stopPlaybackInfo,
	}

	ts := airTestServer(t, expectRequests, func(t *testing.T, w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/play" {
			bytes, err := ioutil.ReadAll(req.Body)
			if err != nil {
				t.Fatal(err)
			}

			result, _ := regexp.Match("Content-Location: http://movie.example.com/go.mp4", bytes)
			if !result {
				t.Fatalf("Incorrect request location (actual %s)", string(bytes))
			}

			result, _ = regexp.Match("Start-Position: 0.0", bytes)
			if !result {
				t.Fatalf("Incorrect request position (actual %s)", string(bytes))
			}
		}

		if req.URL.Path == "/playback-info" {
			xml := responseXMLs[0]
			responseXMLs = responseXMLs[1:]
			w.Write([]byte(xml))
		}
	})

	client := getTestClient(t, ts)
	ch := client.Play("http://movie.example.com/go.mp4")
	<-ch
}

func TestPostAt(t *testing.T) {
	expectRequests := []testExpectRequest{
		{"POST", "/play"},
		{"GET", "/playback-info"},
		{"GET", "/playback-info"},
	}
	responseXMLs := []string{
		playingPlaybackInfo,
		stopPlaybackInfo,
	}

	ts := airTestServer(t, expectRequests, func(t *testing.T, w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/play" {
			bytes, err := ioutil.ReadAll(req.Body)
			if err != nil {
				t.Fatal(err)
			}

			result, _ := regexp.Match("Start-Position: 12.3", bytes)
			if !result {
				t.Fatalf("Incorrect request position (actual %s)", string(bytes))
			}
		}

		if req.URL.Path == "/playback-info" {
			xml := responseXMLs[0]
			responseXMLs = responseXMLs[1:]
			w.Write([]byte(xml))
		}
	})

	client := getTestClient(t, ts)
	ch := client.PlayAt("http://movie.example.com/go.mp4", 12.3)
	<-ch
}

func TestStop(t *testing.T) {
	ts := airTestServer(t, []testExpectRequest{{"POST", "/stop"}}, nil)
	client := getTestClient(t, ts)
	client.Stop()
}

func TestScrub(t *testing.T) {
	position := 12.345
	ts := airTestServer(t, []testExpectRequest{{"POST", "/scrub"}}, func(t *testing.T, w http.ResponseWriter, req *http.Request) {
		values := req.URL.Query()
		positionString := values.Get("position")
		if positionString == "" {
			t.Fatal("Not found query parameter `position`")
		}

		positionFloat, err := strconv.ParseFloat(positionString, 64)
		if err != nil {
			t.Fatal("Incorrect query parameter `position` (actual = %s)", positionString)
		}

		if positionFloat != position {
			t.Fatalf("Incorrect query parameter `position` (actual = %f)", positionFloat)
		}
	})
	client := getTestClient(t, ts)
	client.Scrub(position)
}

func TestRate(t *testing.T) {
	rate := 0.8
	ts := airTestServer(t, []testExpectRequest{{"POST", "/rate"}}, func(t *testing.T, w http.ResponseWriter, req *http.Request) {
		rateString := req.URL.Query().Get("value")
		if rateString == "" {
			t.Fatal("Not found query parameter `value`")
		}

		rateFloat, err := strconv.ParseFloat(rateString, 64)
		if err != nil {
			t.Fatal("Incorrect query parameter `value` (actual = %s)", rateString)
		}

		if rateFloat != rate {
			t.Fatalf("Incorrect query parameter `value` (actual = %f)", rateFloat)
		}
	})
	client := getTestClient(t, ts)
	client.Rate(rate)
}

func TestPhotoLocalFile(t *testing.T) {
	dir := os.TempDir()

	f, err := ioutil.TempFile(dir, "photo_test")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	f.WriteString("localfile")

	ts := airTestServer(t, []testExpectRequest{{"POST", "/photo"}}, func(t *testing.T, w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("X-Apple-Transition") != "None" {
			t.Fatalf("Incorrect request header (actual = %s)", req.Header.Get("X-Apple-Transition"))
		}

		bytes, err := ioutil.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}

		body := string(bytes)
		if body != "localfile" {
			t.Fatalf("Incorrect request body (actual = %s)", body)
		}
	})

	client := getTestClient(t, ts)
	client.Photo(f.Name())
}

func TestPhotoRemoteFile(t *testing.T) {
	remoteTs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body := []byte("remotefile")
		w.Write(body)
	}))

	ts := airTestServer(t, []testExpectRequest{{"POST", "/photo"}}, func(t *testing.T, w http.ResponseWriter, req *http.Request) {
		bytes, err := ioutil.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}

		body := string(bytes)
		if body != "remotefile" {
			t.Fatalf("Incorrect request body (actual = %s)", body)
		}
	})

	client := getTestClient(t, ts)
	client.Photo(remoteTs.URL)
}

func TestPhotoWithSlide(t *testing.T) {
	remoteTs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body := []byte("remotefile")
		w.Write(body)
	}))

	ts := airTestServer(t, []testExpectRequest{{"POST", "/photo"}}, func(t *testing.T, w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("X-Apple-Transition") != "SlideRight" {
			t.Fatalf("Incorrect request header (actual = %s)", req.Header.Get("X-Apple-Transition"))
		}
	})

	client := getTestClient(t, ts)
	client.PhotoWithSlide(remoteTs.URL, SlideRight)
}

func TestGetPlaybackInfo(t *testing.T) {
	expectRequests := []testExpectRequest{
		{"GET", "/playback-info"},
		{"GET", "/playback-info"},
	}
	responseXMLs := []string{stopPlaybackInfo, playingPlaybackInfo}

	ts := airTestServer(t, expectRequests, func(t *testing.T, w http.ResponseWriter, req *http.Request) {
		xml := responseXMLs[0]
		responseXMLs = responseXMLs[1:]
		w.Write([]byte(xml))
	})

	client := getTestClient(t, ts)

	info, err := client.GetPlaybackInfo()
	if err != nil {
		t.Fatal(err)
	}

	if info.IsReadyToPlay {
		t.Fatal("PlaybackInfo is not ready to play status")
	}

	info, err = client.GetPlaybackInfo()
	if err != nil {
		t.Fatal(err)
	}

	if info.Duration != 36.0 || info.Position != 18.0 {
		t.Fatal("Incorrect PlaybackInfo")
	}
}

func airTestServer(t *testing.T, requests []testExpectRequest, handler testHundelrFunc) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if len(requests) == 0 {
			t.Fatal("Incorrect request count")
		}

		expect := requests[0]
		requests = requests[1:]

		if !expect.isMatch(req.Method, req.URL.Path) {
			t.Fatalf("request is not '%s %s' (actual = %s %s)",
				expect.method, expect.path, req.Method, req.URL.Path)
		}

		if handler != nil {
			handler(t, w, req)
		}
	}))
}

func getTestClient(t *testing.T, ts *httptest.Server) *Client {
	addr, port := getAddrAndPort(t, ts.URL)
	device := Device{
		Addr: addr,
		Port: port,
	}
	client, err := NewClientHasDevice(device)
	if err != nil {
		t.Fatal(err)
	}

	return client
}

func getAddrAndPort(t *testing.T, host string) (string, int) {
	u, err := url.Parse(host)
	if err != nil {
		t.Fatal(err)
	}

	split := strings.Split(u.Host, ":")
	addr := split[0]
	port, err := strconv.Atoi(split[1])
	if err != nil {
		t.Fatal(err)
	}

	return addr, port
}
