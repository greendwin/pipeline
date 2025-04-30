Roadmap
=======

TBD: non-failing pipelines still can fail (due to timeout)
return `context.Cause(ctx)` on `ctx.Done()`:
- [x] `First`
- [ ] `FanIn`
- [ ] ...?

- [ ] need `ReadErr` analog for `First`
- [ ] `ctx` can be cancelled by timeout, methods like `First` need predicted behavior (on `err := <-First(ctx, ...)` we should always check `ok`)
- [ ] TBD: add `FirstErr` that returns `ErrCancelled`?