package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/codetasker/backend/internal/debt"
)

func main() {
	if len(os.Args) < 3 || os.Args[1] != "debt" || os.Args[2] != "analyze" {
		printUsage()
		os.Exit(2)
	}

	flags := flag.NewFlagSet("codetaskers debt analyze", flag.ExitOnError)
	repo := flags.String("repo", ".", "local git repository path")
	days := flags.Int("days", 90, "number of days of git history to analyze")
	hourlyCost := flags.Float64("hourly-cost", 35, "hourly engineer cost in USD")
	if err := flags.Parse(os.Args[3:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	result, err := debt.AnalyzeLocalRepo(context.Background(), debt.Options{
		Repo:       *repo,
		Days:       *days,
		HourlyCost: *hourlyCost,
		Now:        time.Now().UTC(),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "debt analyze failed: %v\n", err)
		os.Exit(1)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "encode output failed: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: codetaskers debt analyze --repo ./my-project --days 90 --hourly-cost 35")
}
