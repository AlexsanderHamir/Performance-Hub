package parser

import "github.com/google/pprof/profile"

// Digest is the result of parsing a pprof profile into a digestible structure:
// the same analytical data from the profile, organized for step-by-step use.
type Digest struct {
	Profile *profile.Profile

	SampleTypes   []string
	TotalSamples  int64
	TopFunctions  []FuncStat
	Edges         []CallEdge
	DurationNanos int64
	Period        int64
	PeriodType    string
}

// FuncStat is per-function analytical data.
type FuncStat struct {
	Name       string
	SystemName string
	Filename   string
	Value      int64
}

// CallEdge is a caller→callee relationship and the value (e.g. CPU time) on that edge.
type CallEdge struct {
	Caller string
	Callee string
	Value  int64
}
