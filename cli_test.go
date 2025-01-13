package main

import (
	"github.com/gaarutyunov/gh-exporter/plan"
	"github.com/gaarutyunov/gh-exporter/utils"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestSearch_WithLimit(t *testing.T) {
	cmd := rootCmd

	outFile := filepath.Join(t.TempDir(), "results.csv")

	limit := 10

	cmd.SetArgs([]string{
		"search",
		"--limit", strconv.Itoa(limit),
		"--out", outFile,
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatal(err)
	}

	open, err := os.Open(outFile)
	if err != nil {
		t.Fatal(err)
	}

	counter, err := utils.LineCounter(open)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, limit, counter)
}

func TestSearch_WithLimitAndPagination(t *testing.T) {
	cmd := rootCmd
	outFile := filepath.Join(t.TempDir(), "results.csv")
	limit := 150

	cmd.SetArgs([]string{
		"search",
		"--limit", strconv.Itoa(limit),
		"--out", outFile,
		"--burst", "2", // need to set burst to 2 to avoid being rate limited for search during tests
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatal(err)
	}

	open, err := os.Open(outFile)
	if err != nil {
		t.Fatal(err)
	}

	counter, err := utils.LineCounter(open)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, limit, counter)
}

func TestPlan(t *testing.T) {
	cmd := rootCmd
	outFile := filepath.Join("testdata", "results.csv")
	planFile := filepath.Join(t.TempDir(), "plan.csv")

	cmd.SetArgs([]string{
		"plan",
		"--in", outFile,
		"--out", planFile,
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatal(err)
	}

	expectedPlanFile := filepath.Join("testdata", "plan.csv")

	expected, err := os.ReadFile(expectedPlanFile)
	if err != nil {
		t.Fatal(err)
	}

	actual, err := os.ReadFile(planFile)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, string(expected), string(actual))
}

func TestExport_SkipRemainder(t *testing.T) {
	planFile := filepath.Join("testdata", "plan_small.csv")
	outDir := t.TempDir()

	cmd := rootCmd
	cmd.SetArgs([]string{
		"export",
		"--file", planFile,
		"--out", outDir,
		"--skip-remainder",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatal(err)
	}

	fi, err := plan.Open(planFile)
	if err != nil {
		t.Fatal(err)
	}

	total := fi.Total(true, false)

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, total, len(entries))
}
