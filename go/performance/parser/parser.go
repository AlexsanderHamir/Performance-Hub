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

// DigestProfile parses a pprof Profile into a Digest: the profile's analytical data
// in a structured form you can walk step by step (sample types, functions, etc.).
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
}
