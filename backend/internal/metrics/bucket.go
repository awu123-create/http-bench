package metrics

type bucket struct {
	totalRequests           int64
	success                 int64
	failed                  int64
	latencySum              int64
	minLatency              int64
	maxLatency              int64
	requestBuildErrors      int64
	timeoutErrors           int64
	canceledErrors          int64
	networkErrors           int64
	dnsErrors               int64
	connectionRefusedErrors int64
	connectionResetErrors   int64
	tooManyOpenFilesErrors  int64
	readBodyErrors          int64
	http4xxErrors           int64
	http5xxErrors           int64
	httpStatusErrors        int64
	businessErrors          int64
	clientOverloadErrors    int64
	statusCodes             map[int]int64
}

func newBucket() *bucket {
	return &bucket{
		statusCodes: make(map[int]int64),
	}
}

func (b *bucket) add(result RequestResult) {
	b.totalRequests++

	errorType := normalizeErrorType(result)
	if errorType != ErrorNone {
		b.failed++
		b.addError(errorType)
	} else {
		b.success++
	}

	b.latencySum += result.Latency
	if b.minLatency == 0 || result.Latency < b.minLatency {
		b.minLatency = result.Latency
	}
	if result.Latency > b.maxLatency {
		b.maxLatency = result.Latency
	}
	if result.StatusCode > 0 {
		b.statusCodes[result.StatusCode]++
	}
}

func normalizeErrorType(result RequestResult) ErrorType {
	if result.ErrorType != ErrorNone {
		return result.ErrorType
	}
	if result.Err != nil {
		return ErrorNetwork
	}
	if result.StatusCode >= 400 && result.StatusCode < 500 {
		return ErrorHTTP4xx
	}
	if result.StatusCode >= 500 && result.StatusCode < 600 {
		return ErrorHTTP5xx
	}
	if result.StatusCode < 200 || result.StatusCode >= 600 {
		return ErrorHTTPStatus
	}
	return ErrorNone
}

func (b *bucket) addError(errorType ErrorType) {
	switch errorType {
	case ErrorRequestBuild:
		b.requestBuildErrors++
	case ErrorTimeout:
		b.timeoutErrors++
	case ErrorCanceled:
		b.canceledErrors++
	case ErrorNetwork:
		b.networkErrors++
	case ErrorDNS:
		b.dnsErrors++
	case ErrorConnectionRefused:
		b.connectionRefusedErrors++
	case ErrorConnectionReset:
		b.connectionResetErrors++
	case ErrorTooManyOpenFiles:
		b.tooManyOpenFilesErrors++
	case ErrorReadBody:
		b.readBodyErrors++
	case ErrorHTTP4xx:
		b.http4xxErrors++
	case ErrorHTTP5xx:
		b.http5xxErrors++
	case ErrorHTTPStatus:
		b.httpStatusErrors++
	case ErrorBusiness:
		b.businessErrors++
	case ErrorClientOverload:
		b.clientOverloadErrors++
	}
}

func deepCopy(statusCodes map[int]int64) map[int]int64 {
	copied := make(map[int]int64)
	for k, v := range statusCodes {
		copied[k] = v
	}
	return copied
}

func (b *bucket) snapshot() Snapshot {
	if b.totalRequests == 0 {
		return Snapshot{}
	}

	return Snapshot{
		TotalRequests:           b.totalRequests,
		Success:                 b.success,
		Failed:                  b.failed,
		AvgLatency:              float64(b.latencySum) / float64(b.totalRequests),
		MinLatency:              b.minLatency,
		MaxLatency:              b.maxLatency,
		RequestBuildErrors:      b.requestBuildErrors,
		TimeoutErrors:           b.timeoutErrors,
		CanceledErrors:          b.canceledErrors,
		NetworkErrors:           b.networkErrors,
		DNSErrors:               b.dnsErrors,
		ConnectionRefusedErrors: b.connectionRefusedErrors,
		ConnectionResetErrors:   b.connectionResetErrors,
		TooManyOpenFilesErrors:  b.tooManyOpenFilesErrors,
		ReadBodyErrors:          b.readBodyErrors,
		HTTP4xxErrors:           b.http4xxErrors,
		HTTP5xxErrors:           b.http5xxErrors,
		HTTPStatusErrors:        b.httpStatusErrors,
		BusinessErrors:          b.businessErrors,
		ClientOverloadErrors:    b.clientOverloadErrors,
		StatusCodes:             deepCopy(b.statusCodes),
	}
}

func (b *bucket) reset() {
	*b = bucket{
		statusCodes: make(map[int]int64),
	}
}
