Roadmap
=======

- [ ] return `ctx.Error()` on `ctx.Done()`
- [ ] TBD: non-failing pipelines still can fail (due to timeout)

- [ ] need `ReadErr` analog for `First`
- [ ] `ctx` can be cancelled by timeout, methods like `First` need predicted behavior (on `err := <-First(ctx, ...)` we should always check `ok`)
- [ ] TBD: add `FirstErr` that returns `ErrCancelled`?