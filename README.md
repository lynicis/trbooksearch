<div align="center">

# trbooksearch

*Find the best book prices across Turkish online stores — all from your terminal.*

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![CI](https://github.com/lynicis/trbooksearch/actions/workflows/ci.yml/badge.svg)](https://github.com/lynicis/trbooksearch/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/lynicis/trbooksearch?include_prereleases)](https://github.com/lynicis/trbooksearch/releases)

<br />

[Features](#features) •
[Installation](#installation) •
[Usage](#usage) •
[Supported Sites](#supported-sites) •
[Configuration](#configuration) •

</div>

---

## The Problem

You want to buy a book in Turkey. You check **Kitapyurdu**. Then **Amazon**. Then **Trendyol**. Then **Hepsiburada**. Then maybe **nadirkitap** for used copies. By the time you've compared all the prices (including cargo fees!), you've lost 30 minutes of your life.

## The Solution

```bash
trbooksearch search "Suç ve Ceza"
```

**trbooksearch** searches all major Turkish bookstores simultaneously and presents the results in a beautiful, interactive terminal interface — sorted by total price, including shipping costs.

<div align="center">

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Top 3 En Ucuz / Top 3 Cheapest                      │
├─────────────────────────────────────────────────────────────────────────────┤
│  1. nadirkitap.com    Suc ve Ceza - Dostoyevski       45.00 TL  (Ikinci El) │
│  2. kitapyurdu.com    Suc ve Ceza - F. Dostoyevski    89.00 TL  (Yeni)      │
│  3. trendyol.com      Suc ve Ceza                     92.50 TL  (Yeni)      │
└─────────────────────────────────────────────────────────────────────────────┘
```

</div>

---

## Features

- **Multi-Site Search** — Query 8+ Turkish bookstores in parallel
- **Price Comparison** — See book price + cargo fee + **total price** at a glance
- **Used & New Books** — Separate sections for "Ikinci El" and "Yeni" kitaplar
- **Interactive TUI** — Filter, sort, and scroll through results with keyboard shortcuts
- **ISBN Search** — Search by ISBN for exact matches
- **Clickable Links** — Open book pages directly from your terminal (OSC 8 supported)
- **Loyalty Pricing** — See Amazon Prime, Hepsiburada Premium, and Trendyol Elite discounts
- **Stealth Mode** — Anti-bot detection measures built-in
- **Cloud Scraping** — Optional Firecrawl API support for additional sites

---

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/lynicis/trbooksearch.git
cd trbooksearch

# Build
go build -o trbooksearch .

# Move to PATH (optional)
sudo mv trbooksearch /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/lynicis/trbooksearch@latest
```

### Pre-built Binaries

Download the latest release for your platform from the [Releases](https://github.com/lynicis/trbooksearch/releases) page.

| Platform | Architecture | Download |
|----------|--------------|----------|
| Linux    | amd64        | [trbooksearch-linux-amd64.tar.gz](https://github.com/lynicis/trbooksearch/releases/latest) |
| Linux    | arm64        | [trbooksearch-linux-arm64.tar.gz](https://github.com/lynicis/trbooksearch/releases/latest) |
| macOS    | amd64        | [trbooksearch-darwin-amd64.tar.gz](https://github.com/lynicis/trbooksearch/releases/latest) |
| macOS    | arm64 (M1+)  | [trbooksearch-darwin-arm64.tar.gz](https://github.com/lynicis/trbooksearch/releases/latest) |
| Windows  | amd64        | [trbooksearch-windows-amd64.zip](https://github.com/lynicis/trbooksearch/releases/latest) |

### Prerequisites

- **Chrome/Chromium** must be installed for headless browser scraping
- Alternatively, use `--firecrawl` flag with API configuration (no browser needed)

---

## Usage

### Basic Search

```bash
# Search by book title
trbooksearch search "1984"

# Search by ISBN
trbooksearch search --isbn 9789750726439
```

### Advanced Options

```bash
# Limit results per site (default: 10)
trbooksearch search --limit 5 "Simyaci"

# Search only specific sites
trbooksearch search --sites "kitapyurdu.com,nadirkitap.com" "Dune"

# Exclude specific sites
trbooksearch search --exclude "amazon.com.tr" "Harry Potter"

# Flat view (no grouping by Used/New)
trbooksearch search --flat "Sapiens"

# Use Firecrawl API (enables additional sites)
trbooksearch search --firecrawl "Otostopcularin Galaksi Rehberi"
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑` `↓` | Scroll through results |
| `/` | Filter results |
| `1`-`7` | Filter by specific column |
| `s` | Cycle sort column |
| `S` | Toggle sort direction |
| `Enter` | Open book URL (if terminal supports OSC 8) |
| `q` | Quit |

---

## Supported Sites

### New Books (Yeni Kitaplar)

| Site | Method | Notes |
|------|--------|-------|
| [kitapyurdu.com](https://www.kitapyurdu.com) | Browser / Firecrawl | Turkey's largest online bookstore |
| [amazon.com.tr](https://www.amazon.com.tr) | Browser / Firecrawl | Shows Prime pricing |
| [trendyol.com](https://www.trendyol.com) | Browser / Firecrawl | Shows Elite pricing |
| [hepsiburada.com](https://www.hepsiburada.com) | Browser / Firecrawl | Shows Premium pricing |
| [pandora.com.tr](https://pandora.com.tr) | Firecrawl only | Book store |
| [dr.com.tr](https://dr.com.tr) | Firecrawl only | Book store |
| [idefix.com](https://idefix.com) | Browser / Firecrawl | Book store |
| [bkmkitap.com](https://bkmkitap.com) | Browser / Firecrawl | Book store |

### Used Books (Ikinci El Kitaplar)

| Site | Method | Notes |
|------|--------|-------|
| [nadirkitap.com](https://www.nadirkitap.com) | Browser / Firecrawl | Turkey's largest used book marketplace |
| [letgo.com](https://www.letgo.com) | Firecrawl only | Second-hand marketplace |
| [dolap.com](https://www.dolap.com) | Firecrawl only | Second-hand marketplace |
| [gardrops.com](https://www.gardrops.com) | Firecrawl only | Second-hand marketplace |

---

## Configuration

### Config File Location

```
~/.config/trbooksearch/config.yaml
```

### Firecrawl API Setup

To use the `--firecrawl` flag and access additional sites, you need a [Firecrawl](https://firecrawl.dev) API key.

**Quick Setup:**

```bash
trbooksearch set-api-key fc-your-api-key-here
```

Or interactively:

```bash
trbooksearch set-api-key
# Prompts: Firecrawl API anahtarı: _
```

**Manual Setup:**

You can also create the config file manually:

```yaml
# ~/.config/trbooksearch/config.yaml (Linux)
# ~/Library/Application Support/trbooksearch/config.yaml (macOS)

firecrawl:
  api_key: "fc-your-api-key-here"
  api_url: "https://api.firecrawl.dev"  # optional, this is the default
```

---

## How It Works

```
                    ┌─────────────────┐
                    │  trbooksearch   │
                    │     search      │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              ▼              ▼              ▼
       ┌──────────┐   ┌──────────┐   ┌──────────┐
       │ kitapyurdu│   │  amazon  │   │ nadirkitap│  ...
       └─────┬────┘   └─────┬────┘   └─────┬────┘
             │              │              │
             └──────────────┼──────────────┘
                            │
                            ▼
                 ┌─────────────────────┐
                 │   Price Aggregation │
                 │   + Cargo Fees      │
                 └──────────┬──────────┘
                            │
                            ▼
                 ┌─────────────────────┐
                 │   Interactive TUI   │
                 │   (Bubble Tea)      │
                 └─────────────────────┘
```

1. **Parallel Dispatch** — All scrapers launch concurrently with staggered timing
2. **Stealth Scraping** — Random user agents and anti-detection measures
3. **Price Normalization** — Extracts book price, cargo fee, and calculates total
4. **Real-time Updates** — TUI shows progress as each site completes
5. **Smart Sorting** — Results grouped by condition, sorted by total price

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | [Go](https://go.dev/) 1.26+ |
| CLI Framework | [Cobra](https://github.com/spf13/cobra) |
| TUI Framework | [Bubble Tea](https://github.com/charmbracelet/bubbletea) |
| Styling | [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| Web Scraping | [Rod](https://github.com/go-rod/rod) (headless Chrome) |
| HTML Parsing | [goquery](https://github.com/PuerkitoBio/goquery) |
| Cloud Scraping | [Firecrawl](https://firecrawl.dev) API |

---

## Terminal Compatibility

For the best experience with clickable links, use a terminal that supports OSC 8 hyperlinks:

| Terminal | OSC 8 Support |
|----------|---------------|
| iTerm2 | Yes |
| Ghostty | Yes |
| Windows Terminal | Yes |
| GNOME Terminal | Yes |
| Alacritty | Yes |
| Kitty | Yes |
| macOS Terminal.app | No |

---

## Roadmap

- [ ] Price history tracking
- [ ] Notification alerts for price drops
- [ ] Book cover image preview in TUI

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

```
MIT License

Copyright (c) 2026 Emre Sirmali
```

---

## Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the beautiful TUI framework
- [Rod](https://github.com/go-rod/rod) for reliable browser automation
- [Firecrawl](https://firecrawl.dev) for cloud scraping capabilities

---

<div align="center">

**Made with :coffee: in Turkiye**

*If this project saved you time, consider giving it a star!*

</div>
