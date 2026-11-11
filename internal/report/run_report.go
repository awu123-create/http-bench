package report

import (
	"fmt"
	"strings"

	"github.com/awu123-create/http-bench/internal/run"
)

type RunReport struct {
	runID run.RunID
}

func NewRunReport(runID run.RunID) (RunReport, error) {
	if strings.TrimSpace(string(runID)) == "" {
		return RunReport{}, fmt.Errorf("run ID must not be blank")
	}
	return RunReport{runID: runID}, nil
}

func (r RunReport) RunID() run.RunID {
	return r.runID
}
