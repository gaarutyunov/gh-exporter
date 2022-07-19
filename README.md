# gh-exporter

CLI tool for exporting GitHub repositories based on [GitHub repositories search query](https://docs.github.com/en/search-github/searching-on-github/searching-for-repositories)

## Usage

```shell
Usage:
  gh_exporter [flags]

Flags:
  -h, --help              help for gh_exporter
  -i, --identity string   SSH key path (default "~/.ssh/id_rsa")
  -o, --out string        Output directory (default "~/repos/python")
  -p, --pattern string    Cloning file name pattern (default "*.py")
  -q, --query string      GItHub repos search query (default "q=language:python")
```
