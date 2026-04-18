// Command tenantscope is the standalone runner for the tenantscope analyzer.
// It uses go/analysis's singlechecker driver so it can be invoked like a
// dedicated vet tool:
//
//	go run ./tools/tenantscope/cmd/tenantscope ./internal/addons/...
//
// A non-zero exit code indicates one or more findings, which the CI job
// treats as a build failure.
package main

import (
	"github.com/orkestra/backend/tools/tenantscope"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(tenantscope.Analyzer)
}
