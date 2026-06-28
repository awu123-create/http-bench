package config

import "testing"

func validTestConfig() TestConfig {
	return TestConfig{
		Request: RequestConfig{
			Method:   "GET",
			URL:      "http://example.com/api",
			BodyType: "none",
		},
		Concurrency: 1,
		QPS:         0,
		Duration:    1,
		TimeoutMillis:     1000,
	}
}

func TestValidateConfigAllowsNoBodyAndNoHeaders(t *testing.T) {
	if got := ValidateConfig(validTestConfig()); got != None {
		t.Fatalf("ValidateConfig() = %v, want None", got)
	}
}

func TestValidateConfigRejectsInvalidURL(t *testing.T) {
	cfg := validTestConfig()
	cfg.Request.URL = "example.com/api"

	if got := ValidateConfig(cfg); got != InvalidURL {
		t.Fatalf("ValidateConfig() = %v, want InvalidURL", got)
	}
}

func TestValidateConfigRejectsInvalidMethod(t *testing.T) {
	cfg := validTestConfig()
	cfg.Request.Method = "FETCH"

	if got := ValidateConfig(cfg); got != InvalidMethod {
		t.Fatalf("ValidateConfig() = %v, want InvalidMethod", got)
	}
}

func TestValidateConfigAllowsZeroQPS(t *testing.T) {
	cfg := validTestConfig()
	cfg.QPS = 0

	if got := ValidateConfig(cfg); got != None {
		t.Fatalf("ValidateConfig() = %v, want None", got)
	}
}

func TestValidateConfigRejectsNegativeQPS(t *testing.T) {
	cfg := validTestConfig()
	cfg.QPS = -1

	if got := ValidateConfig(cfg); got != InvalidQPS {
		t.Fatalf("ValidateConfig() = %v, want InvalidQPS", got)
	}
}

func TestValidateConfigAllowsZeroMaxConnsPerHost(t *testing.T) {
	cfg := validTestConfig()
	cfg.MaxConnsPerHost = 0

	if got := ValidateConfig(cfg); got != None {
		t.Fatalf("ValidateConfig() = %v, want None (zero means fallback to concurrency)", got)
	}
}

func TestValidateConfigRejectsNegativeMaxConnsPerHost(t *testing.T) {
	cfg := validTestConfig()
	cfg.MaxConnsPerHost = -1

	if got := ValidateConfig(cfg); got != InvalidMaxConnsPerHost {
		t.Fatalf("ValidateConfig() = %v, want InvalidMaxConnsPerHost", got)
	}
}

func TestValidateConfigRejectsTooLargeMaxConnsPerHost(t *testing.T) {
	cfg := validTestConfig()
	cfg.MaxConnsPerHost = MaxMaxConnsPerHost + 1

	if got := ValidateConfig(cfg); got != InvalidMaxConnsPerHost {
		t.Fatalf("ValidateConfig() = %v, want InvalidMaxConnsPerHost", got)
	}
}

func TestValidateConfigAllowsMaxConnsPerHostUpperBound(t *testing.T) {
	cfg := validTestConfig()
	cfg.MaxConnsPerHost = MaxMaxConnsPerHost

	if got := ValidateConfig(cfg); got != None {
		t.Fatalf("ValidateConfig() = %v, want None (upper bound is allowed)", got)
	}
}
