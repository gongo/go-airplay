package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/gongo/go-airplay"
)

var opts struct {
	mediaURL         string
	startingPosition float64
	playTimeout      int
	showHelpFlag     bool
}

func init() {
	flag.StringVar(&opts.mediaURL, "i", "", "Input media URL")
	flag.Float64Var(&opts.startingPosition, "s", 0.0, "Starting position between 0 (0%) to 1 (100%)")
	flag.IntVar(&opts.playTimeout, "t", -1, "Timeout for play to end (sec)")
	flag.BoolVar(&opts.showHelpFlag, "h", false, "Show this message")
	flag.Parse()

	if opts.showHelpFlag {
		flag.Usage()
		os.Exit(0)
	}

	if opts.mediaURL == "" {
		log.Fatal("options: Missing media URL")
	}

	if opts.startingPosition < 0 || opts.startingPosition > 1 {
		log.Fatal("options: Starting position should between 0 to 1")
	}
}

func airplayClient() *airplay.Client {
	client, err := airplay.FirstClient()
	if err != nil {
		log.Fatal(err)
	}
	return client
}

func playToEnd() {
	client := airplayClient()
	ch := client.PlayAt(opts.mediaURL, opts.startingPosition)
	<-ch
}

func playUntilTimeoutOrEnd() {
	client := airplayClient()
	ch := client.PlayAt(opts.mediaURL, opts.startingPosition)
	timeout := time.After(time.Duration(opts.playTimeout) * time.Second)

	select {
	case <-timeout:
		client.Stop()
	case <-ch:
	}
}

func main() {
	if opts.playTimeout > 0 {
		playUntilTimeoutOrEnd()
	} else {
		playToEnd()
	}
}
