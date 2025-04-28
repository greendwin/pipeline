package main

import (
	"fmt"
	"log"
	"math/rand"
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

func collect(pp *pl.Pipeline, baseUrl string) (<-chan string, pl.Oneshot[error]) {
	return pl.GenerateErr(pp, func(wr pl.Writer[string]) error {
		for _, suf := range suffixes {
			url := baseUrl + suf
			if !strings.HasSuffix(url, "%d") {
				if !wr.Write(url) {
					return nil
				}
				continue
			}

			for k := range 10 {
				if err := tryFail("collecting ", baseUrl); err != nil {
					return err
				}

				if !wr.Write(fmt.Sprintf(url, k)) {
					return nil
				}
			}
		}

		return nil
	})
}

type content string

func download(url string) (res content, err error) {
	time.Sleep(200 * time.Millisecond)

	if err = tryFail("downloading ", url); err != nil {
		return
	}

	log.Printf("download %q", url)
	res = content(fmt.Sprintf("<content:%q>", url))
	return
}

type statsData int

func getStats(cont content) (data statsData, err error) {
	time.Sleep(100 * time.Millisecond)

	if err = tryFail("stats for ", cont); err != nil {
		return
	}

	log.Printf("stats %v -> %d", cont, len(cont))
	data = statsData(len(cont))
	return
}

func tryFail(context ...any) error {
	if rand.Float32() < 0.01 {
		return fmt.Errorf("failed: %s", fmt.Sprint(context...))
	}
	return nil
}

func main() {
	log.SetFlags(log.Ltime)

	pp := pl.NewPipeline()
	defer func() {
		log.Println("shutting down...")
		pp.Shutdown()
		log.Println("shutdown finished")
	}()

	urls1, urlsErr1 := collect(pp, "www.foo.com")
	urls2, urlsErr2 := collect(pp, "www.bar.com")
	urls3, urlsErr3 := collect(pp, "www.quz.com")

	urls := pl.FanIn(pp, urls1, urls2, urls3)
	urlsErr := pl.First(pp, urlsErr1, urlsErr2, urlsErr3)

	cont, contErr := pl.TransformErr(pp, 5, urls, download)
	stats, statsErr := pl.TransformErr(pp, 2, cont, getStats)

	res, resErr := pl.CollectErr(pp, func() (results int, err error) {
		for {
			st, ok := pl.Read(pp, stats)
			if !ok {
				return
			}

			if err = tryFail("collect ", st); err != nil {
				return
			}

			results += int(st)
		}
	})

	cherr := pl.First(pp, urlsErr, contErr, statsErr, resErr)

	select {
	case v := <-res:
		log.Printf("results: %d", v)
	case err := <-cherr:
		log.Printf("**stopping**: %v", err)
	}
}
