package engine

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/awu123-create/http-bench/internal/config"
)

func TestBuildRequestRawJSON(t *testing.T) {
	req, err := buildRequest(context.Background(), config.TestConfig{
		Request: config.RequestConfig{
			Method:   "post",
			URL:      "http://example.com/api",
			Headers:  map[string]string{"X-Test": "yes"},
			BodyType: "raw",
			BodyRaw:  `{"name":"alice"}`,
		},
	})
	if err != nil {
		t.Fatalf("buildRequest returned error: %v", err)
	}

	if req.Method != http.MethodPost {
		t.Fatalf("Method = %q, want POST", req.Method)
	}
	if req.URL.String() != "http://example.com/api" {
		t.Fatalf("URL = %q, want http://example.com/api", req.URL.String())
	}
	if got := req.Header.Get("X-Test"); got != "yes" {
		t.Fatalf("X-Test header = %q, want yes", got)
	}
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if string(body) != `{"name":"alice"}` {
		t.Fatalf("body = %q, want raw JSON body", string(body))
	}
}

func TestBuildRequestPreservesExplicitContentType(t *testing.T) {
	req, err := buildRequest(context.Background(), config.TestConfig{
		Request: config.RequestConfig{
			Method:   "POST",
			URL:      "http://example.com/form",
			Headers:  map[string]string{"Content-Type": "text/plain"},
			BodyType: "x-www-form-urlencoded",
			BodyRaw:  "a=1&b=2",
		},
	})
	if err != nil {
		t.Fatalf("buildRequest returned error: %v", err)
	}

	if got := req.Header.Get("Content-Type"); got != "text/plain" {
		t.Fatalf("Content-Type = %q, want explicit text/plain", got)
	}
}

func TestBuildRequestRequiresMethod(t *testing.T) {
	_, err := buildRequest(context.Background(), config.TestConfig{
		Request: config.RequestConfig{URL: "http://example.com"},
	})
	if err == nil {
		t.Fatal("buildRequest returned nil error for missing method")
	}
}
