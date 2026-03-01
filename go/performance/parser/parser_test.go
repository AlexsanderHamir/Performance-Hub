package parser

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/pprof/profile"
)

// minimalProfile returns a valid *profile.Profile with one sample and a caller->callee stack.
// Used to test DigestProfile and related behavior without filesystem.
func minimalProfile(callerName, calleeName string, sampleValue int64) *profile.Profile {
	fnCaller := &profile.Function{ID: 1, Name: callerName}
	fnCallee := &profile.Function{ID: 2, Name: calleeName}
	locCaller := &profile.Location{ID: 1, Line: []profile.Line{{Function: fnCaller}}}
	locCallee := &profile.Location{ID: 2, Line: []profile.Line{{Function: fnCallee}}}
	return &profile.Profile{
		SampleType:    []*profile.ValueType{{Type: "cpu", Unit: "nanoseconds"}},
		PeriodType:    &profile.ValueType{Type: "cpu", Unit: "nanoseconds"},
		Period:        10000000,
		DurationNanos: 1e9,
		TimeNanos:     0,
		Function:      []*profile.Function{fnCaller, fnCallee},
		Location:      []*profile.Location{locCaller, locCallee},
		Sample: []*profile.Sample{{
			Location: []*profile.Location{locCallee, locCaller}, // callee at 0, caller at 1 (stack)
			Value:    []int64{sampleValue},
		}},
	}
}

func TestParseProfile(t *testing.T) {
	t.Run("returns error when path does not exist", func(t *testing.T) {
		_, err := ParseProfile("/nonexistent/path/to/cpu.prof")
		if err == nil {
			t.Fatal("expected error for nonexistent path")
		}
	})
}

func TestDigestProfile(t *testing.T) {
	t.Run("returns digest with sample types and total when profile has one sample", func(t *testing.T) {
		p := minimalProfile("caller", "callee", 100)
		d, err := DigestProfile(p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(d.SampleTypes) == 0 {
			t.Error("expected at least one sample type")
		}
		if d.TotalSamples != 100 {
			t.Errorf("TotalSamples: got %d, want 100", d.TotalSamples)
		}
	})

	t.Run("returns digest with one call edge when profile has caller-callee stack", func(t *testing.T) {
		p := minimalProfile("caller", "callee", 50)
		d, err := DigestProfile(p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(d.Edges) != 1 {
			t.Fatalf("expected 1 edge, got %d", len(d.Edges))
		}
		if d.Edges[0].Caller != "caller" || d.Edges[0].Callee != "callee" || d.Edges[0].Value != 50 {
			t.Errorf("edge: got Caller=%q Callee=%q Value=%d", d.Edges[0].Caller, d.Edges[0].Callee, d.Edges[0].Value)
		}
	})

	t.Run("returns digest with top functions populated", func(t *testing.T) {
		p := minimalProfile("caller", "callee", 100)
		d, err := DigestProfile(p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(d.TopFunctions) == 0 {
			t.Error("expected at least one top function")
		}
	})

}

func TestFunctionName(t *testing.T) {
	t.Run("returns function name when location has line with function", func(t *testing.T) {
		fn := &profile.Function{Name: "myFunc"}
		loc := &profile.Location{Line: []profile.Line{{Function: fn}}}
		got := FunctionName(loc)
		if got != "myFunc" {
			t.Errorf("got %q, want myFunc", got)
		}
	})

	t.Run("returns empty string when location has no lines", func(t *testing.T) {
		loc := &profile.Location{Line: nil}
		got := FunctionName(loc)
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("returns empty string when line has nil function", func(t *testing.T) {
		loc := &profile.Location{Line: []profile.Line{{Function: nil}}}
		got := FunctionName(loc)
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

func TestIsTimeProfile(t *testing.T) {
	t.Run("returns true when sample type is cpu/nanoseconds", func(t *testing.T) {
		d := &Digest{SampleTypes: []string{"samples/count", "cpu/nanoseconds"}}
		if !IsTimeProfile(d) {
			t.Error("expected true for cpu/nanoseconds")
		}
	})

	t.Run("returns false when sample type is not time-based", func(t *testing.T) {
		d := &Digest{SampleTypes: []string{"alloc_space/bytes"}}
		if IsTimeProfile(d) {
			t.Error("expected false for alloc_space")
		}
	})
}

func TestFormatDuration(t *testing.T) {
	t.Run("formats nanoseconds as seconds string", func(t *testing.T) {
		got := FormatDuration(3_590_000_000)
		if !strings.Contains(got, "3.59") {
			t.Errorf("got %q, expected to contain 3.59", got)
		}
	})
}

func TestFormatValue(t *testing.T) {
	t.Run("returns seconds string when showSeconds is true", func(t *testing.T) {
		got := FormatValue(2_000_000_000, true)
		if !strings.HasSuffix(got, "s") {
			t.Errorf("got %q, expected suffix s", got)
		}
	})

	t.Run("returns integer string when showSeconds is false", func(t *testing.T) {
		got := FormatValue(100, false)
		if got != "100" {
			t.Errorf("got %q, want 100", got)
		}
	})
}

func TestBuildCallGraph(t *testing.T) {
	t.Run("builds graph from edges", func(t *testing.T) {
		edges := []CallEdge{
			{Caller: "a", Callee: "b", Value: 10},
			{Caller: "a", Callee: "c", Value: 5},
		}
		cg := BuildCallGraph(edges)
		if cg == nil {
			t.Fatal("expected non-nil CallGraph")
		}
		if cg.TotalFrom("a") != 15 {
			t.Errorf("TotalFrom(a): got %d, want 15", cg.TotalFrom("a"))
		}
	})
}

func TestCallGraph_Roots(t *testing.T) {
	edges := []CallEdge{
		{Caller: "root", Callee: "child", Value: 10},
	}
	cg := BuildCallGraph(edges)

	t.Run("returns top-of-stack callers when focus is empty", func(t *testing.T) {
		roots := cg.Roots("")
		if len(roots) != 1 || roots[0] != "root" {
			t.Errorf("got roots %v, want [root]", roots)
		}
	})

	t.Run("returns matching functions when focus is substring", func(t *testing.T) {
		roots := cg.Roots("child")
		if len(roots) != 1 || roots[0] != "child" {
			t.Errorf("got roots %v, want [child]", roots)
		}
	})

	t.Run("returns nil when focus matches no function", func(t *testing.T) {
		roots := cg.Roots("nonexistent")
		if roots != nil {
			t.Errorf("got %v, want nil", roots)
		}
	})
}

func TestCallGraph_EdgesFrom(t *testing.T) {
	t.Run("returns edges sorted by value descending", func(t *testing.T) {
		edges := []CallEdge{
			{Caller: "a", Callee: "b", Value: 5},
			{Caller: "a", Callee: "c", Value: 10},
		}
		cg := BuildCallGraph(edges)
		from := cg.EdgesFrom("a")
		if len(from) != 2 {
			t.Fatalf("got %d edges, want 2", len(from))
		}
		if from[0].Value != 10 || from[1].Value != 5 {
			t.Errorf("expected first edge value 10, second 5; got %d, %d", from[0].Value, from[1].Value)
		}
	})
}

func TestPrintDigest(t *testing.T) {
	t.Run("writes header and sample types to writer", func(t *testing.T) {
		p := minimalProfile("caller", "callee", 100)
		d, _ := DigestProfile(p)
		var buf bytes.Buffer
		PrintDigest(d, "", &buf)
		out := buf.String()
		if !strings.Contains(out, "Parsed profile (digest)") {
			t.Error("output should contain digest header")
		}
		if !strings.Contains(out, "Sample types:") {
			t.Error("output should contain sample types")
		}
	})

	t.Run("writes top functions section", func(t *testing.T) {
		p := minimalProfile("caller", "callee", 100)
		d, _ := DigestProfile(p)
		var buf bytes.Buffer
		PrintDigest(d, "", &buf)
		if !strings.Contains(buf.String(), "Top functions by sample value") {
			t.Error("output should contain top functions section")
		}
	})

	t.Run("uses stdout when writer is nil", func(t *testing.T) {
		p := minimalProfile("caller", "callee", 100)
		d, _ := DigestProfile(p)
		PrintDigest(d, "", nil) // must not panic
	})
}

func TestPrintCallTree(t *testing.T) {
	t.Run("writes nothing when roots is empty", func(t *testing.T) {
		cg := BuildCallGraph(nil)
		var buf bytes.Buffer
		PrintCallTree(cg, []string{}, 100, false, &buf)
		if buf.Len() != 0 {
			t.Errorf("expected no output, got %d bytes", buf.Len())
		}
	})

	t.Run("writes root and branch lines when roots given", func(t *testing.T) {
		edges := []CallEdge{{Caller: "root", Callee: "child", Value: 50}}
		cg := BuildCallGraph(edges)
		var buf bytes.Buffer
		PrintCallTree(cg, []string{"root"}, 100, false, &buf)
		out := buf.String()
		if !strings.Contains(out, "root") {
			t.Error("output should contain root name")
		}
		if !strings.Contains(out, "child") {
			t.Error("output should contain callee name")
		}
	})
}
