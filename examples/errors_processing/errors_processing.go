package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/greendwin/pipeline"
)

func main() {
	log.SetFlags(log.Ltime)

	ctx, cancel := pipeline.NewPipeline(context.Background())
	defer func() {
		log.Println("shutting down...")
		pipeline.Shutdown(ctx, cancel)
		log.Println("done.")
	}()

	seq := pipeline.Generate(ctx, func(w pipeline.Writer[int]) {
		for k := range 100 {
			_ = w.Write(k) // on shutdown it would not stuck
		}
	})

	tr, cherr := pipeline.TransformErr(ctx, 5, seq, func(x int) (v string, err error) {
		time.Sleep(100 * time.Millisecond)
		if x == 42 {
			err = fmt.Errorf("found strange number: %d", x)
			return
		}
		v = fmt.Sprintf("%d -> %d", x, x|17)
		return
	})

	finished := pipeline.Process(ctx, 2, tr, func(v string) {
		time.Sleep(100 * time.Millisecond)
		log.Println(v)
	})

	select {
	case err := <-cherr:
		log.Printf("error: %v", err)
	case <-finished:
		log.Println("finished!")
	}
}
