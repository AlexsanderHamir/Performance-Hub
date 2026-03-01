package parser

import (
	"fmt"
	"os"
	"sort"

	"github.com/google/pprof/profile"
)

// ParseProfile reads a pprof profile file (e.g. from -cpuprofile) and returns
// the parsed Profile. Use pprof's own types for all analytical data.
func ParseProfile(path string) (*profile.Profile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return profile.Parse(f)
}

// Digest is the result of parsing a pprof profile into a digestible structure:
// the same analytical data from the profile, organized for step-by-step use.
type Digest struct {
	Profile *profile.Profile

	// Sample types (e.g. "samples", "cpu")
	SampleTypes []string

	// Total sample count
	TotalSamples int64

	// Functions sorted by total value (e.g. CPU time), descending
	TopFunctions []FuncStat

	// Call graph: who called whom and how much value flowed (caller → callee).
	// Sorted by Value descending so hottest edges first.
	Edges []CallEdge

	// Duration and period from the profile
	DurationNanos int64
	Period        int64
	PeriodType    string
}

// FuncStat is per-function analytical data.
type FuncStat struct {
	Name       string
	SystemName string
	Filename   string
	Value      int64 // total samples attributed to this function
}

// CallEdge represents a caller→callee relationship and how much value (e.g. CPU time)
// flowed along that edge. Same as pprof's call graph: "Caller called Callee" for Value.
type CallEdge struct {
	Caller string
	Callee string
	Value  int64
}

// functionName returns the primary function name for a location (first Line's Function).
func functionName(loc *profile.Location) string {
	for _, line := range loc.Line {
		if line.Function != nil {
			return line.Function.Name
		}
	}
	return ""
}

// DigestProfile parses a pprof Profile into a Digest: the profile's analytical data
// in a structured form you can walk step by step (sample types, functions, call graph, etc.).
func DigestProfile(p *profile.Profile) (*Digest, error) {
	if err := p.CheckValid(); err != nil {
		return nil, err
	}

	d := &Digest{Profile: p}
	for _, st := range p.SampleType {
		d.SampleTypes = append(d.SampleTypes, st.Type+"/"+st.Unit)
	}
	d.DurationNanos = p.DurationNanos
	d.Period = p.Period
	if p.PeriodType != nil {
		d.PeriodType = p.PeriodType.Type + "/" + p.PeriodType.Unit
	}

	// Aggregate value per location, then per function (pprof uses Location → Line → Function).
	locValue := make(map[*profile.Location]int64)
	for _, sample := range p.Sample {
		var v int64
		for _, val := range sample.Value {
			v += val
		}
		for _, loc := range sample.Location {
			locValue[loc] += v
		}
		d.TotalSamples += v
	}

	// Build call graph: for each sample stack, caller = Location[i+1], callee = Location[i].
	edgeValue := make(map[string]int64)
	for _, sample := range p.Sample {
		var v int64
		for _, val := range sample.Value {
			v += val
		}
		locs := sample.Location
		for i := 0; i < len(locs)-1; i++ {
			callee := functionName(locs[i])
			caller := functionName(locs[i+1])
			if caller != "" && callee != "" {
				key := caller + "\n" + callee
				edgeValue[key] += v
			}
		}
	}

	for key, val := range edgeValue {
		// key is "caller\ncallee"
		i := 0
		for j := range key {
			if key[j] == '\n' {
				i = j
				break
			}
		}
		caller := key[:i]
		callee := key[i+1:]
		d.Edges = append(d.Edges, CallEdge{Caller: caller, Callee: callee, Value: val})
	}
	sort.Slice(d.Edges, func(i, j int) bool {
		return d.Edges[i].Value > d.Edges[j].Value
	})

	funcValue := make(map[uint64]int64)
	for loc, v := range locValue {
		for _, line := range loc.Line {
			if line.Function != nil {
				funcValue[line.Function.ID] += v
			}
		}
	}

	for _, fn := range p.Function {
		v := funcValue[fn.ID]
		if v == 0 {
			continue
		}
		d.TopFunctions = append(d.TopFunctions, FuncStat{
			Name:       fn.Name,
			SystemName: fn.SystemName,
			Filename:   fn.Filename,
			Value:      v,
		})
	}
	sort.Slice(d.TopFunctions, func(i, j int) bool {
		return d.TopFunctions[i].Value > d.TopFunctions[j].Value
	})
	return d, nil
}

const nanosPerSecond = 1e9

// isTimeProfile returns true if the profile's sample values are in nanoseconds (e.g. cpu/nanoseconds).
func isTimeProfile(d *Digest) bool {
	for _, st := range d.SampleTypes {
		if st == "cpu/nanoseconds" {
			return true
		}
	}
	return false
}

// PrintDigest writes the parsed digest to stdout (step-by-step view of the results).
// Time values are shown in seconds to match pprof's usual display.
func PrintDigest(d *Digest) {
	fmt.Println("--- Parsed profile (digest) ---")
	fmt.Println("Sample types:", d.SampleTypes)
	fmt.Println("Total samples:", d.TotalSamples)
	fmt.Printf("Duration: %.4gs\n", float64(d.DurationNanos)/nanosPerSecond)
	periodSec := float64(d.Period) / nanosPerSecond
	fmt.Printf("Period: %s %.4g\n", d.PeriodType, periodSec)
	fmt.Println()

	showValueSec := isTimeProfile(d)
	fmt.Println("Top functions by sample value:")
	for i, f := range d.TopFunctions {
		if i >= 15 {
			break
		}
		if showValueSec {
			fmt.Printf("  %d\t%s\t(value=%.4gs)\n", i+1, f.Name, float64(f.Value)/nanosPerSecond)
		} else {
			fmt.Printf("  %d\t%s\t(value=%d)\n", i+1, f.Name, f.Value)
		}
		if f.Filename != "" {
			fmt.Printf("       \t%s\n", f.Filename)
		}
	}

	if len(d.Edges) > 0 {
		fmt.Println()
		fmt.Println("Call graph (tree):")
		printCallTree(d, showValueSec)
	}
}

// printCallTree prints the call graph as a tree with branch characters (├ └ │).
// Roots = functions that are never callees; from each root we recurse into callees.
func printCallTree(d *Digest, showValueSec bool) {
	byCaller := make(map[string][]CallEdge)
	callerTotal := make(map[string]int64)
	callees := make(map[string]bool)
	for _, e := range d.Edges {
		byCaller[e.Caller] = append(byCaller[e.Caller], e)
		callerTotal[e.Caller] += e.Value
		callees[e.Callee] = true
	}
	var roots []string
	for c := range byCaller {
		if !callees[c] {
			roots = append(roots, c)
		}
	}
	sort.Slice(roots, func(i, j int) bool {
		return callerTotal[roots[i]] > callerTotal[roots[j]]
	})
	visited := make(map[string]bool)
	for _, root := range roots {
		fmt.Printf("  %s\n", root)
		printCallNode(d, byCaller, showValueSec, root, "", true, visited)
	}
}

func printCallNode(d *Digest, byCaller map[string][]CallEdge, showValueSec bool, name string, prefix string, isLast bool, visited map[string]bool) {
	edges := byCaller[name]
	if len(edges) == 0 {
		return
	}
	sort.Slice(edges, func(i, j int) bool { return edges[i].Value > edges[j].Value })
	for i, e := range edges {
		pct := 100 * float64(e.Value) / float64(d.TotalSamples)
		last := i == len(edges)-1
		var branch, nextPrefix string
		if last {
			branch = "└─ "
			nextPrefix = prefix + "    "
		} else {
			branch = "├─ "
			nextPrefix = prefix + "│   "
		}
		valStr := ""
		if showValueSec {
			valStr = fmt.Sprintf("  (%.4gs)", float64(e.Value)/nanosPerSecond)
		} else {
			valStr = fmt.Sprintf("  (%d)", e.Value)
		}
		fmt.Printf("%s%s%.2f%%  %s%s\n", prefix, branch, pct, e.Callee, valStr)
		if visited[e.Callee] {
			fmt.Printf("%s    (cycle)\n", nextPrefix)
			continue
		}
		visited[e.Callee] = true
		printCallNode(d, byCaller, showValueSec, e.Callee, nextPrefix, last, visited)
		visited[e.Callee] = false
	}
}
