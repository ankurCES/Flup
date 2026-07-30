<div align="center">

```
 ███████╗██╗     ██╗   ██╗██████╗ 
 ██╔════╝██║     ██║   ██║██╔══██╗
 █████╗  ██║     ██║   ██║██████╔╝
 ██╔══╝  ██║     ██║   ██║██╔═══╝ 
 ██║     ███████╗╚██████╔╝██║     
 ╚═╝     ╚══════╝ ╚═════╝ ╚═╝     
```

# **Flup**

### A beautiful TUI HTTP benchmarking tool

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue?style=for-the-badge)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/ankurCES/Flup/ci.yml?style=for-the-badge&logo=github&logoColor=white)](https://github.com/ankurCES/Flup/actions)
[![Release](https://img.shields.io/github/v/release/ankurCES/Flup?style=for-the-badge&logo=github&logoColor=white&color=F59E0B)](https://github.com/ankurCES/Flup/releases)
[![Go Report Card](https://img.shields.io/badge/Go_Report-A+-brightgreen?style=for-the-badge&logo=go&logoColor=white)](https://goreportcard.com/report/github.com/ankurCES/Flup)
[![GoDoc](https://img.shields.io/badge/godoc-reference-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://pkg.go.dev/github.com/ankurCES/Flup)

**Flup** is a terminal-first HTTP benchmarking tool inspired by [plow](https://github.com/six-ddc/plow) and powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea). Point it at a URL, watch the requests fly, and explore latency distributions — all without leaving your terminal.

</div>

---

## ✨ Features

- 🎯 **Run mode** — pick a URL, connections, method, headers, and bodies right inside the TUI
- 📊 **Live metrics** — RPS, latency percentiles, bytes/sec, and status codes updated in real time
- 📈 **Histogram** — visualize the latency distribution as the benchmark runs
- 🔥 **Percentile view** — inspect p50 / p75 / p90 / p95 / p99 / p99.9 with one keystroke
- ⚠️ **Errors panel** — every non-2xx response captured with status code, method, and URL
- 💾 **Persistent history** — every run saved to `~/.flup/history.json`; browse and re-run past jobs
- ⚡ **Built on `fasthttp`** — high-throughput client tuned for low-allocation benchmarks
- 🎨 **Bubble Tea UI** — smooth, responsive, mouse-aware terminal interface with a polished look
- 🧰 **Static benchmarks** — JSON, CSV, Markdown, and plain-text report export
- 🖥️ **Embedded test server** — `flup-server` ships for running offline micro-benchmarks

---

## 📦 Install

### Using `go install`

```bash
go install github.com/ankurCES/Flup/cmd/flup@latest
go install github.com/ankurCES/Flup/cmd/bench_server@latest   # optional test target
```

The binary lands in `$GOPATH/bin` (usually `~/go/bin`). Make sure that directory is on your `$PATH`.

### From source

```bash
git clone https://github.com/ankurCES/Flup.git
cd Flup
make build
./bin/flup
```

### Prebuilt binaries

Grab the latest release for your platform from the [Releases page](https://github.com/ankurCES/Flup/releases):

| Platform | Archive |
| --- | --- |
| Linux (amd64 / arm64) | `.tar.gz` |
| macOS (amd64 / arm64) | `.tar.gz` |
| Windows (amd64) | `.zip` |

---

## 🚀 Usage

```
flup
```

You'll land on the **Run** screen — type a URL, set the connection count, pick a method, and hit `Enter` to start.

### TUI Screens

| Screen | What it shows |
| --- | --- |
| **Run** | Configure URL, method, headers, body, connections, duration |
| **Live** | Real-time RPS, latency, throughput, and status code counts |
| **Summary** | Final aggregate stats once the benchmark ends |
| **Percentiles** | Per-percentile latency breakdown (p50 → p99.9) |
| **Histogram** | Binned latency distribution, scrolling and zooming |
| **Errors** | Non-2xx responses with status code, method, URL, latency |
| **History** | Browse every previous run; re-run or delete entries |

### ⌨️ Keybindings

| Key | Action |
| --- | --- |
| `Enter` | Start benchmark (Run screen) |
| `Tab` / `Shift+Tab` | Move between fields |
| `↑` / `↓` | Adjust numeric values, scroll lists |
| `1` … `7` | Jump to Run / Live / Summary / Percentiles / Histogram / Errors / History |
| `r` | Re-run the most recent benchmark |
| `e` | Export current results (JSON / CSV / Markdown / text) |
| `?` | Toggle help overlay |
| `q` / `Ctrl+C` | Quit |

---

## ⚖️ Comparison

|  | **Flup** | [plow](https://github.com/six-ddc/plow) | [wrk](https://github.com/wg/wrk) | [hey](https://github.com/rakyll/hey) |
| --- | :---: | :---: | :---: | :---: |
| Interactive TUI | ✅ | ✅ | ❌ | ❌ |
| Live latency / RPS graph | ✅ | ✅ | ❌ | ❌ |
| Percentile breakdown | ✅ | ✅ | ⚠️ plugin | ⚠️ plugin |
| Latency histogram | ✅ | ❌ | ❌ | ❌ |
| Persistent run history | ✅ | ❌ | ❌ | ❌ |
| Runs in pure terminal | ✅ | ✅ | ⚠️ scriptable | ✅ |
| Written in Go | ✅ | ✅ | ❌ (C) | ✅ |
| Embedded test server | ✅ | ❌ | ❌ | ❌ |
| Export JSON / CSV / Markdown | ✅ | ❌ | ❌ | ❌ |
| Zero runtime dependencies | ✅ | ✅ | ⚠️ needs OpenSSL | ✅ |

---

## 🧱 Architecture

```
cmd/
  flup/            # main TUI binary
  bench_server/    # bundled test server
internal/
  bench/           # fasthttp-powered benchmark runner
  tui/             # Bubble Tea models, screens, keymaps
  history/         # on-disk run history
  styles/          # lipgloss themes
```

---

## 🤝 Contributing

PRs welcome. Please run the test suite and linter before submitting:

```bash
make test
make lint
```

Open an [issue](https://github.com/ankurCES/Flup/issues) for bugs, feature requests, or before sending large changes so we can align on direction.

---

## 🙏 Credits

- **[Charmbracelet](https://charmbracelet.com)** — for [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Lip Gloss](https://github.com/charmbracelet/lipgloss), and [Bubbles](https://github.com/charmbracelet/bubbles). The TUI ecosystem is unmatched.
- **[six-ddc/plow](https://github.com/six-ddc/plow)** — the original inspiration. Flup borrows its screen layout and ergonomics, then layers on history, histograms, and exports.
- **[valyala/fasthttp](https://github.com/valyala/fasthttp)** — for the high-performance HTTP client that keeps benchmarks honest.

---

## 📄 License

[Apache-2.0](LICENSE) © 2025 ankurCES
