package definition_test

import (
	"testing"
	"time"

	"github.com/awu123-create/http-bench/internal/definition"
)

func TestNewTestDefinition(t *testing.T) {
	request := mustRequestSpec(t)
	maxRequests := int64(10)
	limits, err := definition.NewLoadLimits(2, 3, time.Minute, time.Second, &maxRequests)
	if err != nil {
		t.Fatalf("NewLoadLimits() error = %v", err)
	}

	d, err := definition.NewTestDefinition("definition-1", "", request, limits)
	if err != nil {
		t.Fatalf("NewTestDefinition() error = %v", err)
	}
	if d.ID() != definition.DefinitionID("definition-1") {
		t.Errorf("ID() = %q, want definition-1", d.ID())
	}
	if d.Name() != "" {
		t.Errorf("Name() = %q, want empty name to remain valid", d.Name())
	}
	if d.Request().Method() != request.Method() || d.Request().URL() != request.URL() {
		t.Errorf("Request() did not preserve request configuration")
	}
	if d.Limits().Concurrency() != limits.Concurrency() || *d.Limits().MaxRequests() != maxRequests {
		t.Errorf("Limits() did not preserve load limits")
	}
}

func TestNewTestDefinitionRejectsInvalidInputs(t *testing.T) {
	validRequest := mustRequestSpec(t)
	validLimits := mustLoadLimits(t)

	t.Run("blank definition IDs", func(t *testing.T) {
		for _, id := range []definition.DefinitionID{"", " ", "\t\n"} {
			if _, err := definition.NewTestDefinition(id, "name", validRequest, validLimits); err == nil {
				t.Errorf("NewTestDefinition(%q) error = nil, want validation error", id)
			}
		}
	})

	t.Run("zero request", func(t *testing.T) {
		if _, err := definition.NewTestDefinition("definition-1", "name", definition.RequestSpec{}, validLimits); err == nil {
			t.Fatal("NewTestDefinition() error = nil, want request validation error")
		}
	})

	t.Run("zero load limits", func(t *testing.T) {
		if _, err := definition.NewTestDefinition("definition-1", "name", validRequest, definition.LoadLimits{}); err == nil {
			t.Fatal("NewTestDefinition() error = nil, want load limit validation error")
		}
	})
}

func TestTestDefinitionAccessorsReturnIndependentCopies(t *testing.T) {
	d, err := definition.NewTestDefinition("definition-1", "name", mustRequestSpec(t), mustLoadLimits(t))
	if err != nil {
		t.Fatalf("NewTestDefinition() error = %v", err)
	}

	headers := d.Request().Headers()
	headers["X-Token"] = "changed"
	maxRequests := d.Limits().MaxRequests()
	*maxRequests = 999

	if got := d.Request().Headers()["X-Token"]; got != "original" {
		t.Errorf("definition header after returned map mutation = %q, want original", got)
	}
	if got := d.Limits().MaxRequests(); got == nil || *got != 10 {
		t.Errorf("definition maximum requests after returned pointer mutation = %v, want 10", got)
	}
}

func mustRequestSpec(t *testing.T) definition.RequestSpec {
	t.Helper()
	request, err := definition.NewRequestSpec(
		"GET",
		"https://example.com/api",
		map[string]string{"X-Token": "original"},
	)
	if err != nil {
		t.Fatalf("NewRequestSpec() error = %v", err)
	}
	return request
}

func mustLoadLimits(t *testing.T) definition.LoadLimits {
	t.Helper()
	maxRequests := int64(10)
	limits, err := definition.NewLoadLimits(2, 3, time.Minute, time.Second, &maxRequests)
	if err != nil {
		t.Fatalf("NewLoadLimits() error = %v", err)
	}
	return limits
}
