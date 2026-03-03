// Parser CLI: parses a pprof profile and prints the call graph (tree). Requires -focus; no default feature.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/AlexsanderHamir/Performance-Hub/go/performance/parser"
)

func main() {
	focus := flag.String("focus", "", "isolate call tree to functions whose name contains this (e.g. parser.work); required")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s -focus <substring> <cpu.prof>\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nGenerate a profile: cd go && go test -cpuprofile=cpu.prof -bench=. ./performance/parser/")
	}
	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}
	if *focus == "" {
		fmt.Fprintln(os.Stderr, "error: -focus is required (e.g. -focus \"parser.work\")")
		flag.Usage()
		os.Exit(1)
	}
	path := flag.Arg(0)

	p, err := parser.ParseProfile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	d, err := parser.DigestProfile(p)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	parser.PrintCallGraph(d, *focus, nil)
}
