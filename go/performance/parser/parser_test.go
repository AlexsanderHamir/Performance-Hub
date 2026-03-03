package parser

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/pprof/profile"
)

func minimalProfile(callerName, calleeName string, sampleValue int64) *profile.Profile {
	fnCaller := &profile.Function{ID: 1, Name: callerName, Filename: "caller.go"}
	fnCallee := &profile.Function{ID: 2, Name: calleeName, Filename: "callee.go"}
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
			Location: []*profile.Location{locCallee, locCaller},
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

	t.Run("returns profile when file contains valid profile", func(t *testing.T) {
		p := minimalProfile("caller", "callee", 100)
		dir := t.TempDir()
		path := filepath.Join(dir, "cpu.prof")
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := p.Write(f); err != nil {
			f.Close()
			t.Fatal(err)
		}
		f.Close()
		got, err := ParseProfile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected non-nil profile")
		}
		if len(got.Sample) == 0 {
			t.Error("expected at least one sample")
		}
	})

	t.Run("returns error when file exists but content is invalid", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.prof")
		if err := os.WriteFile(path, []byte("not a profile"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := ParseProfile(path)
		if err == nil {
			t.Fatal("expected error for invalid profile content")
		}
	})
}

func TestParseProfileFromReader(t *testing.T) {
	t.Run("returns error when reader contains invalid data", func(t *testing.T) {
		_, err := ParseProfileFromReader(bytes.NewReader([]byte("not a profile")))
		if err == nil {
			t.Fatal("expected error for invalid profile data")
		}
	})

	t.Run("returns profile when reader contains valid profile", func(t *testing.T) {
		p := minimalProfile("caller", "callee", 100)
		var buf bytes.Buffer
		if err := p.Write(&buf); err != nil {
			t.Fatal(err)
		}
		got, err := ParseProfileFromReader(&buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected non-nil profile")
		}
		if len(got.Sample) == 0 {
			t.Error("expected at least one sample")
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

	t.Run("returns error when profile fails CheckValid", func(t *testing.T) {
		p := minimalProfile("caller", "callee", 100)
		p.Sample[0].Value = []int64{1, 2}
		_, err := DigestProfile(p)
		if err == nil {
			t.Fatal("expected error for invalid profile")
		}
	})

	t.Run("leaves PeriodType empty when profile PeriodType is nil", func(t *testing.T) {
		p := minimalProfile("caller", "callee", 100)
		p.PeriodType = nil
		d, err := DigestProfile(p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if d.PeriodType != "" {
			t.Errorf("PeriodType: got %q, want empty", d.PeriodType)
		}
	})

	t.Run("sorts multiple edges by value descending", func(t *testing.T) {
		// 3 locations (root→mid→leaf) yield 2 edges so DigestProfile's sort.Slice callback is exercised.
		fnRoot := &profile.Function{ID: 1, Name: "root", Filename: "r.go"}
		fnMid := &profile.Function{ID: 2, Name: "mid", Filename: "m.go"}
		fnLeaf := &profile.Function{ID: 3, Name: "leaf", Filename: "l.go"}
		locRoot := &profile.Location{ID: 1, Line: []profile.Line{{Function: fnRoot}}}
		locMid := &profile.Location{ID: 2, Line: []profile.Line{{Function: fnMid}}}
		locLeaf := &profile.Location{ID: 3, Line: []profile.Line{{Function: fnLeaf}}}
		p := &profile.Profile{
			SampleType:    []*profile.ValueType{{Type: "cpu", Unit: "nanoseconds"}},
			PeriodType:    &profile.ValueType{Type: "cpu", Unit: "nanoseconds"},
			Period:        10000000,
			DurationNanos: 1e9,
			TimeNanos:     0,
			Function:      []*profile.Function{fnRoot, fnMid, fnLeaf},
			Location:      []*profile.Location{locRoot, locMid, locLeaf},
			Sample: []*profile.Sample{
				{Location: []*profile.Location{locLeaf, locMid, locRoot}, Value: []int64{10}},
			},
		}
		d, err := DigestProfile(p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(d.Edges) != 2 {
			t.Fatalf("expected 2 edges, got %d", len(d.Edges))
		}
		if d.Edges[0].Value < d.Edges[1].Value {
			t.Errorf("edges should be sorted descending by value, got %d then %d", d.Edges[0].Value, d.Edges[1].Value)
		}
	})

}

func TestSplitEdgeKey(t *testing.T) {
	t.Run("returns caller and callee when key contains newline", func(t *testing.T) {
		caller, callee := splitEdgeKey("a\nb")
		if caller != "a" || callee != "b" {
			t.Errorf("got %q, %q; want a, b", caller, callee)
		}
	})
	t.Run("returns empty strings when key has no newline", func(t *testing.T) {
		caller, callee := splitEdgeKey("noNewline")
		if caller != "" || callee != "" {
			t.Errorf("got %q, %q; want empty", caller, callee)
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

	t.Run("returns multiple roots sorted by value when focus matches several", func(t *testing.T) {
		edges := []CallEdge{
			{Caller: "foox", Callee: "foo", Value: 5},
			{Caller: "fooy", Callee: "foo", Value: 10},
		}
		cg := BuildCallGraph(edges)
		roots := cg.Roots("foo")
		if len(roots) != 3 {
			t.Fatalf("got %d roots, want 3 (foox, fooy, foo)", len(roots))
		}
		if roots[0] != "fooy" || roots[1] != "foox" {
			t.Errorf("expected first two sorted by value (fooy=10, foox=5), got %v", roots)
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

	t.Run("uses stdout when writer is nil", func(t *testing.T) {
		p := minimalProfile("caller", "callee", 100)
		d, _ := DigestProfile(p)
		PrintDigest(d, "", nil) // must not panic
	})

	t.Run("skips call graph section when digest has no edges", func(t *testing.T) {
		d, _ := DigestProfile(minimalProfile("a", "b", 10))
		d.Edges = nil
		var buf bytes.Buffer
		PrintDigest(d, "", &buf)
		if strings.Contains(buf.String(), "Call graph") {
			t.Error("output should not contain call graph when Edges is empty")
		}
	})

	t.Run("writes call graph with focused header when focus matches", func(t *testing.T) {
		d, _ := DigestProfile(minimalProfile("caller", "callee", 10))
		var buf bytes.Buffer
		PrintDigest(d, "caller", &buf)
		if !strings.Contains(buf.String(), "Call graph (tree, focused on \"caller\")") {
			t.Error("output should contain focused call graph header when focus matches")
		}
	})

	t.Run("writes no function matching when focus matches nothing", func(t *testing.T) {
		d, _ := DigestProfile(minimalProfile("caller", "callee", 10))
		var buf bytes.Buffer
		PrintDigest(d, "nonexistentFunc", &buf)
		if !strings.Contains(buf.String(), "No function matching") {
			t.Error("output should contain no-match message when focus matches no function")
		}
	})
}

func TestPrintCallGraph(t *testing.T) {
	t.Run("writes only call graph section to writer", func(t *testing.T) {
		d, _ := DigestProfile(minimalProfile("caller", "callee", 10))
		var buf bytes.Buffer
		PrintCallGraph(d, "caller", &buf)
		out := buf.String()
		if !strings.Contains(out, "Call graph (tree, focused on \"caller\")") {
			t.Error("output should contain focused call graph header")
		}
		if strings.Contains(out, "Parsed profile") || strings.Contains(out, "Top functions") {
			t.Error("output should not contain digest header or top functions")
		}
	})

	t.Run("writes nothing when digest has no edges", func(t *testing.T) {
		d, _ := DigestProfile(minimalProfile("a", "b", 10))
		d.Edges = nil
		var buf bytes.Buffer
		PrintCallGraph(d, "a", &buf)
		if buf.Len() != 0 {
			t.Errorf("expected no output when Edges is empty, got %d bytes", buf.Len())
		}
	})

	t.Run("uses stdout when writer is nil", func(t *testing.T) {
		d, _ := DigestProfile(minimalProfile("caller", "callee", 10))
		PrintCallGraph(d, "caller", nil) // must not panic
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

	t.Run("uses treeBranch for last and non-last sibling", func(t *testing.T) {
		branch, next := treeBranch(true, "  ")
		if branch != "└─ " || next != "      " {
			t.Errorf("treeBranch(true): got %q, %q", branch, next)
		}
		branch, next = treeBranch(false, "  ")
		if branch != "├─ " || !strings.HasSuffix(next, "│   ") {
			t.Errorf("treeBranch(false): got %q, %q", branch, next)
		}
	})

	t.Run("prints cycle when graph has caller-callee cycle", func(t *testing.T) {
		edges := []CallEdge{
			{Caller: "A", Callee: "B", Value: 10},
			{Caller: "B", Callee: "A", Value: 10},
		}
		cg := BuildCallGraph(edges)
		var buf bytes.Buffer
		PrintCallTree(cg, []string{"A"}, 100, false, &buf)
		if !strings.Contains(buf.String(), "(cycle)") {
			t.Error("output should contain (cycle) when call graph has a cycle")
		}
	})
}

func TestParseAndDigestIntegration(t *testing.T) {
	p := minimalProfile("main", "handler", 200)
	dir := t.TempDir()
	path := filepath.Join(dir, "cpu.prof")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Write(f); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	gotProfile, err := ParseProfile(path)
	if err != nil {
		t.Fatalf("ParseProfile: %v", err)
	}
	gotDigest, err := DigestProfile(gotProfile)
	if err != nil {
		t.Fatalf("DigestProfile: %v", err)
	}

	if gotDigest.TotalSamples != 200 {
		t.Errorf("TotalSamples: got %d, want 200", gotDigest.TotalSamples)
	}
	if len(gotDigest.Edges) != 1 {
		t.Fatalf("Edges: got %d, want 1", len(gotDigest.Edges))
	}
	if gotDigest.Edges[0].Caller != "main" || gotDigest.Edges[0].Callee != "handler" || gotDigest.Edges[0].Value != 200 {
		t.Errorf("Edges[0]: got Caller=%q Callee=%q Value=%d", gotDigest.Edges[0].Caller, gotDigest.Edges[0].Callee, gotDigest.Edges[0].Value)
	}

	var buf bytes.Buffer
	PrintDigest(gotDigest, "", &buf)
	if buf.Len() == 0 {
		t.Error("PrintDigest produced no output")
	}
	if !strings.Contains(buf.String(), "main") || !strings.Contains(buf.String(), "handler") {
		t.Error("PrintDigest output should contain main and handler")
	}
}
