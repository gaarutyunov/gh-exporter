# gh-exporter

CLI tool for exporting GitHub repositories based on [GitHub repositories search query](https://docs.github.com/en/search-github/searching-on-github/searching-for-repositories)

## Usage

1. Explore repositories to be exported later

```shell
Usage:
  gh_exporter search [flags]

Flags:
  -h, --help           help for search
  -o, --out string     Search results file (default "~/git-py/results.csv")
  -q, --query string   GItHub repos search query (default "q=language:python")
```

2. Create export plan. This command uses bin packing algorithm to group repositories for further optimized cloning

```shell
Usage:
  gh_exporter plan [flags]

Flags:
  -c, --capacity uint   Repository group capacity (default 2147483648)
  -h, --help            help for plan
  -i, --in string       Search results input for planning (default "~/git-py/results.csv")
  -o, --out string      Plan file path (default "~/git-py/plan.csv")
```

3. Export planned repositories

```shell
Usage:
    gh_exporter export [flags]

Flags:
  -c, --concurrency int   Cloning concurrency (default 10)
  -f, --file string       Plan file path (default "~/git-py/plan.csv")
  -h, --help              help for export
  -i, --identity string   SSH key path (default "~/.ssh/id_rsa")
  -o, --out string        Output directory (default "~/git-py/repos/python")
  -p, --pattern string    Cloning file name pattern (default "*.py")
  -s, --search string     Search results file path to determine total (default "~/git-py/results.csv")
```
