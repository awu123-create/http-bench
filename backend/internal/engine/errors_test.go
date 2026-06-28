package engine

import (
	"context"
	"errors"
	"net"
	"syscall"
	"testing"

	"github.com/awu123-create/http-bench/internal/metrics"
)

func TestClassifyDoError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want metrics.ErrorType
	}{
		{name: "nil", err: nil, want: metrics.ErrorNone},
		{name: "canceled", err: context.Canceled, want: metrics.ErrorCanceled},
		{name: "deadline", err: context.DeadlineExceeded, want: metrics.ErrorTimeout},
		{name: "dns", err: &net.DNSError{Err: "no such host", Name: "example.invalid"}, want: metrics.ErrorDNS},
		{name: "connection_refused", err: &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}, want: metrics.ErrorConnectionRefused},
		{name: "connection_reset", err: &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET}, want: metrics.ErrorConnectionReset},
		{name: "too_many_open_files", err: &net.OpError{Op: "dial", Net: "tcp", Err: syscall.EMFILE}, want: metrics.ErrorTooManyOpenFiles},
		{name: "network", err: errors.New("connection reset by peer"), want: metrics.ErrorNetwork},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyDoError(tt.err); got != tt.want {
				t.Fatalf("classifyDoError() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassifyStatusCode(t *testing.T) {
	tests := []struct {
		statusCode int
		want       metrics.ErrorType
	}{
		{statusCode: 200, want: metrics.ErrorNone},
		{statusCode: 302, want: metrics.ErrorNone},
		{statusCode: 404, want: metrics.ErrorHTTP4xx},
		{statusCode: 500, want: metrics.ErrorHTTP5xx},
		{statusCode: 101, want: metrics.ErrorHTTPStatus},
	}

	for _, tt := range tests {
		if got := classifyStatusCode(tt.statusCode); got != tt.want {
			t.Fatalf("classifyStatusCode(%d) = %q, want %q", tt.statusCode, got, tt.want)
		}
	}
}
