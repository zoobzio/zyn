# Benchmarks

This directory contains performance benchmarks for zyn.

## Running Benchmarks

```bash
# All benchmarks
go test -v -bench=. -benchmem ./testing/benchmarks/...

# Specific benchmark
go test -v -bench=BenchmarkSynapse_Creation -benchmem ./testing/benchmarks/...

# Multiple runs for statistical significance
go test -v -bench=. -benchmem -count=5 ./testing/benchmarks/...

# With CPU profiling
go test -v -bench=. -benchmem -cpuprofile=cpu.prof ./testing/benchmarks/...

# With memory profiling
go test -v -bench=. -benchmem -memprofile=mem.prof ./testing/benchmarks/...
```

## Benchmark Categories

### Synapse Creation

Measures the cost of creating synapse instances:
- Schema generation from Go types
- Pipeline construction
- Option application

### Fire Operations

Measures the overhead of executing synapses:
- Prompt construction
- Session management
- JSON parsing and validation
- Pipeline processing

### Session Operations

Measures session management costs:
- Message append/access
- Prune/truncate operations
- Concurrent access patterns

## Interpreting Results

```
BenchmarkSynapse_Creation/Binary-8         50000    25000 ns/op    4096 B/op    50 allocs/op
                          ^^^^^^           ^^^^^    ^^^^^^^^^      ^^^^^^^^^^   ^^^^^^^^^^^^
                          name             iters    time/op        bytes/op     allocs/op
```

- **iters**: Number of iterations run
- **time/op**: Average time per operation
- **bytes/op**: Average bytes allocated per operation
- **allocs/op**: Average allocations per operation

## Performance Targets

| Operation | Target | Notes |
|-----------|--------|-------|
| Synapse creation | < 100µs | One-time cost |
| Fire (mock provider) | < 50µs | Excludes provider latency |
| Session append | < 1µs | Per message |
| JSON parsing | < 10µs | Typical response size |

## Comparing Results

```bash
# Save baseline
go test -bench=. -benchmem ./testing/benchmarks/... > baseline.txt

# After changes
go test -bench=. -benchmem ./testing/benchmarks/... > current.txt

# Compare (requires benchstat)
benchstat baseline.txt current.txt
```
