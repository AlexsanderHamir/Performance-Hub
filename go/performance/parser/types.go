package parser

import "github.com/google/pprof/profile"

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

type FuncStat struct {
	Name       string
	SystemName string
	Filename   string
	Value      int64
}

type CallEdge struct {
	Caller string
	Callee string
	Value  int64
}
