package parser

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/google/pprof/profile"
)

const NanosPerSecond = 1e9

func FunctionName(loc *profile.Location) string {
	for _, line := range loc.Line {
		if line.Function != nil {
			return line.Function.Name
		}
	}
	return ""
}

func IsTimeProfile(d *Digest) bool {
	for _, st := range d.SampleTypes {
		if st == "cpu/nanoseconds" {
			return true
		}
	}
	return false
}

func FormatDuration(nanos int64) string {
	return fmt.Sprintf("%.4g", float64(nanos)/NanosPerSecond)
}

func FormatValue(value int64, showSeconds bool) string {
	if showSeconds {
		return fmt.Sprintf("%.4gs", float64(value)/NanosPerSecond)
	}
	return fmt.Sprintf("%d", value)
}

type CallGraph struct {
	byCaller    map[string][]CallEdge
	callerTotal map[string]int64
	callees     map[string]bool
	allNames    map[string]bool
}

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

// Roots: empty focus returns top-of-stack roots; non-empty focus returns functions whose name contains focus (nil if no match).
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

func (cg *CallGraph) EdgesFrom(caller string) []CallEdge {
	edges := cg.byCaller[caller]
	out := make([]CallEdge, len(edges))
	copy(out, edges)
	sort.Slice(out, func(i, j int) bool { return out[i].Value > out[j].Value })
	return out
}

func (cg *CallGraph) TotalFrom(caller string) int64 {
	return cg.callerTotal[caller]
}

//go:noinline
func treeBranch(last bool, prefix string) (branch, nextPrefix string) {
	if last {
		return "└─ ", prefix + "    "
	}
	return "├─ ", prefix + "│   "
}

func PrintCallTree(cg *CallGraph, roots []string, totalSamples int64, showValueInSeconds bool, w io.Writer) {
	if len(roots) == 0 {
		return
	}
	visited := make(map[string]bool)
	for _, root := range roots {
		fmt.Fprintf(w, "  %s\n", root)
		printCallNode(cg, totalSamples, showValueInSeconds, root, "", true, visited, w)
	}
}

func printCallNode(cg *CallGraph, totalSamples int64, showValueInSeconds bool, name string, prefix string, isLast bool, visited map[string]bool, w io.Writer) {
	edges := cg.EdgesFrom(name)
	if len(edges) == 0 {
		return
	}
	for i, e := range edges {
		last := i == len(edges)-1
		branch, nextPrefix := treeBranch(last, prefix)
		pct := 100 * float64(e.Value) / float64(totalSamples)
		fmt.Fprintf(w, "%s%s%.2f%%  %s  (%s)\n", prefix, branch, pct, e.Callee, FormatValue(e.Value, showValueInSeconds))
		if visited[e.Callee] {
			fmt.Fprintf(w, "%s    (cycle)\n", nextPrefix)
			continue
		}
		visited[e.Callee] = true
		printCallNode(cg, totalSamples, showValueInSeconds, e.Callee, nextPrefix, last, visited, w)
		visited[e.Callee] = false
	}
}
