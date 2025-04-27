pipeline
========

Concurrent pipelines building library

## Description

This is educational library that follows patterns described in [a blog post](https://go.dev/blog/pipelines).

It allows you to build simple concurrent pipelines that:
* cleanup spawned goroutines and
* stop execution in case of an error

## Examples

### Pipeline

TODO: describe pipeline object and `Finish` method

```go
pp := pipeline.NewPipeline()
defer pp.Finish() // shutdown processing and cleanup resources
```


### Generator

Generators are starting point of any pipeline.

```go
sq := pipeline.Generate(pp, func (wr pl.Writer[int]) {
    for k := range 10 {
        // write can fail in case of no consumer
        if !wr.Write(k * k) {
            // in this case just stop generation
            return
        }
    }
})

// `pipeline` closes channel for us, so we can use range-for
for v := range sq {
    fmt.Println(v)
}
```

### Transform

Generated data can be faned out to worker goroutines and their result is chained to the next channel for further processing.

```go
nums := sequence(pp, 100)

// spawn 4 workers to process generated nums
percents := pipeline.Transform(pp, 4, nums, func (v int) float32 {
    return v / 100.0
})

// results are chained to a channel (i.e. fan-out - fan-in)
// so we can combine multiple transform steps
for v := range percents {
    fmt.Println(v)
}
```


### Process

TODO


### Error handling

TODO


### Full example