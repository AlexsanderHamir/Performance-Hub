# Performance Hub — Design

## Goal

A single place for **automations** (performance-focused and beyond) that can be written in **any language**. Each automation is self-contained, easy to find, and easy to extend.

## Layout

```
Performance Hub/
├── README.md
├── docs/
│   └── DESIGN.md          # This file
├── go/
│   ├── go.mod             # Go module (all Go code under go/ uses this)
│   ├── performance/       # Go performance automations
│   └── ...                # Other categories (e.g. go/reliability/)
├── python/
│   ├── performance/
│   └── ...
└── [other languages]/     # e.g. node/, rust/
    ├── performance/
    └── ...
```

## Rules

1. **One language = one top-level folder**  
   `go/`, `python/`, `node/`, etc. No mixing inside a folder.

2. **One category = one subfolder per language**  
   `performance/` is the main one; add others as needed (e.g. `reliability/`, `security/`).

3. **One automation = one subfolder (or script) inside a category**  
   e.g. `go/performance/benchmark-runner/`, `python/performance/profiler-parse/`.  
   Each has its own README, deps, and entrypoint so it can be run independently.

4. **Dependencies stay local**  
   - Go: use the `go.mod` in `go/`; run and build from the `go/` directory.  
   - Python: `requirements.txt` (or similar) inside the automation folder.  
   - Node: `package.json` inside the automation folder.

5. **Discovery**  
   - README at repo root lists languages and categories.  
   - Each automation folder has a short README: what it does, how to run it, and any env/args.

## Adding a new automation

1. Pick language → e.g. `go/`.
2. Pick category → e.g. `performance/`.
3. Create a folder → e.g. `go/performance/my-automation/`.
4. Add code, README, and dependency file(s).
5. Optionally add a one-line entry in the root README under the right language/category.

## Adding a new language

1. Add a top-level folder: e.g. `python/`, `node/`.
2. Add at least one category, e.g. `python/performance/`.
3. Document the language in the root README and in `docs/DESIGN.md` (optional).

This keeps the repo **flat, predictable, and easy to navigate** while scaling to many languages and automation types.
