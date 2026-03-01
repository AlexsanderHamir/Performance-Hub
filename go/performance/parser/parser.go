package parser

import (
	"fmt"
	"io"
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
	return ParseProfileFromReader(f)
}

// ParseProfileFromReader parses a pprof profile from r. Used by ParseProfile and by tests with in-memory data.
func ParseProfileFromReader(r io.Reader) (*profile.Profile, error) {
	return profile.Parse(r)
}

// DigestProfile parses a pprof Profile into a Digest. Composes small pure helpers
// for metadata, location aggregation, call edges, and function stats.
func DigestProfile(p *profile.Profile) (*Digest, error) {
	if err := p.CheckValid(); err != nil {
		return nil, err
	}
	d := &Digest{Profile: p}
	fillDigestMetadata(d, p)
	locValue, total := aggregateSampleLocations(p)
	d.TotalSamples = total
	d.Edges = buildCallEdgesFromProfile(p)
	sort.Slice(d.Edges, func(i, j int) bool { return d.Edges[i].Value > d.Edges[j].Value })
	d.TopFunctions = aggregateTopFunctions(p, locValue)
	sort.Slice(d.TopFunctions, func(i, j int) bool { return d.TopFunctions[i].Value > d.TopFunctions[j].Value })
	return d, nil
}

func fillDigestMetadata(d *Digest, p *profile.Profile) {
	for _, st := range p.SampleType {
		d.SampleTypes = append(d.SampleTypes, st.Type+"/"+st.Unit)
	}
	d.DurationNanos = p.DurationNanos
	d.Period = p.Period
	if p.PeriodType != nil {
		d.PeriodType = p.PeriodType.Type + "/" + p.PeriodType.Unit
	}
}

func sampleTotalValue(sample *profile.Sample) int64 {
	var v int64
	for _, val := range sample.Value {
		v += val
	}
	return v
}

func aggregateSampleLocations(p *profile.Profile) (map[*profile.Location]int64, int64) {
	locValue := make(map[*profile.Location]int64)
	var total int64
	for _, sample := range p.Sample {
		v := sampleTotalValue(sample)
		for _, loc := range sample.Location {
			locValue[loc] += v
		}
		total += v
	}
	return locValue, total
}

func buildCallEdgesFromProfile(p *profile.Profile) []CallEdge {
	edgeValue := make(map[string]int64)
	for _, sample := range p.Sample {
		v := sampleTotalValue(sample)
		locs := sample.Location
		for i := 0; i < len(locs)-1; i++ {
			callee := FunctionName(locs[i])
			caller := FunctionName(locs[i+1])
			if caller != "" && callee != "" {
				edgeValue[caller+"\n"+callee] += v
			}
		}
	}
	var edges []CallEdge
	for key, val := range edgeValue {
		caller, callee := splitEdgeKey(key)
		edges = append(edges, CallEdge{Caller: caller, Callee: callee, Value: val})
	}
	return edges
}

func splitEdgeKey(key string) (caller, callee string) {
	for i := 0; i < len(key); i++ {
		if key[i] == '\n' {
			return key[:i], key[i+1:]
		}
	}
	return "", ""
}

func aggregateTopFunctions(p *profile.Profile, locValue map[*profile.Location]int64) []FuncStat {
	funcValue := make(map[uint64]int64)
	for loc, v := range locValue {
		for _, line := range loc.Line {
			if line.Function != nil {
				funcValue[line.Function.ID] += v
			}
		}
	}
	var out []FuncStat
	for _, fn := range p.Function {
		v := funcValue[fn.ID]
		if v == 0 {
			continue
		}
		out = append(out, FuncStat{Name: fn.Name, SystemName: fn.SystemName, Filename: fn.Filename, Value: v})
	}
	return out
}

const defaultTopFunctionsLimit = 15

// PrintDigest writes the parsed digest to w (step-by-step view). Pass nil for w to use os.Stdout.
// If focus is non-empty, the call graph is limited to functions whose name contains focus.
func PrintDigest(d *Digest, focus string, w io.Writer) {
	if w == nil {
		w = os.Stdout
	}
	printDigestHeader(d, w)
	showValueSec := IsTimeProfile(d)
	printTopFunctions(d, defaultTopFunctionsLimit, showValueSec, w)
	if len(d.Edges) > 0 {
		printCallGraphSection(d, focus, showValueSec, w)
	}
}

func printDigestHeader(d *Digest, w io.Writer) {
	fmt.Fprintln(w, "--- Parsed profile (digest) ---")
	fmt.Fprintln(w, "Sample types:", d.SampleTypes)
	fmt.Fprintln(w, "Total samples:", d.TotalSamples)
	fmt.Fprintf(w, "Duration: %ss\n", FormatDuration(d.DurationNanos))
	fmt.Fprintf(w, "Period: %s %s\n", d.PeriodType, FormatDuration(d.Period))
	fmt.Fprintln(w)
}

func printTopFunctions(d *Digest, limit int, showValueSec bool, w io.Writer) {
	fmt.Fprintln(w, "Top functions by sample value:")
	for i := 0; i < limit && i < len(d.TopFunctions); i++ {
		f := d.TopFunctions[i]
		fmt.Fprintf(w, "  %d\t%s\t(value=%s)\n", i+1, f.Name, FormatValue(f.Value, showValueSec))
		if f.Filename != "" {
			fmt.Fprintf(w, "       \t%s\n", f.Filename)
		}
	}
	fmt.Fprintln(w)
}

func printCallGraphSection(d *Digest, focus string, showValueSec bool, w io.Writer) {
	if focus != "" {
		fmt.Fprintf(w, "Call graph (tree, focused on %q):\n", focus)
	} else {
		fmt.Fprintln(w, "Call graph (tree):")
	}
	cg := BuildCallGraph(d.Edges)
	roots := cg.Roots(focus)
	if roots == nil && focus != "" {
		fmt.Fprintf(w, "  No function matching %q\n", focus)
		return
	}
	PrintCallTree(cg, roots, d.TotalSamples, showValueSec, w)
}
