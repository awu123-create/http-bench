package run

import (
	"fmt"
	"strings"

	"github.com/awu123-create/http-bench/internal/definition"
)

type RunID string

// TestRun associates a run identity with its immutable definition snapshot.
type TestRun struct {
	id         RunID
	definition definition.TestDefinition
}

// NewTestRun validates the run identity and saves a deep copy of the definition.
func NewTestRun(id RunID, d definition.TestDefinition) (TestRun, error) {
	if strings.TrimSpace(string(id)) == "" {
		return TestRun{}, fmt.Errorf("run ID must not be blank")
	}

	snapshot, err := definition.NewTestDefinition(d.ID(), d.Name(), d.Request(), d.Limits())
	if err != nil {
		return TestRun{}, fmt.Errorf("copy definition snapshot: %w", err)
	}

	return TestRun{id: id, definition: snapshot}, nil
}

func (r TestRun) ID() RunID {
	return r.id
}

// Definition returns the run's definition snapshot.
func (r TestRun) Definition() definition.TestDefinition {
	return r.definition
}
