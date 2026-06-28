package engine

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"syscall"

	"github.com/awu123-create/http-bench/internal/metrics"
)

func classifyDoError(err error) metrics.ErrorType {
	if err == nil {
		return metrics.ErrorNone
	}
	if errors.Is(err, context.Canceled) {
		return metrics.ErrorCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return metrics.ErrorTimeout
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return metrics.ErrorTimeout
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return metrics.ErrorDNS
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		switch {
		case errors.Is(opErr.Err, syscall.ECONNREFUSED):
			return metrics.ErrorConnectionRefused
		case errors.Is(opErr.Err, syscall.ECONNRESET):
			return metrics.ErrorConnectionReset
		case errors.Is(opErr.Err, syscall.EMFILE):
			return metrics.ErrorTooManyOpenFiles
		}
	}
	return metrics.ErrorNetwork
}

func classifyStatusCode(statusCode int) metrics.ErrorType {
	switch {
	case statusCode >= 200 && statusCode < 400:
		return metrics.ErrorNone
	case statusCode >= 400 && statusCode < 500:
		return metrics.ErrorHTTP4xx
	case statusCode >= 500 && statusCode < 600:
		return metrics.ErrorHTTP5xx
	default:
		return metrics.ErrorHTTPStatus
	}
}

func httpStatusMessage(statusCode int) string {
	if text := http.StatusText(statusCode); text != "" {
		return text
	}
	return fmt.Sprintf("unexpected HTTP status %d", statusCode)
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
