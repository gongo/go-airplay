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

	"time"

	"github.com/gongo/text-parameters"
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

	playingPlaybackInfoAt4G = `
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>readyToPlay</key>
	<integer>1</integer>
</dict>
</plist>`
)

type playbackInfoParam struct {
	Location string  `parameters:"Content-Location"`
	Position float64 `parameters:"Start-Position"`
}

func TestMain(m *testing.M) {
	requestInverval = time.Millisecond
	os.Exit(m.Run())
}

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
			u := &playbackInfoParam{}
			decoder := parameters.NewDecorder(req.Body)
			decoder.Decode(u)

			if u.Location != "http://movie.example.com/go.mp4" {
				t.Fatalf("Incorrect request location (actual %s)", u.Location)
			}

			if u.Position != 0.0 {
				t.Fatalf("Incorrect request position (actual %f)", u.Position)
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
			u := &playbackInfoParam{}
			decoder := parameters.NewDecorder(req.Body)
			decoder.Decode(u)

			if u.Position != 12.3 {
				t.Fatalf("Incorrect request position (actual %f)", u.Position)
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
			t.Fatalf("Incorrect query parameter `position` (actual = %s)", positionString)
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
			t.Fatalf("Incorrect query parameter `value` (actual = %s)", rateString)
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

func TestGetPlaybackInfoWithVariousVersion(t *testing.T) {
	expectRequests := []testExpectRequest{
		{"GET", "/playback-info"},
		{"GET", "/playback-info"},
	}
	responseXMLs := []string{playingPlaybackInfo, playingPlaybackInfoAt4G}

	ts := airTestServer(t, expectRequests, func(t *testing.T, w http.ResponseWriter, req *http.Request) {
		xml := responseXMLs[0]
		responseXMLs = responseXMLs[1:]
		w.Write([]byte(xml))
	})

	client := getTestClient(t, ts)

	for range responseXMLs {
		info, err := client.GetPlaybackInfo()
		if err != nil {
			t.Fatal(err)
		}

		if !info.IsReadyToPlay {
			t.Fatal("PlaybackInfo is not ready to play status")
		}
	}
}

func TestClientToPasswordRequiredDevice(t *testing.T) {
	expectRequests := []testExpectRequest{
		{"POST", "/play"},
		{"POST", "/play"},
		{"GET", "/playback-info"},
		{"GET", "/playback-info"},
	}
	responseXMLs := []string{
		playingPlaybackInfo,
		stopPlaybackInfo,
	}
	playRequestCount := 1

	ts := airTestServer(t, expectRequests, func(t *testing.T, w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/play":
			switch playRequestCount {
			case 1:
				w.Header().Add("WWW-Authenticate", "Digest realm=\"AirPlay\", nonce=\"4444\"")
				w.WriteHeader(http.StatusUnauthorized)
			case 2:
				pattern := regexp.MustCompile("^Digest .*response=\"([^\"]+)\"")
				results := pattern.FindStringSubmatch(req.Header.Get("Authorization"))
				if results == nil {
					t.Fatalf("Unexpected request: %s", req.Header.Get("Authorization"))
				}

				// response was created on the assumption that password is "gongo"
				expect := "f53f5ad052f58cee48f550b9632e0446"
				actual := results[1]
				if expect != actual {
					t.Fatalf("Incorrect authorization header (actual %s)", actual)
				}
			default:
				t.Fatal("Surplus request has occurs")
			}

			playRequestCount++
		case "/playback-info":
			xml := responseXMLs[0]
			responseXMLs = responseXMLs[1:]
			w.Write([]byte(xml))
		}
	})

	client := getTestClient(t, ts)
	client.SetPassword("gongo")
	ch := client.Play("http://movie.example.com/go.mp4")
	if err := <-ch; err != nil {
		t.Fatal(err)
	}
}

func TestClientWithErrorAboutPassword(t *testing.T) {
	expectRequests := []testExpectRequest{
		{"POST", "/play"},
		{"POST", "/play"},
		{"POST", "/play"},
	}
	playRequestCount := 1

	ts := airTestServer(t, expectRequests, func(t *testing.T, w http.ResponseWriter, req *http.Request) {
		switch playRequestCount {
		case 1, 2:
			w.Header().Add("WWW-Authenticate", "Digest realm=\"AirPlay\", nonce=\"4444\"")
			w.WriteHeader(http.StatusUnauthorized)
		case 3:
			pattern := regexp.MustCompile("^Digest .*response=\"([^\"]+)\"")
			results := pattern.FindStringSubmatch(req.Header.Get("Authorization"))
			if results == nil {
				t.Fatalf("Unexpected request: %s", req.Header.Get("Authorization"))
			}

			// response was created on the assumption that password is "gongo"
			expect := "f53f5ad052f58cee48f550b9632e0446"
			actual := results[1]
			if expect != actual {
				w.Header().Add("WWW-Authenticate", "Digest realm=\"AirPlay\", nonce=\"4444\"")
				w.WriteHeader(http.StatusUnauthorized)
			}
		default:
			t.Fatal("Surplus request has occurs")
		}

		playRequestCount++
	})

	var ch <-chan error

	client := getTestClient(t, ts)
	ch = client.Play("http://movie.example.com/go.mp4")
	if err := <-ch; err == nil {
		t.Fatal("It should occurs [password required] error")
	}

	client.SetPassword("wrongpassword")
	ch = client.Play("http://movie.example.com/go.mp4")
	if err := <-ch; err == nil {
		t.Fatal("It should occurs [wrong password] error")
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
	client, err := NewClient(&ClientParam{Addr: addr, Port: port})
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
