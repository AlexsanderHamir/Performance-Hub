# Task: Code Quality
Created: 03/01/26 17:51 PST
Branch: task/03-01-2026_5-51PM-code-quality

> Create and switch to the task branch first. All changes for this task must be made on that branch.

## Instruction
Analyze the parser package for bad code quality.

## Context
I want this project to be easy and obvious to maintain, I want to get rid of:

1. Hard to understand logic
2. Reduntant code / logic / patterns
3. Confusing naming

## Acceptance Criteria
- [x] All unit tests must pass.
- [x] Analyze if we have integrations tests and implement the ones that are missing.

## Commits
<!-- Agent appends every commit here as it is made — never delete -->

### 03/01/26 18:05 PST
`refactor(parser): clarify naming and use strings.Cut in splitEdgeKey`

### 03/01/26 18:06 PST
`test(parser): add integration test for ParseProfile to DigestProfile to PrintDigest`

### 03/01/26 18:08 PST
`chore(tasks): close 03-01-2026_5-51PM-code-quality`

## Agent Notes
<!-- Agent appends entries here — oldest at top, newest at bottom. Never delete. -->

### 03/01/26 17:53 PST
STATUS: started
BRANCH: task/03-01-2026_5-51PM-code-quality
Read task: analyze parser for hard-to-understand logic, redundancy, confusing naming; ensure tests pass; add missing integration tests. Traced parser package (parser.go, helpers.go, types.go, parser_test.go, bench_test.go). Proceeding to create branch and analyze.

### 03/01/26 18:07 PST
STATUS: completed
SUMMARY: Refactored parser for clarity: renamed locValue→sampleValueByLocation, edgeValue→valueByEdgeKey, funcValue→valueByFunctionID, showValueSec→showValueInSeconds; replaced manual newline split in splitEdgeKey with strings.Cut; updated README. Added TestParseAndDigestIntegration covering ParseProfile→DigestProfile→PrintDigest with file I/O. All tests pass.
PR: branch task/03-01-2026_5-51PM-code-quality
