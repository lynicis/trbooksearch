# Search Accuracy Improvements Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Eliminate irrelevant results and wrong editions by adding Turkish-aware text matching, relevance scoring with filtering, and `--author`/`--publisher` edition filtering flags.

**Architecture:** Add a `relevance` package with Turkish text normalization and token-based scoring. Move post-filtering from individual scrapers into a centralized relevance-scoring pass in `engine.go`. Expose `Relevance` on `BookResult`, display it in the TUI, and add CLI flags for author/publisher/threshold filtering.

**Tech Stack:** Go 1.26, no new dependencies (stdlib `unicode`, `strings` only)

---

## Task 1: Create the `relevance` package with Turkish normalization

**Files:**
- Create: `internal/relevance/normalize.go`
- Create: `internal/relevance/normalize_test.go`

### Step 1: Write the failing tests

```go
// internal/relevance/normalize_test.go
package relevance

import "testing"

func TestNormalizeTurkish(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Suç ve Ceza", "suc ve ceza"},
		{"İSTANBUL", "istanbul"},
		{"IŞIK", "isik"},
		{"Güneş", "gunes"},
		{"Öğretmen", "ogretmen"},
		{"Üç", "uc"},
		{"Şeker", "seker"},
		{"çiçek", "cicek"},
		{"  multiple   spaces  ", "multiple spaces"},
		{"", ""},
	}
	for _, tt := range tests {
		got := NormalizeTurkish(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeTurkish(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"Suç ve Ceza", []string{"suc", "ve", "ceza"}},
		{"İstanbul'un Çocukları", []string{"istanbulun", "cocuklari"}},
		{"  ", nil},
		{"one", []string{"one"}},
	}
	for _, tt := range tests {
		got := Tokenize(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("Tokenize(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("Tokenize(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}
```

### Step 2: Run tests to verify they fail

Run: `go test ./internal/relevance/ -v`
Expected: FAIL — package does not exist

### Step 3: Write the implementation

```go
// internal/relevance/normalize.go
package relevance

import (
	"strings"
	"unicode"
)

// turkishReplacements maps Turkish-specific characters to ASCII equivalents.
var turkishReplacements = map[rune]rune{
	'ş': 's', 'Ş': 's',
	'ç': 'c', 'Ç': 'c',
	'ö': 'o', 'Ö': 'o',
	'ü': 'u', 'Ü': 'u',
	'ğ': 'g', 'Ğ': 'g',
	'ı': 'i', 'İ': 'i',
	'â': 'a', 'Â': 'a',
	'î': 'i', 'Î': 'i',
	'û': 'u', 'Û': 'u',
}

// NormalizeTurkish lowercases, replaces Turkish-specific characters with ASCII
// equivalents, strips non-alphanumeric/non-space characters, and collapses whitespace.
func NormalizeTurkish(s string) string {
	// Turkish-aware lowercasing: İ→i, I→ı (but we normalize ı→i too)
	s = strings.ToLowerSpecial(unicode.TurkishCase, s)

	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false

	for _, r := range s {
		if rep, ok := turkishReplacements[r]; ok {
			r = rep
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevSpace = false
		} else if unicode.IsSpace(r) {
			if !prevSpace && b.Len() > 0 {
				b.WriteRune(' ')
				prevSpace = true
			}
		}
		// skip other characters (punctuation, etc.)
	}

	return strings.TrimSpace(b.String())
}

// Tokenize normalizes a string and splits it into lowercase, de-accented word tokens.
func Tokenize(s string) []string {
	normalized := NormalizeTurkish(s)
	if normalized == "" {
		return nil
	}
	return strings.Fields(normalized)
}
```

### Step 4: Run tests to verify they pass

Run: `go test ./internal/relevance/ -v`
Expected: PASS

### Step 5: Commit

```bash
git add internal/relevance/
git commit -m "feat(relevance): add Turkish text normalization and tokenizer"
```

---

## Task 2: Add token-based scoring functions

**Files:**
- Create: `internal/relevance/score.go`
- Create: `internal/relevance/score_test.go`

### Step 1: Write the failing tests

```go
// internal/relevance/score_test.go
package relevance

import (
	"math"
	"testing"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.01
}

func TestTitleScore(t *testing.T) {
	tests := []struct {
		title, query string
		want         float64
	}{
		// exact match
		{"Suç ve Ceza", "Suç ve Ceza", 1.0},
		// title contains all query words + extras
		{"Suç ve Ceza - Dostoyevski", "Suç ve Ceza", 1.0},
		// partial match (2 out of 3 query words)
		{"Suç ve Barış", "Suç ve Ceza", 0.67},
		// no match
		{"Kırmızı Pazartesi", "Suç ve Ceza", 0.0},
		// Turkish normalization: ş/ç handled
		{"Suc ve Ceza", "Suç ve Ceza", 1.0},
		// case-insensitive
		{"SUÇ VE CEZA", "suç ve ceza", 1.0},
		// empty query
		{"Suç ve Ceza", "", 0.0},
		// empty title
		{"", "Suç ve Ceza", 0.0},
	}
	for _, tt := range tests {
		got := TitleScore(tt.title, tt.query)
		if !almostEqual(got, tt.want) {
			t.Errorf("TitleScore(%q, %q) = %.2f, want %.2f", tt.title, tt.query, got, tt.want)
		}
	}
}

func TestFieldMatchScore(t *testing.T) {
	tests := []struct {
		field, query string
		want         float64
	}{
		{"Dostoyevski", "Dostoyevski", 1.0},
		{"Fyodor Dostoyevski", "Dostoyevski", 1.0},
		{"İş Bankası Kültür Yayınları", "İş Bankası", 1.0},
		{"Can Yayınları", "İş Bankası", 0.0},
		{"", "Dostoyevski", 0.0},
		{"Dostoyevski", "", 0.0},
	}
	for _, tt := range tests {
		got := FieldMatchScore(tt.field, tt.query)
		if !almostEqual(got, tt.want) {
			t.Errorf("FieldMatchScore(%q, %q) = %.2f, want %.2f", tt.field, tt.query, got, tt.want)
		}
	}
}

func TestComputeRelevance(t *testing.T) {
	// With only title query
	score := ComputeRelevance("Suç ve Ceza - Dostoyevski", "", "", "Suç ve Ceza", "", "")
	if score < 0.8 {
		t.Errorf("expected high relevance for matching title, got %.2f", score)
	}

	// With title + author match
	score2 := ComputeRelevance("Suç ve Ceza", "Dostoyevski", "", "Suç ve Ceza", "Dostoyevski", "")
	if score2 < score {
		t.Errorf("expected author match to boost score: title-only=%.2f, with-author=%.2f", score, score2)
	}

	// Irrelevant result
	score3 := ComputeRelevance("Python Programlama", "Guido", "Kodlab", "Suç ve Ceza", "", "")
	if score3 > 0.2 {
		t.Errorf("expected low relevance for irrelevant title, got %.2f", score3)
	}
}
```

### Step 2: Run tests to verify they fail

Run: `go test ./internal/relevance/ -v -run "TestTitle|TestField|TestCompute"`
Expected: FAIL — undefined functions

### Step 3: Write the implementation

```go
// internal/relevance/score.go
package relevance

// TitleScore computes how well a result title matches the search query.
// Returns 0.0–1.0 based on what fraction of query tokens appear in the title.
func TitleScore(title, query string) float64 {
	queryTokens := Tokenize(query)
	if len(queryTokens) == 0 {
		return 0.0
	}

	titleTokens := Tokenize(title)
	if len(titleTokens) == 0 {
		return 0.0
	}

	titleSet := make(map[string]bool, len(titleTokens))
	for _, t := range titleTokens {
		titleSet[t] = true
	}

	matched := 0
	for _, q := range queryTokens {
		if titleSet[q] {
			matched++
		}
	}

	return float64(matched) / float64(len(queryTokens))
}

// FieldMatchScore checks if all tokens of filterValue appear in the field.
// Returns 1.0 if all tokens match, 0.0 otherwise. Used for author/publisher matching.
// Returns 0.0 if either argument is empty.
func FieldMatchScore(field, filterValue string) float64 {
	filterTokens := Tokenize(filterValue)
	if len(filterTokens) == 0 {
		return 0.0
	}

	fieldTokens := Tokenize(field)
	if len(fieldTokens) == 0 {
		return 0.0
	}

	fieldSet := make(map[string]bool, len(fieldTokens))
	for _, t := range fieldTokens {
		fieldSet[t] = true
	}

	for _, q := range filterTokens {
		if !fieldSet[q] {
			return 0.0
		}
	}
	return 1.0
}

// ComputeRelevance calculates an overall relevance score (0.0–1.0) for a book result.
//
// Parameters:
//   - title, author, publisher: fields from the BookResult
//   - query: the user's search query
//   - authorFilter, publisherFilter: optional --author/--publisher flag values (empty = not set)
//
// Weights:
//   - Title match: 70% of score (always applied)
//   - Author match: 15% of score (only when --author is set, otherwise redistributes to title)
//   - Publisher match: 15% of score (only when --publisher is set, otherwise redistributes to title)
func ComputeRelevance(title, author, publisher, query, authorFilter, publisherFilter string) float64 {
	titleScore := TitleScore(title, query)

	hasAuthorFilter := authorFilter != ""
	hasPublisherFilter := publisherFilter != ""

	// Dynamic weight redistribution
	titleWeight := 1.0
	authorWeight := 0.0
	publisherWeight := 0.0

	if hasAuthorFilter && hasPublisherFilter {
		titleWeight = 0.70
		authorWeight = 0.15
		publisherWeight = 0.15
	} else if hasAuthorFilter {
		titleWeight = 0.85
		authorWeight = 0.15
	} else if hasPublisherFilter {
		titleWeight = 0.85
		publisherWeight = 0.15
	}

	score := titleScore * titleWeight

	if hasAuthorFilter {
		score += FieldMatchScore(author, authorFilter) * authorWeight
	}
	if hasPublisherFilter {
		score += FieldMatchScore(publisher, publisherFilter) * publisherWeight
	}

	return score
}
```

### Step 4: Run tests to verify they pass

Run: `go test ./internal/relevance/ -v`
Expected: PASS

### Step 5: Commit

```bash
git add internal/relevance/
git commit -m "feat(relevance): add title/field scoring and composite relevance computation"
```

---

## Task 3: Add `Relevance` field to `BookResult`

**Files:**
- Modify: `internal/scraper/scraper.go` (struct only)

### Step 1: Add the field

In `internal/scraper/scraper.go`, add to the `BookResult` struct after the `CargoUnknown` field (line 53):

```go
Relevance    float64  // 0.0–1.0 relevance score computed post-search
```

### Step 2: Verify build

Run: `go build ./...`
Expected: SUCCESS (Relevance defaults to 0.0 everywhere, no breakage)

### Step 3: Commit

```bash
git add internal/scraper/scraper.go
git commit -m "feat(scraper): add Relevance field to BookResult"
```

---

## Task 4: Add `SearchOptions` and centralize relevance filtering in the engine

**Files:**
- Modify: `internal/engine/engine.go`
- Create: `internal/engine/engine_test.go`

### Step 1: Write the failing test

```go
// internal/engine/engine_test.go
package engine

import (
	"testing"

	"github.com/lynicis/trbooksearch/internal/scraper"
)

func TestFilterByRelevance(t *testing.T) {
	results := []scraper.BookResult{
		{Title: "Suç ve Ceza", Relevance: 0.9},
		{Title: "Python Kitabı", Relevance: 0.1},
		{Title: "Suç ve Ceza Özet", Relevance: 0.6},
	}

	filtered := FilterByRelevance(results, 0.3)
	if len(filtered) != 2 {
		t.Errorf("expected 2 results above threshold, got %d", len(filtered))
	}
	for _, r := range filtered {
		if r.Relevance < 0.3 {
			t.Errorf("result %q has relevance %.2f, below threshold 0.3", r.Title, r.Relevance)
		}
	}
}

func TestScoreAndFilterResults(t *testing.T) {
	results := []scraper.BookResult{
		{Title: "Suç ve Ceza", Author: "Dostoyevski", Publisher: "Can"},
		{Title: "Mutfak Sırları", Author: "Arda", Publisher: "Alfa"},
		{Title: "Suç ve Ceza - Özel Baskı", Author: "Fyodor Dostoyevski", Publisher: "İş Bankası"},
	}

	opts := SearchOptions{
		Query:           "Suç ve Ceza",
		AuthorFilter:    "",
		PublisherFilter: "",
		MinRelevance:    0.3,
	}

	scored := ScoreAndFilterResults(results, opts)
	// "Mutfak Sırları" should be filtered out
	for _, r := range scored {
		if r.Title == "Mutfak Sırları" {
			t.Errorf("irrelevant result %q should have been filtered", r.Title)
		}
	}
	// Remaining results should have Relevance > 0
	for _, r := range scored {
		if r.Relevance <= 0 {
			t.Errorf("result %q has zero relevance", r.Title)
		}
	}
}
```

### Step 2: Run tests to verify they fail

Run: `go test ./internal/engine/ -v`
Expected: FAIL — undefined functions

### Step 3: Write the implementation

Add the following to `internal/engine/engine.go`. Add `"github.com/lynicis/trbooksearch/internal/relevance"` to the imports.

```go
// SearchOptions holds query and filter parameters for relevance scoring.
type SearchOptions struct {
	Query           string
	SearchType      scraper.SearchType
	AuthorFilter    string  // from --author flag, empty = not set
	PublisherFilter string  // from --publisher flag, empty = not set
	MinRelevance    float64 // minimum relevance threshold (0.0-1.0)
}

// ScoreAndFilterResults computes relevance scores for all results and
// filters out those below the minimum relevance threshold.
func ScoreAndFilterResults(results []scraper.BookResult, opts SearchOptions) []scraper.BookResult {
	for i := range results {
		results[i].Relevance = relevance.ComputeRelevance(
			results[i].Title,
			results[i].Author,
			results[i].Publisher,
			opts.Query,
			opts.AuthorFilter,
			opts.PublisherFilter,
		)
	}
	return FilterByRelevance(results, opts.MinRelevance)
}

// FilterByRelevance removes results below the given relevance threshold.
func FilterByRelevance(results []scraper.BookResult, threshold float64) []scraper.BookResult {
	if threshold <= 0 {
		return results
	}
	filtered := make([]scraper.BookResult, 0, len(results))
	for _, r := range results {
		if r.Relevance >= threshold {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
```

Then update the `Search` method signature from:

```go
func (e *Engine) Search(ctx context.Context, query string, searchType scraper.SearchType, statusCh chan<- SiteStatus) SearchResult {
```

to:

```go
func (e *Engine) Search(ctx context.Context, opts SearchOptions, statusCh chan<- SiteStatus) SearchResult {
```

Inside `Search`, replace `query` with `opts.Query` and `searchType` with `opts.SearchType` in the goroutine's `s.Search()` call.

After `wg.Wait()` and before the price sort, add:

```go
results = ScoreAndFilterResults(results, opts)
```

### Step 4: Run tests to verify they pass

Run: `go test ./internal/engine/ -v`
Expected: PASS

### Step 5: Commit

```bash
git add internal/engine/
git commit -m "feat(engine): add relevance scoring and filtering to search pipeline"
```

---

## Task 5: Remove `MatchesQuery` from individual scrapers

**Files:**
- Modify: `internal/scraper/sites/nadirkitap.go` — remove `MatchesQuery` call
- Modify: `internal/scraper/sites/trendyol.go` — remove `MatchesQuery` call
- Modify: `internal/scraper/sites/dolap.go` — remove `MatchesQuery` call
- Modify: `internal/scraper/scraper.go` — remove `MatchesQuery` function

Since the engine now handles relevance filtering centrally, per-scraper filtering is redundant.

### Step 1: Remove MatchesQuery calls

In **nadirkitap.go** ~line 96: Remove the `if !scraper.MatchesQuery(result.Title, query)` guard. Always append the result.

In **trendyol.go** ~line 72: Remove the `!scraper.MatchesQuery(name, query)` condition from the skip check. Keep only the `name == ""` check.

In **dolap.go** ~line 99: Remove the `MatchesQuery` check on `displayTitle` and `productURL`. Always append results.

In **scraper.go** lines 190-194: Delete the `MatchesQuery` function entirely.

### Step 2: Verify build

Run: `go build ./...`
Expected: SUCCESS (may fail if engine.go signature change hasn't propagated — see Task 6)

### Step 3: Verify no remaining references

Run: `grep -r "MatchesQuery" internal/`
Expected: No output

### Step 4: Commit

```bash
git add internal/scraper/
git commit -m "refactor: remove per-scraper MatchesQuery, relevance filtering is now centralized"
```

---

## Task 6: Add `--author`, `--publisher`, `--min-relevance` CLI flags and thread through TUI

**Files:**
- Modify: `cmd/search.go`
- Modify: `internal/tui/model.go`
- Modify: `internal/tui/update.go`
- Modify: `internal/tui/view.go` (only `m.query` → `m.searchOpts.Query` references)

### Step 1: Add flag variables to `cmd/search.go`

Add to the `var` block:

```go
flagAuthor       string
flagPublisher    string
flagMinRelevance float64
```

Add to `init()`:

```go
searchCmd.Flags().StringVar(&flagAuthor, "author", "", "Yazar adına göre filtrele")
searchCmd.Flags().StringVar(&flagPublisher, "publisher", "", "Yayınevi adına göre filtrele")
searchCmd.Flags().Float64Var(&flagMinRelevance, "min-relevance", 0.3, "Minimum alaka düzeyi eşiği (0.0-1.0)")
```

### Step 2: Update `runSearch` in `cmd/search.go`

Replace the direct query/searchType usage with:

```go
opts := engine.SearchOptions{
	Query:           query,
	SearchType:      searchType,
	AuthorFilter:    flagAuthor,
	PublisherFilter: flagPublisher,
	MinRelevance:    flagMinRelevance,
}

model := tui.NewModel(eng, opts, !flagFlat, ctx)
```

### Step 3: Update `internal/tui/model.go`

Replace `query string` and `searchType scraper.SearchType` fields with `searchOpts engine.SearchOptions`.

Update `NewModel` signature:

```go
func NewModel(eng *engine.Engine, opts engine.SearchOptions, grouped bool, ctx context.Context) Model {
```

Store `searchOpts: opts` in the returned Model. Remove the old `query` and `searchType` fields.

### Step 4: Update `internal/tui/update.go`

- Replace `m.query` → `m.searchOpts.Query`
- Replace `m.searchType` → `m.searchOpts.SearchType`
- In `runSearch` function, change the call to:
  ```go
  result := eng.Search(ctx, opts, statusCh)
  ```
  And update the function signature to accept `engine.SearchOptions` instead of separate `query`/`searchType`.

### Step 5: Update `internal/tui/view.go`

Replace all `m.query` references with `m.searchOpts.Query` (in the `View()` header rendering and `renderSearching` method).

### Step 6: Verify build

Run: `go build ./...`
Expected: SUCCESS

### Step 7: Commit

```bash
git add cmd/search.go internal/tui/ internal/engine/
git commit -m "feat: add --author/--publisher/--min-relevance flags, thread SearchOptions through pipeline"
```

---

## Task 7: Add relevance column to the TUI table

**Files:**
- Modify: `internal/tui/view.go`

### Step 1: Add relevance column constants

Add to the column width constants (after `colNote`):

```go
colRelevance = 6
```

Add to the column index constants (after `colIdxTotal`):

```go
colIdxRelevance = 7
```

### Step 2: Update `renderTable` header

Change the header format string to include the relevance column between "Toplam" and "Not":

```go
header := fmt.Sprintf("  %-*s %-*s %-*s %-*s %*s %*s %*s %*s %-*s",
	colSite, "Site",
	colTitle, "Başlık",
	colAuthor, "Yazar",
	colSeller, "Satıcı",
	colPrice, "Fiyat",
	colCargo, "Kargo",
	colTotal, "Toplam",
	colRelevance, "Alaka",
	colNote, "Not",
)
```

### Step 3: Update `renderTable` row rendering

After the `total` variable, add:

```go
rel := fmt.Sprintf("%%%d", int(r.Relevance*100))
var relStyled string
switch {
case r.Relevance >= 0.8:
	relStyled = priceStyle.Render(fmt.Sprintf("%*s", colRelevance, rel))
case r.Relevance >= 0.5:
	relStyled = warningStyle.Render(fmt.Sprintf("%*s", colRelevance, rel))
default:
	relStyled = dimStyle.Render(fmt.Sprintf("%*s", colRelevance, rel))
}
```

Add `relStyled` to the row format string (between total and note).

### Step 4: Update `applySort` for relevance column

Add to the switch in `applySort`:

```go
case colIdxRelevance:
	cmp = compareFloat(sorted[i].Relevance, sorted[j].Relevance)
```

### Step 5: Update `matchesFilter` for relevance column

Add:

```go
case colIdxRelevance:
	return match(fmt.Sprintf("%d", int(r.Relevance*100)))
```

### Step 6: Update column name arrays and key bindings

In `View()` and the key handler sections, update `colNames` slices:

```go
colNames := []string{"Site", "Başlık", "Yazar", "Satıcı", "Fiyat", "Kargo", "Toplam", "Alaka"}
```

Add `"8"` to the key handler case for column-specific filter:

```go
case "1", "2", "3", "4", "5", "6", "7", "8":
```

Add `colIdxRelevance` to the sort column cycle in the `s` key handler:

```go
sortCols := []int{-1, colIdxSite, colIdxTitle, colIdxAuthor, colIdxSeller, colIdxTotal, colIdxRelevance}
```

### Step 7: Update footer help text

Change `1-7 sütun` to `1-8 sütun` in the footer.

### Step 8: Verify build

Run: `go build ./...`
Expected: SUCCESS

### Step 9: Commit

```bash
git add internal/tui/view.go
git commit -m "feat(tui): add relevance column with color-coded display and sort/filter support"
```

---

## Task 8: Integration verification

### Step 1: Run all tests

Run: `go test ./... -v`
Expected: ALL PASS

### Step 2: Build

Run: `go build -o trbooksearch .`
Expected: SUCCESS

### Step 3: Manual smoke tests

Test 1 — basic search:
```bash
./trbooksearch search "Suç ve Ceza"
```
Expected: Results with "Alaka" column, irrelevant results filtered out.

Test 2 — author filter:
```bash
./trbooksearch search "Suç ve Ceza" --author "Dostoyevski"
```
Expected: Dostoyevski editions score higher.

Test 3 — publisher filter:
```bash
./trbooksearch search "Suç ve Ceza" --publisher "İş Bankası"
```
Expected: İş Bankası editions score higher.

Test 4 — strict threshold:
```bash
./trbooksearch search "Suç ve Ceza" --min-relevance 0.8
```
Expected: Only close matches shown.

Test 5 — disable filtering:
```bash
./trbooksearch search "Suç ve Ceza" --min-relevance 0
```
Expected: All results shown (same as old behavior), relevance column still visible.

### Step 4: Final commit

```bash
git add -A
git commit -m "test: verify search accuracy improvements end-to-end"
```

---

## Summary of all files changed

| File | Action | Purpose |
|------|--------|---------|
| `internal/relevance/normalize.go` | Create | Turkish text normalization and tokenizer |
| `internal/relevance/normalize_test.go` | Create | Tests for normalization |
| `internal/relevance/score.go` | Create | Title/field/composite scoring functions |
| `internal/relevance/score_test.go` | Create | Tests for scoring |
| `internal/scraper/scraper.go` | Modify | Add `Relevance` field, remove `MatchesQuery` |
| `internal/engine/engine.go` | Modify | Add `SearchOptions`, `ScoreAndFilterResults`, update `Search` signature |
| `internal/engine/engine_test.go` | Create | Tests for engine scoring/filtering |
| `internal/scraper/sites/nadirkitap.go` | Modify | Remove `MatchesQuery` call |
| `internal/scraper/sites/trendyol.go` | Modify | Remove `MatchesQuery` call |
| `internal/scraper/sites/dolap.go` | Modify | Remove `MatchesQuery` call |
| `cmd/search.go` | Modify | Add `--author`, `--publisher`, `--min-relevance` flags |
| `internal/tui/model.go` | Modify | Replace query/searchType with `SearchOptions` |
| `internal/tui/update.go` | Modify | Use `SearchOptions` for search dispatch |
| `internal/tui/view.go` | Modify | Add relevance column, color coding, sort/filter |
