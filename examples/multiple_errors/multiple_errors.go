package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/greendwin/pipeline"
)

var suffixes = [...]string{
	"/v1",
	"/v1/foo",
	"/v1/foo/%d",
	"/v1/bar",
	"/v1/bar/%d",
}

func collect(ctx context.Context, baseUrl string) func(pipeline.Writer[string]) error {
	_ = ctx // `ctx` would have been used in the real collection

	return func(wr pipeline.Writer[string]) error {
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
	}
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

func sumResults(ctx context.Context, stats <-chan statsData) func() (int, error) {
	return func() (results int, err error) {
		for {
			st, ok := pipeline.Read(ctx, stats)
			if !ok {
				return
			}

			if err = tryFail("collect ", st); err != nil {
				return
			}

			results += int(st)
		}
	}
}

func tryFail(context ...any) error {
	if rand.Float32() < 0.005 {
		return fmt.Errorf("failed: %s", fmt.Sprint(context...))
	}
	return nil
}

func main() {
	log.SetFlags(log.Ltime)

	ctx, cancel := pipeline.NewPipeline(context.Background())
	defer func() {
		log.Println("shutting down...")
		cancel()
		log.Println("shutdown finished")
	}()

	urls1, urlsErr1 := pipeline.GenerateErr(ctx, collect(ctx, "www.foo.com"))
	urls2, urlsErr2 := pipeline.GenerateErr(ctx, collect(ctx, "www.bar.com"))
	urls3, urlsErr3 := pipeline.GenerateErr(ctx, collect(ctx, "www.quz.com"))

	urls, urlsFanErr := pipeline.FanIn(ctx, urls1, urls2, urls3)
	urlsErr := pipeline.FirstErr(ctx, urlsErr1, urlsErr2, urlsErr3, urlsFanErr)

	cont, contErr := pipeline.TransformErr(ctx, 5, urls, download)
	stats, statsErr := pipeline.TransformErr(ctx, 2, cont, getStats)

	res, resErr := pipeline.CollectErr(ctx, sumResults(ctx, stats))

	v, err := pipeline.ReadErr(ctx, res, urlsErr, contErr, statsErr, resErr)
	if err != nil {
		log.Printf("**STOPPING**: %v", err)
		return
	}

	log.Printf("results: %d", v)
}
