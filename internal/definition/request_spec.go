package definition

import (
	"fmt"
	"maps"
	"net/url"
	"strings"
)

type RequestSpec struct {
	method  string
	rawURL  string
	headers map[string]string
}

// NewRequestSpec validates and copies a request configuration.
func NewRequestSpec(method, rawURL string, headers map[string]string) (RequestSpec, error) {
	if !isToken(method) {
		return RequestSpec{}, fmt.Errorf("request method must be a non-empty RFC token")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return RequestSpec{}, fmt.Errorf("parse request URL: %w", err)
	}
	if !parsedURL.IsAbs() ||
		(!strings.EqualFold(parsedURL.Scheme, "http") && !strings.EqualFold(parsedURL.Scheme, "https")) ||
		parsedURL.Hostname() == "" {
		return RequestSpec{}, fmt.Errorf("request URL must be an absolute HTTP or HTTPS URL with a host")
	}

	for name, value := range headers {
		if !isToken(name) {
			return RequestSpec{}, fmt.Errorf("header name %q must be a non-empty RFC token", name)
		}
		for i := 0; i < len(value); i++ {
			c := value[i]
			if (c < 0x20 && c != '\t') || c == 0x7f {
				return RequestSpec{}, fmt.Errorf("header %q value contains an invalid control character", name)
			}
		}
	}

	return RequestSpec{
		method:  method,
		rawURL:  rawURL,
		headers: cloneHeaders(headers),
	}, nil
}

func (s RequestSpec) Method() string {
	return s.method
}

// URL returns the absolute request URL.
func (s RequestSpec) URL() string {
	return s.rawURL
}

// Headers returns an independent copy of the request headers.
func (s RequestSpec) Headers() map[string]string {
	return cloneHeaders(s.headers)
}

func cloneHeaders(headers map[string]string) map[string]string {
	return maps.Clone(headers)
}

func isToken(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			continue
		}
		if !strings.ContainsRune("!#$%&'*+-.^_`|~", rune(c)) {
			return false
		}
	}
	return true
}
