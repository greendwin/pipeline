package main

import (
	"fmt"
	"log"
	"time"

	pl "github.com/greendwin/pipeline"
)

func main() {
	log.SetFlags(log.Ltime)

	pp := pl.NewPipeline()
	defer func() {
		log.Println("shutting down...")
		pp.Shutdown()
		log.Println("done.")
	}()

	seq := pl.Generate(pp, func(w pl.Writer[int]) {
		for k := range 100 {
			_ = w.Write(k) // on shutdown it would not stuck
		}
	})

	tr, cherr := pl.TransformErr(pp, 5, seq, func(x int) (v string, err error) {
		time.Sleep(100 * time.Millisecond)
		if x == 42 {
			err = fmt.Errorf("found strange number: %d", x)
			return
		}
		v = fmt.Sprintf("%d -> %d", x, x|17)
		return
	})

	finished := pl.Process(pp, 2, tr, func(v string) {
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
