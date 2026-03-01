// Parser CLI: parses a pprof profile using the pprof profile package and
// prints analytical data (function names, sample types, top functions, etc.).
package main

import (
	"fmt"
	"os"

	"github.com/AlexsanderHamir/Performance-Hub/go/performance/parser"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <cpu.prof>\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Generate a profile first: cd go && go test -cpuprofile=cpu.prof -bench=. ./performance/parser/")
		os.Exit(1)
	}
	path := os.Args[1]

	p, err := parser.ParseProfile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	s, err := parser.Summarize(p)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	parser.PrintSummary(s)
}
