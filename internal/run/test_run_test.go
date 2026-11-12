package run_test

import (
	"testing"
	"time"

	"github.com/awu123-create/http-bench/internal/definition"
	"github.com/awu123-create/http-bench/internal/run"
)

func TestNewTestRun(t *testing.T) {
	d := newTestDefinition(t)
	r, err := run.NewTestRun("run-1", d)
	if err != nil {
		t.Fatalf("NewTestRun() error = %v", err)
	}
	if r.ID() != run.RunID("run-1") {
		t.Errorf("ID() = %q, want run-1", r.ID())
	}
	if r.Definition().ID() != d.ID() || r.Definition().Name() != d.Name() {
		t.Errorf("Definition() did not preserve definition identity and name")
	}
}

func TestNewTestRunRejectsInvalidInputs(t *testing.T) {
	d := newTestDefinition(t)
	for _, id := range []run.RunID{"", " ", "\t\n"} {
		t.Run("blank run ID "+string(id), func(t *testing.T) {
			if _, err := run.NewTestRun(id, d); err == nil {
				t.Fatal("NewTestRun() error = nil, want validation error")
			}
		})
	}

	t.Run("zero definition", func(t *testing.T) {
		if _, err := run.NewTestRun("run-1", definition.TestDefinition{}); err == nil {
			t.Fatal("NewTestRun() error = nil, want definition validation error")
		}
	})
}

func TestTestRunDefinitionSnapshotIsIsolated(t *testing.T) {
	inputHeaders := map[string]string{"X-Token": "original"}
	request, err := definition.NewRequestSpec("GET", "https://example.com/api", inputHeaders)
	if err != nil {
		t.Fatalf("NewRequestSpec() error = %v", err)
	}
	inputMaxRequests := int64(10)
	limits, err := definition.NewLoadLimits(2, 3, time.Minute, time.Second, &inputMaxRequests)
	if err != nil {
		t.Fatalf("NewLoadLimits() error = %v", err)
	}
	d, err := definition.NewTestDefinition("definition-1", "name", request, limits)
	if err != nil {
		t.Fatalf("NewTestDefinition() error = %v", err)
	}
	r, err := run.NewTestRun("run-1", d)
	if err != nil {
		t.Fatalf("NewTestRun() error = %v", err)
	}

	inputHeaders["X-Token"] = "changed through original input"
	inputMaxRequests = 99
	definitionHeaders := d.Request().Headers()
	definitionHeaders["X-Token"] = "changed through definition"
	definitionMaxRequests := d.Limits().MaxRequests()
	*definitionMaxRequests = 98

	snapshotHeaders := r.Definition().Request().Headers()
	snapshotHeaders["X-Token"] = "changed through snapshot output"
	snapshotMaxRequests := r.Definition().Limits().MaxRequests()
	*snapshotMaxRequests = 97

	freshSnapshot := r.Definition()
	if got := freshSnapshot.Request().Headers()["X-Token"]; got != "original" {
		t.Errorf("snapshot header = %q, want original", got)
	}
	if got := freshSnapshot.Limits().MaxRequests(); got == nil || *got != 10 {
		t.Errorf("snapshot maximum requests = %v, want 10", got)
	}
	if got := d.Request().Headers()["X-Token"]; got != "original" {
		t.Errorf("original definition header = %q, want original", got)
	}
	if got := d.Limits().MaxRequests(); got == nil || *got != 10 {
		t.Errorf("original definition maximum requests = %v, want 10", got)
	}
}

func newTestDefinition(t *testing.T) definition.TestDefinition {
	t.Helper()
	request, err := definition.NewRequestSpec("GET", "https://example.com/api", nil)
	if err != nil {
		t.Fatalf("NewRequestSpec() error = %v", err)
	}
	limits, err := definition.NewLoadLimits(1, 1, time.Minute, time.Second, nil)
	if err != nil {
		t.Fatalf("NewLoadLimits() error = %v", err)
	}
	d, err := definition.NewTestDefinition("definition-1", "name", request, limits)
	if err != nil {
		t.Fatalf("NewTestDefinition() error = %v", err)
	}
	return d
}
