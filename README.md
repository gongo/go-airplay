go-airplay
==========

Go bindings for AirPlay client

## Requirements

- [github.com/armon/mdns](https://github.com/armon/mdns)

## Usage

### Videos

```go
import "github.com/gongo/go-airplay"

client := airplay.NewClient()
ch := client.Play("http://movie.example.com/go.mp4")

// Block until have played content to the end
<-ch
```

Specifying the start position:

```go
// Start from 42% of length of content.
client.PlayAt("http://movie.example.com/go.mp4", 0.42)
```

See [example/player](./example/player/main.go) :

### Devices

```go
devices := airplay.Devices()
```

See [example/devices](./example/devices/main.go) :

## LICENSE

[MIT License](./LICENSE.txt).
