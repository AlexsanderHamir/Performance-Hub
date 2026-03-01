# Parser — pprof profile parser

Uses **pprof’s own** `github.com/google/pprof/profile` package to parse CPU (or other) profiles and expose analytical data: function names, sample types, top functions by value, duration, etc.

## 1. Generate a CPU profile

From the repo’s `go/` directory:

```bash
cd go
go test -cpuprofile=cpu.prof -bench=. ./performance/parser/
```

This runs the small benchmarks in this package and writes `cpu.prof` in the current directory.

## 2. Parse and print summary

Using the CLI:

```bash
go run ./cmd/parser/ cpu.prof
```

Or build and run:

```bash
go build -o parser ./cmd/parser/
./parser cpu.prof
```

## 3. Use the parser in code

```go
import "github.com/AlexsanderHamir/Performance-Hub/go/performance/parser"

p, err := parser.ParseProfile("cpu.prof")
if err != nil { ... }

// Use pprof’s *profile.Profile (Sample, Location, Function, SampleType, etc.)
summary, err := parser.Summarize(p)
parser.PrintSummary(summary)
```

All analytical data comes from the official `profile` types (e.g. `Profile.Function`, `Profile.Sample`, `Location`, `ValueType`).
