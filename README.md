# gh-exporter

CLI tool for exporting GitHub repositories based on [GitHub repositories search query](https://docs.github.com/en/search-github/searching-on-github/searching-for-repositories)

## Usage

### Search

First, you need to search for repositories you want to export. You can use the following command to search for repositories:

```bash
gh_exporter search --out results.csv
```

The `results.csv` file will contain the search results. You can use the `--query` option to specify the search query.

You can try out with 100 repositories by using the `--limit` option:

```bash
gh_exporter search --out results.csv --limit 100
```

To see all available options, run:

```bash
gh_exporter search --help
```

### Plan

After you have the search results, you can plan the export using the following command:

```bash
gh_exporter plan --in results.csv --out plan.csv --capacity 1073741824
```

It will split the repositories into chunks of 1GB and save the plan to the `plan.csv` file.

To see all available options, run:

```bash
gh_exporter plan --help
```

### Export

Finally, you can export the repositories using the following command:

```bash
gh_exporter export --in plan.csv --out repos --pattern "*.py"
```

It will clone the repositories to the `repos` directory using the `plan.csv` file by chunks.
It will only clone files that match the `*.py` pattern.

You can use the `--concurrency` option to specify the number of concurrent downloads.

Also, you can try in memory cloning to speed up and save disk space by using the `--in-memory` option.:

```bash
gh_exporter export --in plan.csv --out repos --in-memory
```

But be aware that it might consume a lot of memory for repositories with a lot of commit history.

To see all available options, run:

```bash
gh_exporter export --help
```