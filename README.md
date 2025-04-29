Pipeline
========

Concurrent computations library

This is an educational project that follows the patterns described in [Pipelines and cancellation
](https://go.dev/blog/pipelines).

It uses `context` to cancel computations in case of errors. Additionally, it tracks spawned goroutines, so you can be sure that all resources are cleaned up when the pipeline shuts down.

## Examples

There are several examples in the repository:
* `basic_transform` shows how to create a simple computation pipeline creation with proper cancellation
* `errors_processing` demonstrates how to handle errors
* `multiple_errors` shows a more complex example, where each stage may raise an error

## Functions

### NewPipeline

Function `NewPipeline` is useful when you want to ensure that all goroutines exit when the pipeline is cancelled.

```go
ctx, cancel := pipeline.NewPipeline(context.Background())

// run `cancel` and wait all spawned goroutines to finish
defer pipeline.Shutdown(ctx, cancel)
```

Function `NewPipeline` is optional. You can use all pipeline creation methods without it if you don't need goroutine tracking.


### Generate

Generator is the starting point of any pipeline. The `Generate` function creates a channel for writing generated values.

The method `pipeline.Writer[T].Write` returns `false` if the context is cancelled.

```go
squares := pipeline.Generate(ctx, func (wr pipeline.Writer[int]) {
    k := 0
    for {
        if !wr.Write(k * k) {
            // `ctx` was cancelled
            return
        }

        k += 1
    }
})
```


### Transform

The `Transform` function spawns multiple workers and processes input channel in parallel.

Results are unordered.

```go
nums := sequence(ctx, 100)

// spawn 4 workers to process generated nums
percents := pipeline.Transform(ctx, 4, nums, func (v int) float32 {
    return v / 100.0
})

// results are chained to a channel (i.e. fan-out - fan-in)
// so we can combine multiple transform steps
for v := range percents {
    fmt.Println(v)
}
```


### Collect

Function `Collect` can be used to collect the final results. It accepts a function that returns a result, and returns an oneshot channel so that the final result can be awaited in the next step.

```go
result := pipeline.Collect(ctx, func () (sum float32) {
    for {
        v, ok := pipeline.Read(ctx, percents)
        if !ok {
            return sum
        }

        sum += baseCost * v
    }
})

sum, ok := pipeline.Read(ctx, result)
if ok {
    log.Println("final sum = ", sum)
} else {
    log.Println("timeout!")
}
```


### Error handling

Each processing function has a fallable version that returns an oneshot error channel in addition to the results.

```go
contents, contErr := pipeline.TransformErr(ctx, 16, urls, func (url string) ([]byte, error) {
    // request data...
})

// process content, e.g. saving it to disk
finished, procErr := pipeline.Process(ctx, 4, contenst, func(cont []byte) error {
    // store `cont` to disk...
})

select {
case err := <-pipeline.First(ctx, contErr, procErr):
    log.Println(err)
case <-finished:
    log.Println("done.")
}
```
