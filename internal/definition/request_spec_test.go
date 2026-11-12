package definition_test

import (
	"reflect"
	"testing"

	"github.com/awu123-create/http-bench/internal/definition"
)

func TestNewRequestSpecAcceptsValidConfigurations(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		rawURL  string
		headers map[string]string
	}{
		{
			name:    "HTTPS request",
			method:  "GET",
			rawURL:  "https://example.com/api?q=1",
			headers: map[string]string{"X-Token": "abc"},
		},
		{
			name:   "HTTP request without headers",
			method: "POST",
			rawURL: "http://127.0.0.1:8080/api",
		},
		{
			name:    "RFC token method and header name",
			method:  "CUSTOM!#$%&'*+-.^_`|~",
			rawURL:  "HTTP://example.com",
			headers: map[string]string{"X_Custom~": "value\té"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := definition.NewRequestSpec(tt.method, tt.rawURL, tt.headers)
			if err != nil {
				t.Fatalf("NewRequestSpec() error = %v", err)
			}
			if spec.Method() != tt.method {
				t.Errorf("Method() = %q, want %q", spec.Method(), tt.method)
			}
			if spec.URL() != tt.rawURL {
				t.Errorf("URL() = %q, want %q", spec.URL(), tt.rawURL)
			}
			if !reflect.DeepEqual(spec.Headers(), tt.headers) {
				t.Errorf("Headers() = %#v, want %#v", spec.Headers(), tt.headers)
			}
		})
	}
}

func TestNewRequestSpecRejectsInvalidConfigurations(t *testing.T) {
	const validURL = "https://example.com/api"
	tests := []struct {
		name    string
		method  string
		rawURL  string
		headers map[string]string
	}{
		{name: "empty method", method: "", rawURL: validURL},
		{name: "method with space", method: "GET POST", rawURL: validURL},
		{name: "method with separator", method: "GET@", rawURL: validURL},
		{name: "method with newline", method: "GET\n", rawURL: validURL},
		{name: "method with non-ASCII character", method: "GÉT", rawURL: validURL},
		{name: "empty URL", method: "GET", rawURL: ""},
		{name: "relative URL", method: "GET", rawURL: "/api"},
		{name: "URL without scheme or host", method: "GET", rawURL: "example.com/api"},
		{name: "non-HTTP scheme", method: "GET", rawURL: "ftp://example.com/api"},
		{name: "URL without host", method: "GET", rawURL: "https:///api"},
		{name: "URL with port but without hostname", method: "GET", rawURL: "http://:8080/api"},
		{name: "malformed URL", method: "GET", rawURL: "http://[::1"},
		{name: "empty header name", method: "GET", rawURL: validURL, headers: map[string]string{"": "value"}},
		{name: "header name with space", method: "GET", rawURL: validURL, headers: map[string]string{"Bad Header": "value"}},
		{name: "header name with separator", method: "GET", rawURL: validURL, headers: map[string]string{"Bad:Header": "value"}},
		{name: "header value with NUL", method: "GET", rawURL: validURL, headers: map[string]string{"X-Test": "bad\x00value"}},
		{name: "header value with C0 control", method: "GET", rawURL: validURL, headers: map[string]string{"X-Test": "bad\x01value"}},
		{name: "header value with vertical tab", method: "GET", rawURL: validURL, headers: map[string]string{"X-Test": "bad\vvalue"}},
		{name: "header value with DEL", method: "GET", rawURL: validURL, headers: map[string]string{"X-Test": "bad\x7fvalue"}},
		{name: "header value with CR", method: "GET", rawURL: validURL, headers: map[string]string{"X-Test": "bad\rvalue"}},
		{name: "header value with LF", method: "GET", rawURL: validURL, headers: map[string]string{"X-Test": "bad\nvalue"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := definition.NewRequestSpec(tt.method, tt.rawURL, tt.headers); err == nil {
				t.Fatal("NewRequestSpec() error = nil, want validation error")
			}
		})
	}
}

func TestRequestSpecCopiesHeadersAtInputAndOutputBoundaries(t *testing.T) {
	input := map[string]string{"X-Token": "original"}
	spec, err := definition.NewRequestSpec("GET", "https://example.com", input)
	if err != nil {
		t.Fatalf("NewRequestSpec() error = %v", err)
	}

	input["X-Token"] = "changed through input"
	input["X-Input-Only"] = "new"
	if got := spec.Headers(); !reflect.DeepEqual(got, map[string]string{"X-Token": "original"}) {
		t.Fatalf("Headers() after input mutation = %#v", got)
	}

	output := spec.Headers()
	output["X-Token"] = "changed through output"
	output["X-Output-Only"] = "new"
	if got := spec.Headers(); !reflect.DeepEqual(got, map[string]string{"X-Token": "original"}) {
		t.Errorf("Headers() after output mutation = %#v", got)
	}
}
