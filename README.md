# Co

A concurrency lib project with GENERIC SUPPORTED dedicate to help developer ease the pain of dealing goroutine and channel when coding 2 more lines of channel code become annoying.

`Co` provides a lot of (at least in planning) helper functions and utils such as `AwaitAll` `AwaitAny` and Parallel execution mechanism with concurrency limitation.

If there are some concurrency patterns you would like me to implement and there is none available in community or in go lib, you are more than welcome to open an issue.

## Listence

MIT

## Doc

https://godoc.org/github.com/tempura-shrimp/co

## Examples

### Parallel

```golang
// replace with `co.NewParallel` if no response needed
p := co.NewParallelWithResponse(10) // worker size
for i := 0; i < 10000; i++ {
    i := i
    p.AddWithResponse[int](func() int {
        return i + 1
    })
}

// Wait doesn't indicate a Run, the job will run once added
// So, you could ignore Wait() in some cases
vals := p.Wait()
```

### Awaits

```golang

handlers := make([]func() int, 0)
for i := 0; i < 1000; i++ {
    i := i
    handlers = append(handlers, func() int {
        return i + 1
    })
}

responses := co.AwaitAll[int](handlers...)
```

## Benchmark

```
goos: darwin
goarch: amd64
pkg: github.com/tempura-shrimp/co
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkAwaitAll-12               10000            184177 ns/op
BenchmarkParallel-12               10000            185064 ns/op
BenchmarkWithoutParallel-12        10000            982347 ns/op
PASS
```
