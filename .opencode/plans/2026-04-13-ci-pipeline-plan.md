# CI Pipeline Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add GitHub Actions CI (lint/test/build on push/PR) and a release workflow (GoReleaser on GitHub Release publish) to the trbooksearch project.

**Architecture:** Two separate GitHub Actions workflows — `ci.yml` for quality gates on every push/PR, and `release.yml` triggered by GitHub Release `published` events that uses GoReleaser to cross-compile binaries for 6 targets (linux/macOS/windows x amd64/arm64) and attach them to the release. A version variable is injected via ldflags at build time.

**Tech Stack:** GitHub Actions, GoReleaser v2, golangci-lint v2, Go 1.26.x

---

### Task 1: Add version variable to main.go

**Files:**
- Modify: `main.go`

**Step 1: Add version/commit/date variables**

Add ldflags-injectable variables to `main.go`:

```go
package main

import "trbooksearch/cmd"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, date)
	cmd.Execute()
}
```

**Step 2: Add version command to cmd/root.go**

Add a `SetVersionInfo` function and a `version` subcommand:

```go
var (
	appVersion = "dev"
	appCommit  = "none"
	appDate    = "unknown"
)

func SetVersionInfo(version, commit, date string) {
	appVersion = version
	appCommit = commit
	appDate = date
}
```

Add a version command to `init()`:

```go
func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Sürüm bilgilerini göster",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("trbooksearch %s (commit: %s, built: %s)\n", appVersion, appCommit, appDate)
		},
	})
}
```

**Step 3: Verify it compiles**

Run: `go build -o trbooksearch . && ./trbooksearch version`
Expected: `trbooksearch dev (commit: none, built: unknown)`

**Step 4: Verify ldflags injection works**

Run: `go build -ldflags "-X main.version=test -X main.commit=abc -X main.date=now" -o trbooksearch . && ./trbooksearch version`
Expected: `trbooksearch test (commit: abc, built: now)`

---

### Task 2: Create `.goreleaser.yml`

**Files:**
- Create: `.goreleaser.yml`

**Step 1: Write the GoReleaser config**

```yaml
version: 2

before:
  hooks:
    - go mod tidy

builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}

archives:
  - format: tar.gz
    name_template: >-
      {{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: "checksums.txt"

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^ci:"
```

**Step 2: Verify config is valid**

Run: `go install github.com/goreleaser/goreleaser/v2@latest && goreleaser check`
Expected: no errors

**Step 3: Test a snapshot build**

Run: `goreleaser build --snapshot --clean`
Expected: binaries created in `dist/` for all 6 targets

---

### Task 3: Create `.github/workflows/ci.yml`

**Files:**
- Create: `.github/workflows/ci.yml`

**Step 1: Write the CI workflow**

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

permissions:
  contents: read

jobs:
  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: golangci/golangci-lint-action@v7
        with:
          version: latest

  test:
    name: Test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: go test -race ./...

  build:
    name: Build
    runs-on: ubuntu-latest
    needs: [lint, test]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: go build -v ./...
```

---

### Task 4: Create `.github/workflows/release.yml`

**Files:**
- Create: `.github/workflows/release.yml`

**Step 1: Write the release workflow**

```yaml
name: Release

on:
  release:
    types: [published]

permissions:
  contents: write

jobs:
  goreleaser:
    name: Build & Release
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

### Task 5: Update `.gitignore`

**Files:**
- Modify: `.gitignore`

**Step 1: Add GoReleaser dist directory**

Add `dist/` to `.gitignore` so snapshot builds don't get committed.

---

### Task 6: Verify everything locally

**Step 1: Verify Go build**

Run: `go build -v ./...`
Expected: clean compilation

**Step 2: Verify GoReleaser snapshot**

Run: `goreleaser build --snapshot --clean`
Expected: 6 binaries in `dist/`

**Step 3: Verify version command**

Run: `./dist/trbooksearch_darwin_arm64/trbooksearch version`
Expected: version output with snapshot info
