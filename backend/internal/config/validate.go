package config

import (
	"net/http"
	"net/url"
	"strings"
)

type ValidateError int

const (
	None ValidateError = iota
	InvalidMethod
	InvalidURL
	InvalidBodyType
	InvalidConcurrency
	InvalidDuration
	InvalidQPS
	InvalidTimeout
	InvalidMaxConnsPerHost
)

const (
	MaxMaxConnsPerHost = 10000
	MaxTimeoutMillis   = 60000
	MaxConcurrency     = 10000
	MaxQPS             = 100000
	MaxDuration        = 3600
)

func ValidateConfig(cfg TestConfig) ValidateError {
	method := strings.ToUpper(strings.TrimSpace(cfg.Request.Method))
	if !validMethod(method) {
		return InvalidMethod
	}
	if !validURL(cfg.Request.URL) {
		return InvalidURL
	}
	if !validBodyType(cfg.Request.BodyType) {
		return InvalidBodyType
	}
	if cfg.Concurrency <= 0 || cfg.Concurrency > MaxConcurrency {
		return InvalidConcurrency
	}
	if cfg.Duration <= 0 || cfg.Duration > MaxDuration {
		return InvalidDuration
	}
	if cfg.QPS < 0 || cfg.QPS > MaxQPS {
		return InvalidQPS
	}
	if cfg.TimeoutMillis <= 0 || cfg.TimeoutMillis > MaxTimeoutMillis {
		return InvalidTimeout
	}
	if cfg.MaxConnsPerHost < 0 || cfg.MaxConnsPerHost > MaxMaxConnsPerHost {
		return InvalidMaxConnsPerHost
	}
	return None
}

func validMethod(method string) bool {
	switch method {
	case http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions:
		return true
	default:
		return false
	}
}

func validURL(rawURL string) bool {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func validBodyType(bodyType string) bool {
	switch strings.TrimSpace(bodyType) {
	case "", "none", "raw", "x-www-form-urlencoded", "form-data":
		return true
	default:
		return false
	}
}
