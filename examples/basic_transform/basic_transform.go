package main

import (
	"context"
	"log"
	"time"

	"github.com/greendwin/pipeline"
)

func sequence(ctx context.Context, count int) <-chan int {
	return pipeline.Generate(ctx, func(wr pipeline.Writer[int]) {
		for k := range count {
			if !wr.Write(k) {
				return
			}
		}
	})
}

type result struct {
	x, y int
}

func square(ctx context.Context, threads int, nums <-chan int) <-chan result {
	return pipeline.Transform(ctx, threads, nums, func(x int) result {
		log.Printf("square(%d): started", x)
		time.Sleep(500 * time.Millisecond)
		log.Printf("square(%d): finished", x)
		return result{x, x * x}
	})
}

func printNums(ctx context.Context, in <-chan result) pipeline.Signal {
	return pipeline.Process(ctx, 1, in, func(r result) {
		log.Printf("process: %d -> %d", r.x, r.y)
	})
}

func main() {
	log.SetFlags(log.Ltime)

	ctx, cancel := pipeline.NewPipeline(context.Background())

	nums := sequence(ctx, 20)
	sq := square(ctx, 4, nums)
	finished := printNums(ctx, sq)

	if !finished.TryWait(2 * time.Second) {
		log.Println("timeout!")
	}

	log.Println("shutting down...")
	cancel()
	log.Println("done.")
}
