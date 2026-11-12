package report_test

import (
	"testing"

	"github.com/awu123-create/http-bench/internal/report"
	"github.com/awu123-create/http-bench/internal/run"
)

func TestNewRunReport(t *testing.T) {
	r, err := report.NewRunReport("run-1")
	if err != nil {
		t.Fatalf("NewRunReport() error = %v", err)
	}
	if r.RunID() != run.RunID("run-1") {
		t.Errorf("RunID() = %q, want run-1", r.RunID())
	}
}

func TestNewRunReportRejectsBlankRunIDs(t *testing.T) {
	for _, id := range []run.RunID{"", " ", "\t\n"} {
		t.Run(string(id), func(t *testing.T) {
			if _, err := report.NewRunReport(id); err == nil {
				t.Fatal("NewRunReport() error = nil, want validation error")
			}
		})
	}
}
