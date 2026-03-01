// Helpers: composable building blocks for working with a digested profile.
//
//   - FunctionName, IsTimeProfile: inspect profile/location and digest.
//   - FormatDuration, FormatValue: format numbers for output.
//   - BuildCallGraph(d.Edges) → *CallGraph: build a caller-keyed graph.
//   - CallGraph.Roots(focus): get root names (full tree or filtered by focus).
//   - CallGraph.EdgesFrom(caller), TotalFrom(caller): walk the graph.
//   - PrintCallTree(cg, roots, ...): write tree to an io.Writer (compose with any output).
package parser

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/google/pprof/profile"
)

// NanosPerSecond is the number of nanoseconds in one second (for time formatting).
const NanosPerSecond = 1e9

// FunctionName returns the primary function name for a profile location (first Line's Function).
// Use this when walking sample stacks to get caller/callee names.
func FunctionName(loc *profile.Location) string {
	for _, line := range loc.Line {
		if line.Function != nil {
			return line.Function.Name
		}
	}
	return ""
}

// IsTimeProfile returns true if the digest's sample values are in nanoseconds (e.g. cpu/nanoseconds),
// so you can choose to format values as seconds.
func IsTimeProfile(d *Digest) bool {
	for _, st := range d.SampleTypes {
		if st == "cpu/nanoseconds" {
			return true
		}
	}
	return false
}

// FormatDuration formats nanoseconds as seconds (e.g. "3.59").
func FormatDuration(nanos int64) string {
	return fmt.Sprintf("%.4g", float64(nanos)/NanosPerSecond)
}

// FormatValue formats a sample value: as seconds when showSeconds is true (for CPU profiles), else as integer.
func FormatValue(value int64, showSeconds bool) string {
	if showSeconds {
		return fmt.Sprintf("%.4gs", float64(value)/NanosPerSecond)
	}
	return fmt.Sprintf("%d", value)
}

// CallGraph is a caller-keyed view of call edges, for walking or printing the call tree.
// Build it with BuildCallGraph; then use Roots and EdgesFrom to compose your own views.
type CallGraph struct {
	byCaller    map[string][]CallEdge
	callerTotal map[string]int64
	callees     map[string]bool
	allNames    map[string]bool
}

// BuildCallGraph builds a CallGraph from the given edges. Use d.Edges after DigestProfile.
func BuildCallGraph(edges []CallEdge) *CallGraph {
	byCaller := make(map[string][]CallEdge)
	callerTotal := make(map[string]int64)
	callees := make(map[string]bool)
	allNames := make(map[string]bool)
	for _, e := range edges {
		byCaller[e.Caller] = append(byCaller[e.Caller], e)
		callerTotal[e.Caller] += e.Value
		callees[e.Callee] = true
		allNames[e.Caller] = true
		allNames[e.Callee] = true
	}
	return &CallGraph{
		byCaller:    byCaller,
		callerTotal: callerTotal,
		callees:     callees,
		allNames:    allNames,
	}
}

// Roots returns the root function names for the graph. If focus is empty, returns functions
// that are never callees (top of stack). If focus is non-empty, returns functions whose name
// contains focus (substring match). Sorted by total value descending. Returns nil if focus
// is set but no function matches.
func (cg *CallGraph) Roots(focus string) []string {
	roots := cg.collectRoots(focus)
	if focus != "" && len(roots) == 0 {
		return nil
	}
	return cg.sortedRootsByValue(roots)
}

func (cg *CallGraph) sortedRootsByValue(roots []string) []string {
	sort.Slice(roots, func(i, j int) bool {
		return cg.callerTotal[roots[i]] > cg.callerTotal[roots[j]]
	})
	return roots
}

func (cg *CallGraph) collectRoots(focus string) []string {
	var roots []string
	if focus != "" {
		for name := range cg.allNames {
			if strings.Contains(name, focus) {
				roots = append(roots, name)
			}
		}
		return roots
	}
	for c := range cg.byCaller {
		if !cg.callees[c] {
			roots = append(roots, c)
		}
	}
	return roots
}

// EdgesFrom returns the call edges from the given caller, sorted by value descending.
func (cg *CallGraph) EdgesFrom(caller string) []CallEdge {
	edges := cg.byCaller[caller]
	out := make([]CallEdge, len(edges))
	copy(out, edges)
	sort.Slice(out, func(i, j int) bool { return out[i].Value > out[j].Value })
	return out
}

// TotalFrom returns the total value of all edges from the given caller.
func (cg *CallGraph) TotalFrom(caller string) int64 {
	return cg.callerTotal[caller]
}

// treeBranch returns the branch character and prefix for the next line (├ └ │).
func treeBranch(last bool, prefix string) (branch, nextPrefix string) {
	if last {
		return "└─ ", prefix + "    "
	}
	return "├─ ", prefix + "│   "
}

// PrintCallTree writes the call tree to w: roots first, then each branch with ├ └ │.
// If roots is nil or empty, nothing is written (caller can print a "no match" message).
func PrintCallTree(cg *CallGraph, roots []string, totalSamples int64, showValueSec bool, w io.Writer) {
	if len(roots) == 0 {
		return
	}
	visited := make(map[string]bool)
	for _, root := range roots {
		fmt.Fprintf(w, "  %s\n", root)
		printCallNode(cg, totalSamples, showValueSec, root, "", true, visited, w)
	}
}

func printCallNode(cg *CallGraph, totalSamples int64, showValueSec bool, name string, prefix string, isLast bool, visited map[string]bool, w io.Writer) {
	edges := cg.EdgesFrom(name)
	if len(edges) == 0 {
		return
	}
	for i, e := range edges {
		last := i == len(edges)-1
		branch, nextPrefix := treeBranch(last, prefix)
		pct := 100 * float64(e.Value) / float64(totalSamples)
		fmt.Fprintf(w, "%s%s%.2f%%  %s  (%s)\n", prefix, branch, pct, e.Callee, FormatValue(e.Value, showValueSec))
		if visited[e.Callee] {
			fmt.Fprintf(w, "%s    (cycle)\n", nextPrefix)
			continue
		}
		visited[e.Callee] = true
		printCallNode(cg, totalSamples, showValueSec, e.Callee, nextPrefix, last, visited, w)
		visited[e.Callee] = false
	}
}
