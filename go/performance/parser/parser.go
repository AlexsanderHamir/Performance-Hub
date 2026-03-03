package parser

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/google/pprof/profile"
)

func ParseProfile(path string) (*profile.Profile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	p, err := ParseProfileFromReader(f)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func ParseProfileFromReader(r io.Reader) (*profile.Profile, error) {
	return profile.Parse(r)
}

func DigestProfile(p *profile.Profile) (*Digest, error) {
	if err := p.CheckValid(); err != nil {
		return nil, err
	}
	d := &Digest{Profile: p}
	fillDigestMetadata(d, p)
	_, total := aggregateSampleLocations(p)
	d.TotalSamples = total
	d.Edges = buildCallEdgesFromProfile(p)
	sort.Slice(d.Edges, func(i, j int) bool { return d.Edges[i].Value > d.Edges[j].Value })
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
	sampleValueByLocation := make(map[*profile.Location]int64)
	var total int64
	for _, sample := range p.Sample {
		v := sampleTotalValue(sample)
		for _, loc := range sample.Location {
			sampleValueByLocation[loc] += v
		}
		total += v
	}
	return sampleValueByLocation, total
}

func buildCallEdgesFromProfile(p *profile.Profile) []CallEdge {
	valueByEdgeKey := make(map[string]int64)
	for _, sample := range p.Sample {
		v := sampleTotalValue(sample)
		locs := sample.Location
		for i := 0; i < len(locs)-1; i++ {
			callee := FunctionName(locs[i])
			caller := FunctionName(locs[i+1])
			if caller != "" && callee != "" {
				valueByEdgeKey[caller+"\n"+callee] += v
			}
		}
	}
	var edges []CallEdge
	for key, val := range valueByEdgeKey {
		caller, callee := splitEdgeKey(key)
		edges = append(edges, CallEdge{Caller: caller, Callee: callee, Value: val})
	}
	return edges
}

func splitEdgeKey(key string) (caller, callee string) {
	caller, callee, ok := strings.Cut(key, "\n")
	if !ok {
		return "", ""
	}
	return caller, callee
}

// PrintDigest: pass nil for w to use os.Stdout; non-empty focus limits the call graph to functions whose name contains focus.
func PrintDigest(d *Digest, focus string, w io.Writer) {
	if w == nil {
		w = os.Stdout
	}
	printDigestHeader(d, w)
	showValueInSeconds := IsTimeProfile(d)
	if len(d.Edges) > 0 {
		printCallGraphSection(d, focus, showValueInSeconds, w)
	}
}

// PrintCallGraph writes only the call graph (tree) section to w. Pass nil for w to use os.Stdout. focus limits the tree to functions whose name contains focus.
func PrintCallGraph(d *Digest, focus string, w io.Writer) {
	if w == nil {
		w = os.Stdout
	}
	showValueInSeconds := IsTimeProfile(d)
	if len(d.Edges) > 0 {
		printCallGraphSection(d, focus, showValueInSeconds, w)
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

func printCallGraphSection(d *Digest, focus string, showValueInSeconds bool, w io.Writer) {
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
	PrintCallTree(cg, roots, d.TotalSamples, showValueInSeconds, w)
}
