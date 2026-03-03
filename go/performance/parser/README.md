# parser

Parses pprof CPU (or other) profiles into a digest so you can inspect sample types and the call graph without re-parsing raw profile format. Exists because pprof’s `profile` package gives raw structures; this package adds aggregated views (edges, tree printing) and a small CLI to view the call graph.

## Usage

**CLI** (from repo `go/` directory):

The CLI has no default feature. You must pass `-focus` to request the only supported feature (call graph tree). Example:

```bash
go run ./cmd/parser/ -focus "parser.work" cpu.prof
```

Generate a profile first:

```bash
go test -cpuprofile=cpu.prof -bench=. ./performance/parser/
```

**Library**:

```go
import "github.com/AlexsanderHamir/Performance-Hub/go/performance/parser"

p, err := parser.ParseProfile("cpu.prof")
if err != nil {
	// file missing or invalid profile format
	return err
}
digest, err := parser.DigestProfile(p)
if err != nil {
	// profile failed CheckValid (e.g. sample type mismatch)
	return err
}
parser.PrintDigest(digest, "", nil)             // digest header + full call tree to stdout
parser.PrintDigest(digest, "parser.work", nil)  // digest header + call tree limited to names containing "parser.work"
parser.PrintCallGraph(digest, "parser.work", nil) // only call graph (tree) to stdout
parser.PrintDigest(digest, "", myWriter)        // write to custom io.Writer
```

## Configuration

The parser package has no env or config. The CLI (`cmd/parser`) supports:

| Name   | Type   | Default | Description |
|--------|--------|---------|-------------|
| `-focus` | string | (required) | Limit call graph to functions whose name contains this substring (e.g. `parser.work`). Must be set; CLI has no default feature. |
| positional | string | (required) | Path to the pprof profile file (e.g. `cpu.prof`). |

## Dependencies

- **github.com/google/pprof/profile** — parsing and types (`Profile`, `Location`, `Function`, `Sample`, `ValueType`). All analytical data in `Digest` is derived from these; the package does not redefine profile semantics.

## Helpers

Grouped by concern.

### Parsing and digest

| Symbol | Signature | Behavior |
|--------|-----------|----------|
| `ParseProfile` | `(path string) (*profile.Profile, error)` | Reads a profile file from disk and parses it. Fails if the file cannot be opened or content is not valid pprof. |
| `ParseProfileFromReader` | `(r io.Reader) (*profile.Profile, error)` | Parses a profile from an `io.Reader`. Use for in-memory or streaming input. |
| `DigestProfile` | `(p *profile.Profile) (*Digest, error)` | Turns a valid `Profile` into a `Digest` (sample types, total, edges). Returns error if `p.CheckValid()` fails. |

### Types (output of digest)

| Symbol | Purpose |
|--------|---------|
| `Digest` | Aggregated view: `Profile`, `SampleTypes`, `TotalSamples`, `Edges`, `DurationNanos`, `Period`, `PeriodType`. |
| `CallEdge` | One caller→callee edge: `Caller`, `Callee`, `Value`. |

### Inspecting profile and digest

| Symbol | Signature | Behavior |
|--------|-----------|----------|
| `FunctionName` | `(loc *profile.Location) string` | Primary function name for a location (first `Line`’s `Function.Name`). Returns `""` if no function. |
| `IsTimeProfile` | `(d *Digest) bool` | True if any sample type is `"cpu/nanoseconds"`, so you can format values as seconds. |

### Formatting

| Symbol | Signature | Behavior |
|--------|-----------|----------|
| `FormatDuration` | `(nanos int64) string` | Nanoseconds as seconds (e.g. `"3.59"`). |
| `FormatValue` | `(value int64, showSeconds bool) string` | If `showSeconds` true, value as seconds with `"s"` suffix; otherwise integer string. |
| `NanosPerSecond` | `const = 1e9` | Used by duration/value formatting. |

### Call graph

| Symbol | Signature | Behavior |
|--------|-----------|----------|
| `BuildCallGraph` | `(edges []CallEdge) *CallGraph` | Builds a caller-keyed graph from `Digest.Edges`. |
| `CallGraph.Roots` | `(focus string) []string` | Root names: empty `focus` = top-of-stack callers; non-empty = functions whose name contains `focus`. Returns `nil` when `focus` is set but no function matches. Sorted by total value descending. |
| `CallGraph.EdgesFrom` | `(caller string) []CallEdge` | Edges from `caller`, sorted by value descending. |
| `CallGraph.TotalFrom` | `(caller string) int64` | Sum of all edge values from `caller`. |

### Printing

| Symbol | Signature | Behavior |
|--------|-----------|----------|
| `PrintDigest` | `(d *Digest, focus string, w io.Writer)` | Writes digest header and call tree to `w`. Pass `nil` for `w` to use `os.Stdout`. Non-empty `focus` limits the call graph to functions whose name contains `focus`. |
| `PrintCallGraph` | `(d *Digest, focus string, w io.Writer)` | Writes only the call graph (tree) section to `w`. Pass `nil` for `w` to use `os.Stdout`. Used by the CLI. |
| `PrintCallTree` | `(cg *CallGraph, roots []string, totalSamples int64, showValueInSeconds bool, w io.Writer)` | Writes a tree of roots and their callees to `w`. If `roots` is nil or empty, writes nothing (caller can print a “no match” message). |

## Errors

| Where | Condition | Cause |
|-------|-----------|--------|
| `ParseProfile` | `err != nil` | File not found, permission denied, or file content is not valid pprof. |
| `ParseProfileFromReader` | `err != nil` | Reader content is not valid pprof. |
| `DigestProfile` | `err != nil` | `p.CheckValid()` failed (e.g. sample type count does not match `Sample.Value` length, or profile is otherwise inconsistent). |

## Decisions

See [DECISIONS.md](./DECISIONS.md).
