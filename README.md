go-airplay
==========

Go bindings for AirPlay client

## Requirements

- [github.com/armon/mdns](https://github.com/armon/mdns)

## Usage

### Videos

```go
import "github.com/gongo/go-airplay"

client, err := airplay.NewClient()
if err != nil {
	log.Fatal(err)
}

ch := client.Play("http://movie.example.com/go.mp4")

// Block until have played content to the end
<-ch
```

Specifying the start position:

```go
// Start from 42% of length of content.
client.PlayAt("http://movie.example.com/go.mp4", 0.42)
```

Seek specify seconds:

```go
// Seek to 120 seconds from start position.
client.Scrub(120.0)
```

See:

- [example/player](./example/player/main.go)
- [example/seeker](./example/seeker/main.go)

### Images

```go
// local file
client.Photo("/path/to/gopher.jpg")

// remote file
client.Photo("http://blog.golang.org/gopher/plush.jpg")
```

You can specify the transition want to slide:

```go
client.Photo("/path/to/gopher.jpg", airplay.SlideNone) // eq client.Photo("..")
client.Photo("/path/to/gopher.jpg", airplay.SlideDissolve)
client.Photo("/path/to/gopher.jpg", airplay.SlideRight)
client.Photo("/path/to/gopher.jpg", airplay.SlideLeft)
```

See [example/slideshow](./example/slideshow/main.go) :

### Devices

```go
devices := airplay.Devices()
```

See [example/devices](./example/devices/main.go) :

## LICENSE

[MIT License](./LICENSE.txt).
