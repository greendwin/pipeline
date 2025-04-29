Pipeline
========

A concurrent computations library.

This project follows the patterns described in the [Pipelines and cancellation
](https://go.dev/blog/pipelines) blog post.

It uses `context` to cancel computations when errors occur. Additionally, it keeps track of spawned goroutines so you can ensure all resources are cleanly closed when the pipeline finishes.

## Examples

The repository contains several examples:
* `basic_transform`: A simple pipeline creation with cancellation.
* `errors_processing`: Handling errors in a pipeline.
* `multiple_errors`: A more complex example with multple stages that may raise errors.


## Functions

### NewPipeline

`NewPipeline` ensures that all goroutines in the pipeline exit when it cancelled.

```go
ctx, cancel := pipeline.NewPipeline(context.Background())

// run `cancel` and wait all spawned goroutines to finish
defer pipeline.Shutdown(ctx, cancel)
```

It is optional, so you can create a pipeline without it if goroutine tracking is not needed.


### Generate

Generator is the starting point of any pipeline. The `Generate` function creates a channel for writing generated values.

The `Write` method of the `pipeline.Writer[T]` type returns `false` if the context is cancelled.

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

`Transform` creates multiple workers and processes the input channel in parallel.

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

`Collect` can be used to gather the final results. It takes a function that returns a result and returns a oneshot channel to wait for the final result in the next step.

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

Each pipeline function has a fallable version that returns an oneshot error channel in addition to the result.

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
