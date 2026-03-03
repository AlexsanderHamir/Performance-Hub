# Task: Remove Parser Features
Created: 03/02/26 17:14 PST
Branch: task/03-02-2026_5-14PM-remove-parser-features

> Create and switch to the task branch first. All changes for this task must be made on that branch.

## Instruction
What needs to be done in plain language. (Describe the code quality work here.)

1. Remove the `Top functions by sample value` feature.
2. The parser CLI should not have a default feature, the only one we support as of right now is `Call graph (tree, focused on "parser.work")`, users should always have to specify what features they want to use.

## Context
Any additional background, links, or constraints.

## Acceptance Criteria
- [x] All tests must pass.
- [x] Any existing test for the functionality removed should be removed as well.
- [x] Code coverage should not go down from 100%.

## Commits

### 03/02/26 17:35 PST
`feat(parser): remove Top functions by sample value feature`

### 03/02/26 17:36 PST
`feat(parser): CLI require -focus, add PrintCallGraph; update docs`

### 03/02/26 17:42 PST
`chore(tasks): close 03-02-2026_5-14PM-remove-parser-features`

## Agent Notes
<!-- Agent appends entries here — oldest at top, newest at bottom. Never delete. -->

### 03/02/26 17:20 PST
STATUS: started
BRANCH: task/03-02-2026_5-14PM-remove-parser-features
Read task: remove "Top functions by sample value" (Digest.TopFunctions, aggregateTopFunctions, printTopFunctions); make CLI require explicit -focus so only supported feature is call graph (tree, focused). No default—users must specify -focus. Traced parser package and cmd/parser/main.go. Proceeding with implementation.

### 03/02/26 17:38 PST
STATUS: completed
SUMMARY: Removed Top functions feature (Digest.TopFunctions, FuncStat, aggregateTopFunctions, printTopFunctions); removed related tests. Added PrintCallGraph; CLI now requires -focus and calls PrintCallGraph only. Updated README and DECISIONS. All tests pass, coverage 100%. Two atomic commits.
PR: https://github.com/AlexsanderHamir/Performance-Hub/pull/2
