package metrics

import (
	"time"
)

type ErrorType string

const (
	ErrorNone              ErrorType = ""
	ErrorRequestBuild      ErrorType = "request_build"       // 请求构建错误
	ErrorTimeout           ErrorType = "timeout"             // 超时错误
	ErrorCanceled          ErrorType = "canceled"            // 请求被取消
	ErrorNetwork           ErrorType = "network"             // 网络错误
	ErrorDNS               ErrorType = "dns"                 // DNS错误
	ErrorConnectionRefused ErrorType = "connection_refused"  // 连接被拒绝
	ErrorConnectionReset   ErrorType = "connection_reset"    // 连接重置
	ErrorTooManyOpenFiles  ErrorType = "too_many_open_files" // 打开的文件过多
	ErrorReadBody          ErrorType = "read_body"           // 读取响应体错误
	ErrorHTTP4xx           ErrorType = "http_4xx"            // 客户端错误
	ErrorHTTP5xx           ErrorType = "http_5xx"            // 服务器错误
	ErrorHTTPStatus        ErrorType = "http_status"         // HTTP状态码错误
	ErrorBusiness          ErrorType = "business"            // 业务错误
	ErrorClientOverload    ErrorType = "client_overload"     // 客户端过载
)

// 压测中一次请求的结果
type RequestResult struct {
	StatusCode int
	Latency    int64
	Err        error
	ErrorType  ErrorType
	ErrorMsg   string
	FinishedAt time.Time
}

// Snapshot 是按秒聚合的压测结果快照
type Snapshot struct {
	TotalRequests           int64              `json:"total_requests"`
	Success                 int64              `json:"success"`
	Failed                  int64              `json:"failed"`
	AvgLatency              float64            `json:"avg_latency"`
	MinLatency              int64              `json:"min_latency"`
	MaxLatency              int64              `json:"max_latency"`
	RequestBuildErrors      int64              `json:"request_build_errors"`
	TimeoutErrors           int64              `json:"timeout_errors"`
	CanceledErrors          int64              `json:"canceled_errors"`
	NetworkErrors           int64              `json:"network_errors"`
	DNSErrors               int64              `json:"dns_errors"`
	ConnectionRefusedErrors int64              `json:"connection_refused_errors"`
	ConnectionResetErrors   int64              `json:"connection_reset_errors"`
	TooManyOpenFilesErrors  int64              `json:"too_many_open_files_errors"`
	ReadBodyErrors          int64              `json:"read_body_errors"`
	HTTP4xxErrors           int64              `json:"http_4xx_errors"`
	HTTP5xxErrors           int64              `json:"http_5xx_errors"`
	HTTPStatusErrors        int64              `json:"http_status_errors"`
	BusinessErrors          int64              `json:"business_errors"`
	ClientOverloadErrors    int64              `json:"client_overload_errors"`
	StatusCodes             map[int]int64      `json:"status_codes"`
}

// Collector 是压测结果收集器的接口
type Collector interface {
	Record(result RequestResult)
	Snapshots() <-chan Snapshot
}
