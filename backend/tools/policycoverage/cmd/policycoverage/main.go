// Command policycoverage runs the Phase 5.1 reconciliation scanner over the
// Orkestra backend and emits a markdown + JSON coverage report. Exit code
// is non-zero when any error-severity diagnostic survives the baseline,
// which CI treats as a build failure.
//
// Usage:
//
//	go run ./tools/policycoverage/cmd/policycoverage \
//	    -baseline=tools/policycoverage/baseline.txt \
//	    -markdown=policy-coverage-report.md \
//	    -json=policy-coverage-report.json \
//	    -cedar=internal/core/authz/cedar/policies \
//	    ./internal/...
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/orkestra/backend/tools/policycoverage"
)

func main() {
	baseline := flag.String("baseline", "", "path to baseline file of accepted drift (category:key per line)")
	markdown := flag.String("markdown", "", "write markdown report to this path (default: stdout)")
	jsonOut := flag.String("json", "", "write JSON report to this path")
	cedarDir := flag.String("cedar", "internal/core/authz/cedar/policies", "directory of .cedar policy files for the informational reconciliation section")
	flag.Parse()
	patterns := flag.Args()
	if len(patterns) == 0 {
		patterns = []string{"./internal/..."}
	}

	baseSet, err := policycoverage.LoadBaseline(*baseline)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}

	findings, err := policycoverage.Scan(patterns, *cedarDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}

	report := policycoverage.Reconcile(findings, baseSet)

	if *markdown == "" {
		if err := policycoverage.WriteMarkdown(os.Stdout, report, findings); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(2)
		}
	} else {
		f, err := os.Create(*markdown)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(2)
		}
		if err := policycoverage.WriteMarkdown(f, report, findings); err != nil {
			f.Close()
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(2)
		}
		f.Close()
	}

	if *jsonOut != "" {
		f, err := os.Create(*jsonOut)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(2)
		}
		if err := policycoverage.WriteJSON(f, report); err != nil {
			f.Close()
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(2)
		}
		f.Close()
	}

	fmt.Fprintf(os.Stderr, "policycoverage: %d errors, %d warnings, %d info\n",
		report.Summary[policycoverage.SeverityError],
		report.Summary[policycoverage.SeverityWarn],
		report.Summary[policycoverage.SeverityInfo],
	)
	if report.HasErrors() {
		os.Exit(1)
	}
}
