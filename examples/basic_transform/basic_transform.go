package main

import (
	"log"
	"time"

	pl "github.com/greendwin/pipeline"
)

func sequence(pp *pl.Pipeline, count int) <-chan int {
	return pl.Generate(pp, func(wr pl.Writer[int]) {
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

func square(pp *pl.Pipeline, threads int, nums <-chan int) <-chan result {
	return pl.Transform(pp, threads, nums, func(x int) result {
		log.Printf("square(%d): started", x)
		time.Sleep(500 * time.Millisecond)
		log.Printf("square(%d): finished", x)
		return result{x, x * x}
	})
}

func printNums(pp *pl.Pipeline, in <-chan result) pl.Signal {
	return pl.Process(pp, 1, in, func(r result) {
		log.Printf("process: %d -> %d", r.x, r.y)
	})
}

func main() {
	log.SetFlags(log.Ltime)

	pp := pl.NewPipeline()

	nums := sequence(pp, 20)
	sq := square(pp, 4, nums)
	finished := printNums(pp, sq)

	if !finished.TryWait(2 * time.Second) {
		log.Println("timeout!")
	}

	log.Println("shutting down...")
	pp.Shutdown()
	log.Println("done.")
}
