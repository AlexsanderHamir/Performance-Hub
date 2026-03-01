# Parser — pprof profile parser

Uses **pprof’s own** `github.com/google/pprof/profile` package to parse CPU (or other) profiles and expose analytical data: function names, sample types, top functions by value, duration, etc.

## 1. Generate a CPU profile

From the repo’s `go/` directory:

```bash
cd go
go test -cpuprofile=cpu.prof -bench=. ./performance/parser/
```

This runs the small benchmarks in this package and writes `cpu.prof` in the current directory.

## 2. Parse and view results

Using the CLI:

```bash
go run ./cmd/parser/ cpu.prof
```

Isolate the call tree to a specific function (substring match):

```bash
go run ./cmd/parser/ -focus parser.work cpu.prof
```

Or build and run:

```bash
go build -o parser ./cmd/parser/
./parser cpu.prof
./parser -focus "sha256.Write" cpu.prof
```

## 3. Use the parser in code

```go
import "github.com/AlexsanderHamir/Performance-Hub/go/performance/parser"

p, err := parser.ParseProfile("cpu.prof")
if err != nil { ... }

// DigestProfile parses the profile into a step-by-step structure (sample types, functions, etc.)
digest, err := parser.DigestProfile(p)
parser.PrintDigest(digest, "", nil)            // full call tree to stdout
parser.PrintDigest(digest, "parser.work", nil) // call tree focused on "parser.work"
parser.PrintDigest(digest, "", myWriter)       // or inject your own io.Writer
```

All analytical data comes from the official `profile` types (e.g. `Profile.Function`, `Profile.Sample`, `Location`, `ValueType`).
