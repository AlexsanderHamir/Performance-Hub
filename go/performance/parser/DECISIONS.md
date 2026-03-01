# Decisions

## Use pprof’s profile types as-is

**Context:** We need to parse pprof profile files and expose analytical data (sample types, top functions, call edges).

**Options considered:** (1) Define our own structs and map from pprof after parse. (2) Use `github.com/google/pprof/profile` for parsing and types; add only a thin `Digest` and helpers on top.

**Decision:** Use pprof’s `Profile`, `Location`, `Function`, `Sample`, `ValueType` everywhere. The parser returns `*profile.Profile` from `ParseProfile`/`ParseProfileFromReader`; `Digest` holds `Profile` and adds derived slices (`SampleTypes`, `TopFunctions`, `Edges`) and metadata. No redefinition of sample or location semantics.

**Tradeoffs:** Dependency on pprof and its versioning. In return, we stay aligned with the format and avoid drift or duplicate validation logic.

---

## Digest as a separate step

**Context:** Callers need aggregated views (top functions, edges, call tree) from a profile.

**Options considered:** (1) Single function that reads path and returns a digest. (2) Parse → `Profile`, then `DigestProfile(Profile)` → `Digest`.

**Decision:** Two steps: `ParseProfile`/`ParseProfileFromReader` return `*profile.Profile`; `DigestProfile(p)` returns `*Digest`. The same `Profile` can be digested once and printed multiple times with different focus, or used for other analysis without forcing a digest.

**Tradeoffs:** One extra call in the typical “parse and print” path. Benefit: clearer separation and reuse.

---

## Call graph from edges only

**Context:** We need to walk the call graph (roots, children) for tree printing and focus filtering.

**Options considered:** (1) Store a full graph (nodes + edges). (2) Build a caller-keyed map from `[]CallEdge`; roots = callers that are never callees; children = `EdgesFrom(caller)`.

**Decision:** `CallGraph` is built only from `[]CallEdge` (e.g. `Digest.Edges`). No explicit node type. Roots computed as “callers that never appear as callee” when focus is empty; when focus is set, roots are all functions whose name contains the focus string. Edges and totals are derived from the same slice.

**Tradeoffs:** Cycles in the profile show up as repeated nodes in the tree; we detect them with a `visited` set and print “(cycle)”. No extra structure for “all nodes” beyond what we need for roots and edges.
