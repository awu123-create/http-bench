package config

// RequestConfig 是压测请求的配置结构体
type RequestConfig struct {
	Method   string            `json:"method"`    // HTTP method (GET, POST, etc.)
	URL      string            `json:"url"`       // 压测的 URL
	Headers  map[string]string `json:"headers"`   // 压测时 HTTP 请求的请求头
	BodyType string            `json:"body_type"` // 压测时 HTTP 请求的请求体类型，支持 form-data、x-www-form-urlencoded、raw
	BodyRaw  string            `json:"body_raw"`  // 压测时 HTTP 请求的请求体内容
}

// TestConfig 是压测测试的配置结构体
type TestConfig struct {
	Name            string        `json:"name"`               // 压测测试的名称
	Request         RequestConfig `json:"request"`            // 压测请求的配置
	Concurrency     int           `json:"concurrency"`        // 并发数
	QPS             int           `json:"qps"`                // 每秒请求数
	Duration        int           `json:"duration"`           // 压测持续时间，单位为秒
	TimeoutMillis   int           `json:"timeout_millis"`     // 压测请求的超时时间，单位为毫秒
	MaxConnsPerHost int           `json:"max_conns_per_host"` // 每个主机的最大连接数
}
