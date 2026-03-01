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

// Summary holds analytical data derived from a pprof Profile.
type Summary struct {
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

// Summarize builds analytical data from a parsed profile using pprof's data.
func Summarize(p *profile.Profile) (*Summary, error) {
	if err := p.CheckValid(); err != nil {
		return nil, err
	}

	s := &Summary{Profile: p}
	for _, st := range p.SampleType {
		s.SampleTypes = append(s.SampleTypes, st.Type+"/"+st.Unit)
	}
	s.DurationNanos = p.DurationNanos
	s.Period = p.Period
	if p.PeriodType != nil {
		s.PeriodType = p.PeriodType.Type + "/" + p.PeriodType.Unit
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
		s.TotalSamples += v
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
		s.TopFunctions = append(s.TopFunctions, FuncStat{
			Name:       fn.Name,
			SystemName: fn.SystemName,
			Filename:   fn.Filename,
			Value:      v,
		})
	}
	sort.Slice(s.TopFunctions, func(i, j int) bool {
		return s.TopFunctions[i].Value > s.TopFunctions[j].Value
	})
	return s, nil
}

const nanosPerSecond = 1e9

// isTimeProfile returns true if the profile's sample values are in nanoseconds (e.g. cpu/nanoseconds).
func isTimeProfile(s *Summary) bool {
	for _, st := range s.SampleTypes {
		if st == "cpu/nanoseconds" {
			return true
		}
	}
	return false
}

// PrintSummary writes a human-readable summary of the profile to stdout.
// Time values are shown in seconds to match pprof's usual display.
func PrintSummary(s *Summary) {
	fmt.Println("--- Profile summary (via pprof profile package) ---")
	fmt.Println("Sample types:", s.SampleTypes)
	fmt.Println("Total samples:", s.TotalSamples)
	fmt.Printf("Duration: %.4gs\n", float64(s.DurationNanos)/nanosPerSecond)
	periodSec := float64(s.Period) / nanosPerSecond
	fmt.Printf("Period: %s %.4g\n", s.PeriodType, periodSec)
	fmt.Println()

	showValueSec := isTimeProfile(s)
	fmt.Println("Top functions by sample value:")
	for i, f := range s.TopFunctions {
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
