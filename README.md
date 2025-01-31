# gh-exporter

CLI tool for exporting GitHub repositories based on [GitHub repositories search query](https://docs.github.com/en/search-github/searching-on-github/searching-for-repositories)

## Installation

You can install the package using go:

```bash
go install github.com/gaarutyunov/gh-exporter@latest
```

## Usage

### Prerequisites

1. You will need to configure [GitHub token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens) in a `GITHUB_TOKEN` environment variable for authorization
2. And an [ssh key](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/adding-a-new-ssh-key-to-your-github-account) attached to your account in GitHub for cloning without a limit. The path to the key should be specified with the `--identity` option for `export` command (see in instruction below).

### Search

First, you need to search for repositories you want to export. You can use the following command to search for repositories:

```bash
gh-exporter search --out results.csv
```

The `results.csv` file will contain the search results. You can use the `--query` option to specify the search query.

You can try out with 100 repositories by using the `--limit` option:

```bash
gh-exporter search --out results.csv --limit 100
```

To see all available options, run:

```bash
gh-exporter search --help
```

### Plan

After you have the search results, you can plan the export using the following command:

```bash
gh-exporter plan --in results.csv --out plan.csv --capacity 1073741824
```

It will split the repositories into chunks of 1GB and save the plan to the `plan.csv` file.

To see all available options, run:

```bash
gh-exporter plan --help
```

### Export

Finally, you can export the repositories using the following command:

```bash
gh-exporter export --in plan.csv --out raw_repos --pattern "*.py"
```

It will clone the repositories to the `repos` directory using the `plan.csv` file by chunks.
It will only clone files that match the `*.py` pattern.

You can use the `--concurrency` option to specify the number of concurrent downloads.

Also, you can try in memory cloning to speed up and save disk space by using the `--in-memory` option.:

```bash
gh-exporter export --in plan.csv --out raw_repos --in-memory
```

But be aware that it might consume a lot of memory for repositories with a lot of commit history.

Also, don't forget to specify the path to your SSH key with the `--identity` option.

```bash
gh-exporter export --in plan.csv --out raw_repos --identity ~/.ssh/gh_rsa --pattern "*.py"
```

To see all available options, run:

```bash
gh-exporter export --help
```
