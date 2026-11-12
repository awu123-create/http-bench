package definition

import (
	"fmt"
	"maps"
	"strings"
)

type DefinitionID string

type TestDefinition struct {
	id      DefinitionID
	name    string
	request RequestSpec
	limits  LoadLimits
}

// NewTestDefinition validates and copies a complete test definition.
func NewTestDefinition(
	id DefinitionID,
	name string,
	request RequestSpec,
	limits LoadLimits,
) (TestDefinition, error) {
	if strings.TrimSpace(string(id)) == "" {
		return TestDefinition{}, fmt.Errorf("definition ID must not be blank")
	}

	requestCopy, err := NewRequestSpec(request.method, request.rawURL, request.headers)
	if err != nil {
		return TestDefinition{}, fmt.Errorf("validate request: %w", err)
	}
	limitsCopy, err := NewLoadLimits(
		limits.concurrency,
		limits.qps,
		limits.maxDuration,
		limits.requestTimeout,
		limits.maxRequests,
	)
	if err != nil {
		return TestDefinition{}, fmt.Errorf("validate load limits: %w", err)
	}

	return TestDefinition{
		id:      id,
		name:    name,
		request: requestCopy,
		limits:  limitsCopy,
	}, nil
}

func (d TestDefinition) ID() DefinitionID {
	return d.id
}

// Name returns the optional display name.
func (d TestDefinition) Name() string {
	return d.name
}

// Request returns an independent copy of the request configuration.
func (d TestDefinition) Request() RequestSpec {
	return RequestSpec{
		method:  d.request.method,
		rawURL:  d.request.rawURL,
		headers: maps.Clone(d.request.headers),
	}
}

// Limits returns an independent copy of the load limits.
func (d TestDefinition) Limits() LoadLimits {
	limits := d.limits
	if d.limits.maxRequests != nil {
		value := *d.limits.maxRequests
		limits.maxRequests = &value
	}
	return limits
}
