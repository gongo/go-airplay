package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/gongo/go-airplay"
)

var opts struct {
	imagePath    string
	showHelpFlag bool
}

func init() {
	flag.StringVar(&opts.imagePath, "i", "", "Input image path (local or remote)")
	flag.BoolVar(&opts.showHelpFlag, "h", false, "Show this message")
	flag.Parse()

	if opts.showHelpFlag {
		flag.Usage()
		os.Exit(0)
	}

	if opts.imagePath == "" {
		flag.Usage()
		log.Fatal("options: Missing image path")
	}
}

func main() {
	client, _ := airplay.NewClient()
	rand.Seed(time.Now().UnixNano())

	transitions := []airplay.SlideTransition{
		airplay.SlideRight,
		airplay.SlideLeft,
	}

	timeout := time.After(10 * time.Second)
	interval := time.Tick(time.Second)

	for {
		select {
		case <-timeout:
			return
		case <-interval:
			index := rand.Intn(len(transitions))
			client.PhotoWithSlide(opts.imagePath, transitions[index])
		}
	}
}
