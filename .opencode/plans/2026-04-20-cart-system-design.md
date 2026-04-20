# Cart System Design

**Date:** 2026-04-20
**Status:** Approved

## Goal

Add a persistent, multi-book shopping cart with cargo-aware optimization. The
user collects books over one or many sessions, then runs an optimizer that
finds the cheapest combination of sellers, exploiting the fact that buying
multiple books from the same `(site, seller)` pair pays cargo only once.

This fits the tool's existing price-comparison mission and introduces no new
runtime dependencies.

## Scope

### In scope (v1)

- Persistent cart stored in the XDG config directory alongside `config.yaml`.
- Adding a book (query + all candidate listings) to the cart from the search
  TUI with a single keypress, and from the CLI for scripting.
- Managing the cart via CLI subcommands (`list`, `remove`, `clear`, `refresh`).
- Optimizer that minimises total cost including per-seller cargo, with
  optional constraints (`--max-sellers`, `--new-only`, `--exclude`).
- Loyalty program awareness driven by `config.yaml`.
- Top-N alternative solutions with labelled variants ("Fewest sellers",
  "Single seller") always included.

### Non-goals (v1)

- Actual checkout / cart-to-site integration — sites have no public cart APIs.
- Price-drop alerts or price history (already on the project roadmap; separate
  feature).
- Book-cover thumbnails (roadmap item; unrelated).
- Sharing / syncing carts across devices.
- Free-cargo-threshold modeling (e.g. "free over 150 TL"). `FreeCargo` + loyalty
  is enough for v1.
- Partial-seller loyalty (e.g. Trendyol Elite covering only specific sellers).

## Architecture

New package `internal/cart` owns the data model, persistence, and optimizer.
Pure Go, no I/O beyond the cart file. Testable in isolation.

Touch points in existing code:

- `internal/config` — add `Loyalty` section (amazon_prime, trendyol_elite,
  hepsiburada_premium).
- `internal/tui` — add `a` key in search results to add the current query's
  full result set to the cart; brief footer flash confirmation; dedup overlay.
- `cmd/` — new `cart` command group: `cart list`, `cart add`, `cart remove`,
  `cart clear`, `cart refresh`, `cart optimize`.
- `internal/engine` — no functional changes; optimizer sits above engine.
  `cart refresh` re-runs existing `Engine.Search` for each entry.

```
┌──────────────┐       ┌───────────────┐
│  search TUI  │──'a'─▶│ internal/cart │
└──────────────┘       │               │
                       │  cart.yaml    │
┌──────────────┐       │               │
│  cmd/cart_*  │──────▶│  Load/Save    │
└──────────────┘       │  Optimize     │
                       └───────────────┘
                              │
                              ▼
                       ┌───────────────┐
                       │   Optimizer   │
                       │  (pure func)  │
                       └───────────────┘
```

## Data model

```go
// internal/cart/cart.go (conceptual)

type Cart struct {
    Version int         `yaml:"version"`          // schema version, starts at 1
    Entries []CartEntry `yaml:"entries"`
}

type CartEntry struct {
    ID         string               `yaml:"id"`          // 8-char hex hash
    Query      string               `yaml:"query"`
    SearchType scraper.SearchType   `yaml:"search_type"` // Title / ISBN
    AddedAt    time.Time            `yaml:"added_at"`
    FetchedAt  time.Time            `yaml:"fetched_at"`  // last re-fetch
    Candidates []scraper.BookResult `yaml:"candidates"`
    Note       string               `yaml:"note,omitempty"`
}
```

`Candidates` stores `scraper.BookResult` directly because it already carries
`Site`, `Seller`, `Price`, `CargoFee`, `FreeCargo`, `LoyaltyNote`, `Condition`,
`URL`, `Category`, and `CargoUnknown` — everything the optimizer needs, with
no translation layer.

`ID` is an 8-char hex of a hash over `Query + AddedAt` so `cart remove 3f2a` is
unambiguous even when the user has two entries for the same query (first
edition vs reprint).

## Persistence

- **Location:** `$XDG_CONFIG_HOME/trbooksearch/cart.yaml`, using
  `os.UserConfigDir()` via the same helper pattern as
  `internal/config/config.go`.
- **Format:** YAML.
- **Permissions:** `0600` (cart may imply purchase intent; keep private; same
  as existing `config.Save` in `internal/config/config.go:90`).
- **Atomic write:** write to `cart.yaml.tmp`, then `os.Rename`. Prevents
  corruption on crash mid-write.
- **Missing file:** returns empty `Cart{Version: 1}`, not an error (mirrors
  `config.Load`'s "file missing is fine" behaviour at
  `internal/config/config.go:40`).
- **Version mismatch:** future loader will error on unknown `Version` rather
  than silently ignoring. `yaml.v3` ignores unknown fields by default, so
  adding fields stays backward-compatible.

### Public cart package API

Narrow, side-effect-free except where noted:

```
cart.Load() (Cart, error)                        // read file
cart.Save(c Cart) error                          // atomic write
cart.Add(c *Cart, entry CartEntry)               // append
cart.Remove(c *Cart, id string) bool             // returns true if removed
cart.Clear(c *Cart)
cart.FindByQuery(c Cart, q string) []CartEntry   // for dedup suggestions
```

Helpers mutate `*Cart`; callers do their own `Save`. This keeps the optimizer
and its tests pure.

### Dedup on add

When adding a book whose `Query` (case-insensitive, trimmed) matches an
existing entry:

- TUI: overlay prompt with `E` (replace / update candidates + bump FetchedAt),
  `N` (add as new entry), `Esc` (cancel).
- CLI (`cart add`): flags `--replace`, `--new-entry`, or default to the TUI
  prompt when stdin is a TTY. `--yes` is an alias for `--replace`.

## Cargo model

The optimizer treats cargo as paid once per **`(Site, Seller)`** bundle, not
per site and not per item. This matches reality for marketplaces like
Trendyol, Hepsiburada, and Amazon where each seller ships independently, and
degenerates correctly to "per site" for single-seller sites like
kitapyurdu.com.

When multiple candidate listings in a cart share the same `(Site, Seller)`
key, the cargo of the cheapest listing from that seller is used as the
bundle's cargo cost.

### Loyalty

Config additions:

```yaml
loyalty:
  amazon_prime: true
  trendyol_elite: false
  hepsiburada_premium: false
```

At optimizer normalise time, for each listing:

```
if listing.FreeCargo                                                    → effective_cargo = 0
elif listing.Site == "amazon.com.tr" && loyalty.amazon_prime            → effective_cargo = 0
elif listing.Site == "trendyol.com" && loyalty.trendyol_elite           → effective_cargo = 0
elif listing.Site == "hepsiburada.com" && loyalty.hepsiburada_premium   → effective_cargo = 0
else                                                                    → effective_cargo = listing.CargoFee
```

Known limitation (documented, not fixed in v1): Trendyol Elite and similar
programs sometimes don't cover third-party sellers. This modeling is
site-coarse.

### CargoUnknown handling

Candidates with `CargoUnknown: true` are stored but **skipped** in the
optimizer. The optimizer output surfaces a summary warning:

```
Warning: 2 listings skipped due to unknown cargo.
```

## Optimizer

### Problem statement

Given cart entries `E = [e₁, ..., e_N]`, each with candidate listings
`e_i.Candidates`, find an assignment `a: book → listing` minimising:

```
total(a) = Σ price(a(e_i))  +  Σ cargo_per_seller(s)
          i                    s∈S
```

where `S = { (a(e_i).Site, a(e_i).Seller) : i = 1..N }`.

### Constraints

All AND'd when supplied:

- `--max-sellers N`: `|S| ≤ N`.
- `--new-only`: only consider listings where `Category == NewBook`.
- `--exclude site1,site2`: drop candidates from those sites.
- Loyalty: from config, as described above.

### Algorithm

**Phase 1 — normalise.** For each entry, produce a filtered candidate list:
apply `--new-only`, `--exclude`, drop `CargoUnknown`, apply loyalty to
cargo. If any entry ends up with zero candidates, return an error naming the
entry.

**Phase 2 — group.** For each book, group candidates by `(Site, Seller)` and
keep only the cheapest listing per seller. Sellers dominated within their own
seller for a given book are never optimal.

**Phase 3 — enumerate seller subsets.** Let `U = ⋃ᵢ sellers_i` be the
universe of sellers appearing in any book. For each subset `S ⊆ U` that
covers all books:

```
cost(S) = Σᵢ min_{s ∈ S ∩ sellers_i} price_i(s)  +  Σ_{s∈S used} cargo(s)
```

A seller `s ∈ S` is "used" iff it wins `argmin` for at least one book. After
choosing the argmins, drop unused sellers from `S` before computing cargo.

Prunes:

1. Skip subsets larger than `--max-sellers`.
2. Branch-and-bound: sort sellers by "max price saving enabled" descending;
   prune when a lower bound on the remaining subset ≥ best-known cost.

**Fallback.** If `|U| > 22`, fall back to beam search (keep top-K partial
subsets by lower bound) and emit a warning: *"large cart (22+ sellers), using
heuristic; may miss optimum."* For realistic personal carts, this path will
almost never trigger.

**Phase 4 — top-N collection.** Maintain a size-`K+1` max-heap keyed by
`cost(S)` during enumeration, where `K` is the user-requested N (default 5).
Return the heap sorted.

**Phase 5 — labelled variants.** Always include (if a solution exists):

- *Single-seller minimum*: best `|S| = 1`.
- *Fewest sellers*: minimum `|S|`, ties broken by cost.

These are labelled and appear in output even if not in the top N.

### Optimizer output

```go
type OptimizationResult struct {
    Solutions []Solution
    Warnings  []string
}

type Solution struct {
    Label        string                          // "", "Fewest sellers", "Single seller"
    Assignment   map[string]scraper.BookResult   // entry.ID -> chosen listing
    Bundles      []Bundle
    Total        float64
    Subtotal     float64
    CargoTotal   float64
    SellerCount  int
}

type Bundle struct {
    Site, Seller string
    Books        []string    // entry IDs in this bundle
    Subtotal     float64
    Cargo        float64
}
```

Pure function: `Optimize(entries []CartEntry, cfg Config, opts OptimizeOptions)
(OptimizationResult, error)`.

## CLI

All wired in `cmd/` alongside existing `root.go`, `search.go`,
`setapikey.go`. Structure: `cmd/cart.go` for the `cart` parent command, then
one file per subcommand (`cart_list.go`, `cart_add.go`, ...).

### `trbooksearch cart list`

Non-interactive table to stdout. Columns: `ID`, `Query`, `Candidates`,
`Cheapest`, `Added`, `Fetched` (with amber colour when older than 7 days).

Flags:

- `--json` — machine-readable output.
- `--show-candidates <id>` — expand one entry's candidate listings.

### `trbooksearch cart add <query>`

Non-interactive add path. Runs `engine.Search`, stores all candidates. Honours
existing search flags (`--isbn`, `--sites`, `--exclude`, `--firecrawl`,
`--limit`). Dedup flags: `--replace`, `--new-entry`, `--yes`.

### `trbooksearch cart remove <id>...`

Removes one or more entries by ID. `cart remove --all` aliases `cart clear`
and requires `--yes`.

### `trbooksearch cart clear`

Empties the cart. Prompts unless `--yes`.

### `trbooksearch cart refresh [<id>...]`

Re-runs `engine.Search` for each entry (all, or selected IDs), updates
`Candidates` and `FetchedAt`. Progress printed sequentially to stdout — lighter
and more scriptable than reusing the full search TUI. One progress line per
entry.

**Failure semantics:** if refresh for an entry returns zero candidates or a
hard error, keep the existing `Candidates` untouched and do **not** bump
`FetchedAt`. Warn the user. The optimizer continues to work with
stale-but-present data.

### `trbooksearch cart optimize`

Runs `cart.Optimize`. Output is an interactive Bubble Tea TUI reusing the
lipgloss conventions of `internal/tui/view.go`:

```
Top Solutions                               (↑↓ select, Enter details, o open all, q quit)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  #1                 3 sellers   7 books   Subtotal 540 TL + Cargo 45 TL = 585 TL
  #2  Fewest sellers 1 seller    7 books   Subtotal 620 TL + Cargo 15 TL = 635 TL
  #3                 2 sellers   7 books   Subtotal 550 TL + Cargo 30 TL = 580 TL
  #4  Single-seller  1 seller    7 books   Subtotal 620 TL + Cargo 15 TL = 635 TL
  #5                 3 sellers   7 books   Subtotal 555 TL + Cargo 40 TL = 595 TL
  #6                 4 sellers   7 books   Subtotal 535 TL + Cargo 60 TL = 595 TL

Selected: #1
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Bundle 1: kitapyurdu.com / kitapyurdu (Cargo 15 TL)
  • 1984 — George Orwell                                          89.00 TL  [→]
  • Simyacı — Paulo Coelho                                        72.00 TL  [→]
  • Dune — Frank Herbert                                         124.00 TL  [→]

Bundle 2: trendyol.com / Kitap Sahaf (Cargo 15 TL)
  • Suç ve Ceza — Dostoyevski                                     45.00 TL  [→]  İkinci El

Bundle 3: amazon.com.tr / Amazon (Prime — free)
  • Sapiens — Yuval Harari                                       110.00 TL  [→]

Warnings:
  • 2 listings skipped (unknown cargo)
```

Flags:

- `--top N` (default 5)
- `--max-sellers N`
- `--new-only`
- `--exclude site1,site2`
- `--json` — non-interactive structured output (disables TUI).

Keys: `↑↓` pick a solution, `Enter` open each book's URL (OSC 8), `o` open
all URLs in the selected solution, `q` quit.

## TUI integration in `search`

In `internal/tui/update.go` normal-mode key handler (alongside `/`, `1-8`,
`s`, `r`), add:

- **`a`** — add current query to cart. Grabs `m.searchOpts.Query`,
  `m.searchOpts.SearchType`, and **all unfiltered** `m.results.Results`.
  Wraps into a `CartEntry`, then `cart.Load → cart.Add → cart.Save`. Shows a
  2-second footer flash: `✓ added 23 listings for '1984' (cart: 3 entries)`.
  On dedup hit, enters a small confirmation overlay (`E` / `N` / `Esc`).

Rationale for "all unfiltered": current filter/sort in the TUI is only a
viewing aid; maximising the optimizer's choice space matches the user's
original intent.

Flash notification: add a `flashMsg` with `text` and `expireAt` to `Model`;
clear via a deferred command similar to the existing `tickMsg` cadence.

## Config additions

`internal/config/config.go` gains:

```go
type Config struct {
    Firecrawl FirecrawlConfig `yaml:"firecrawl"`
    Loyalty   LoyaltyConfig   `yaml:"loyalty"`
}

type LoyaltyConfig struct {
    AmazonPrime        bool `yaml:"amazon_prime"`
    TrendyolElite      bool `yaml:"trendyol_elite"`
    HepsiburadaPremium bool `yaml:"hepsiburada_premium"`
}
```

Backward compatible: missing `loyalty:` block yields all `false`. Existing
configs keep working.

Helper: `cfg.Loyalty.FreeCargoForSite(site string) bool` centralises the
site → program mapping so the optimizer doesn't hardcode strings.

## File layout

```
internal/
  cart/
    cart.go           # Cart, CartEntry, Load, Save, Add, Remove, Clear
    cart_test.go
    optimize.go       # Optimize, Solution, Bundle, normalise, enumeration
    optimize_test.go
    render.go         # TUI render helpers for cart optimize view
  config/
    config.go         # +Loyalty fields
    config_test.go    # +loyalty round-trip
  tui/
    update.go         # +'a' key, flash message
    model.go          # +flash state
    view.go           # +flash renderer
cmd/
  cart.go             # parent command
  cart_list.go
  cart_add.go
  cart_remove.go
  cart_clear.go
  cart_refresh.go
  cart_optimize.go
  cart_test.go
```

## Error and edge-case matrix

| Situation | Handling |
|---|---|
| `cart.yaml` missing | empty `Cart{Version: 1}`, no file created until first Save |
| `cart.yaml` has unknown `Version` | error: "cart file version N not supported" |
| `cart.yaml` malformed YAML | error surfacing path + parse error |
| Atomic write tmp leak | next write overwrites; log warning if detected |
| Refresh: zero results or hard error for entry | keep old candidates, warn, don't bump `FetchedAt` |
| Optimize: empty cart | error, exit 1 |
| Optimize: entry with no viable candidates | error naming the entry, exit 1 |
| Dedup: same-query entry exists | TUI overlay / CLI flags as above |
| Loyalty config conflicts with scraper `FreeCargo: true` | honour per-listing `FreeCargo: true` |

## Testing strategy

Follows the project's existing table-driven conventions from
`internal/tui/tui_test.go`, `internal/engine/engine_test.go`, and
`internal/config/config_test.go`.

**`internal/cart/cart_test.go`:**

- Load from nonexistent path → empty Cart.
- Save then Load round-trip.
- Atomic write survives simulated mid-write error; temp file cleanup.
- `Add` / `Remove` / `Clear` state transitions.
- Dedup detection by case-insensitive trimmed query match.
- Version mismatch returns error.

**`internal/cart/optimize_test.go`** — pure logic, aim ≥ 90% coverage:

- Single-book cart → picks cheapest listing.
- Two books, same seller cheapest → bundles; cargo once.
- Two books, different sellers cheapest → two cargo payments; verify vs.
  bundle alternative.
- Loyalty-covered site makes bundling via that site cheaper → optimizer picks
  it.
- `--max-sellers 1` forces single-seller solution even if more expensive.
- `--new-only` filters used out.
- `--exclude amazon.com.tr` drops candidates.
- CargoUnknown skipping + warning.
- Entry with no viable candidates → error.
- Top-N sorted; alternatives labelled.
- Large synthetic cart (10 books × 30 candidates) completes in < 100 ms.
- Pathological universe size triggers beam-search fallback; verify warning.

**`cmd/cart_test.go`:** subcommand smoke tests hitting a file-backed cart
using `t.TempDir()`. Add a `TRBOOKSEARCH_CONFIG_DIR` env override if one
doesn't exist yet, following the pattern `internal/config` uses via
`loadFrom` / `saveTo`.

**TUI:** extend `internal/tui/tui_test.go`:

- `a` key on search results updates the cart file.
- Dedup overlay shows and handles `E` / `N` / `Esc`.
- Flash message appears and clears on tick.

## Documentation

Add a `## Cart` section to `README.md`:

- When to use cart vs. plain search.
- Setting up loyalty programs in `config.yaml`.
- Example workflow: add two books, run `cart optimize --max-sellers 2`.
- Limitations (freshness, CargoUnknown, loyalty granularity).

## Rollout

Single PR. The feature is additive — no existing flag, command, or file
format changes. Users who don't interact with `cart` see no behavioural
difference.
