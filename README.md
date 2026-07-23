# InvestingLive News

> A focused terminal reader for the latest headlines from InvestingLive.

InvestingLive News is a small, fast Go application that reads the public
[InvestingLive news RSS feed](https://investinglive.com/feed/news/) and filters
articles to the time range you care about. Run it interactively for a polished
date-range picker, or pass a duration for scripts and scheduled jobs.

```text
INVESTINGLIVE NEWS
Articles from the last hour
2026-07-21 14:30 (CEST) to 2026-07-21 15:30 (CEST)  |  2 articles

 1  Euro edges higher as markets digest central bank commentary
    Jul 21, 15:18  https://www.investinglive.com/...

 2  US stock futures point to a steady open
    Jul 21, 14:52  https://www.investinglive.com/...
```

## Highlights

- Interactive presets and custom local date/time ranges
- Non-interactive duration arguments for automation
- Newest-first results with publication times and direct article links
- Screen-reader-friendly prompts through `ACCESSIBLE=1`
- Respect for the [`NO_COLOR`](https://no-color.org/) convention
- Request timeouts, response-size limits, and safe terminal rendering
- No accounts, API keys, database, or persistent local data

## Requirements

- Go 1.25 or newer, or [mise](https://mise.jdx.dev/)
- An internet connection to read the RSS feed

## Quick start

With mise, install the pinned toolchain and launch the interactive picker:

```shell
mise install
mise run news
```

With Go directly:

```shell
go run .
```

When standard input or output is redirected, the app automatically uses its
non-interactive mode with a one-hour default window.

## Command-line usage

Pass a friendly minutes-or-hours window:

```shell
go run . 30m
go run . 90min
go run . 2h
go run . -last 1.5hours
```

Non-interactive results include publication timestamps in UTC using RFC3339,
making the output stable across machines and local timezones.

Use another compatible RSS endpoint when testing or integrating:

```shell
go run . -feed https://example.com/news.xml -last 6h
```

See every option with `go run . -help`. Release builds also report their
version with `investinglive-news -version`.

## Accessibility

Set `ACCESSIBLE=1` to replace full-screen controls and animations with linear,
screen-reader-friendly prompts. Set `NO_COLOR=1` to disable color in the result
view.

```shell
ACCESSIBLE=1 go run .
```

In PowerShell:

```powershell
$env:ACCESSIBLE = "1"
go run .
```

## Development

```shell
mise run check
mise run build
```

`check` formats the code, runs static analysis, and executes the test suite.
See [CONTRIBUTING.md](CONTRIBUTING.md) for the full contributor workflow and
[SECURITY.md](SECURITY.md) for vulnerability reporting.

## Scope and disclaimer

The upstream RSS feed contains only its currently published items, so older
custom ranges may return no results even when matching articles exist on the
website. This project is an independent feed reader, is not affiliated with
InvestingLive, and does not provide financial advice.

## License

Released under the [MIT License](LICENSE).
