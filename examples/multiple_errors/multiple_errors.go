package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	pl "github.com/greendwin/pipeline"
)

var suffixes = [...]string{
	"/v1",
	"/v1/foo",
	"/v1/foo/%d",
	"/v1/bar",
	"/v1/bar/%d",
}

func collect(pp *pl.Pipeline, baseUrl string) <-chan string {
	return pl.Generate(pp, func(wr pl.Writer[string]) {
		for _, suf := range suffixes {
			url := baseUrl + suf
			if !strings.HasSuffix(url, "%d") {
				if !wr.Write(url) {
					return
				}
				continue
			}

			for k := range 10 {
				if !wr.Write(fmt.Sprintf(url, k)) {
					return
				}
			}
		}
	})
}

type content string

func download(url string) content {
	time.Sleep(200 * time.Millisecond)
	log.Printf("download %q", url)
	return content(fmt.Sprintf("<content:%q>", url))
}

type statsData int

func getStats(cont content) statsData {
	time.Sleep(100 * time.Millisecond)
	log.Printf("stats %v -> %d", cont, len(cont))
	return statsData(len(cont))
}

func main() {
	log.SetFlags(log.Ltime)

	pp := pl.NewPipeline()
	defer func() {
		log.Println("shutting down...")
		pp.Shutdown()
		log.Println("shutdown finished")
	}()

	urls1 := collect(pp, "www.foo.com")
	urls2 := collect(pp, "www.bar.com")
	urls3 := collect(pp, "www.quz.com")

	urls := pl.FanIn(pp, urls1, urls2, urls3)
	cont := pl.Transform(pp, 5, urls, download)
	stats := pl.Transform(pp, 2, cont, getStats)

	res := pl.Collect(pp, func() (results int) {
		for st := range stats {
			results += int(st)
		}
		return
	})

	log.Printf("results: %d", <-res)
}

// TODO: add random error on each stage
