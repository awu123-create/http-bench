package engine

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"errors"

	"github.com/awu123-create/http-bench/internal/config"
)

func buildRequest(ctx context.Context, cfg config.TestConfig) (*http.Request, error) {
	reqCfg := cfg.Request
	method := strings.ToUpper(strings.TrimSpace(reqCfg.Method))
	if method == ""{
		return nil, errors.New("HTTP method is required")
	}

	var body io.Reader
	var contentType string
	var forceContentType bool

	switch reqCfg.BodyType {
	case "none", "":
	case "raw":
		body = bytes.NewReader([]byte(reqCfg.BodyRaw))
		contentType = "application/json"
	case "x-www-form-urlencoded":
		body = strings.NewReader(reqCfg.BodyRaw)
		contentType = "application/x-www-form-urlencoded"
	case "form-data":
		buf, ct, err := buildMultipartBody(reqCfg.BodyRaw)
		if err != nil {
			return nil, fmt.Errorf("build multipart body: %w", err)
		}
		body = bytes.NewReader(buf)
		contentType = ct
		forceContentType = true
	default:
		return nil, fmt.Errorf("unsupported body type: %s", reqCfg.BodyType)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqCfg.URL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	for k, v := range reqCfg.Headers {
		req.Header.Set(k, v)
	}
	if contentType != "" {
		if forceContentType || req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", contentType)
		}
	}
	return req, nil
}

type formDataSpec struct {
	Fields map[string]string `json:"fields"`
	Files  []formFileSpec    `json:"files"`
}

type formFileSpec struct {
	Name          string `json:"name"`
	Filename      string `json:"filename"`
	Content       string `json:"content"`
	ContentBase64 string `json:"content_base64"`
}

func buildMultipartBody(bodyRaw string) ([]byte, string, error) {
	if strings.TrimSpace(bodyRaw) == "" {
		return nil, "", errors.New("multipart body is required")
	}

	var spec formDataSpec
	if err := json.Unmarshal([]byte(bodyRaw), &spec); err != nil {
		return nil, "", err
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// 写普通字段
	for k, v := range spec.Fields {
		if err := w.WriteField(k, v); err != nil {
			return nil, "", fmt.Errorf("write field %s: %w", k, err)
		}
	}

	// 写文件 part
	for _, fileSpec := range spec.Files {
		if fileSpec.Content == "" && fileSpec.ContentBase64 == "" {
			return nil, "", fmt.Errorf("file %s has no content", fileSpec.Name)
		}

		fw, err := w.CreateFormFile(fileSpec.Name, fileSpec.Filename)
		if err != nil {
			return nil, "", fmt.Errorf("create form file %s: %w", fileSpec.Name, err)
		}

		// 决定写入的内容，优先使用 base64 解码的内容
		var content []byte
		if fileSpec.ContentBase64 != "" {
			content, err = base64.StdEncoding.DecodeString(fileSpec.ContentBase64)
			if err != nil {
				return nil, "", fmt.Errorf("decode file content %s: %w", fileSpec.Name, err)
			}
		} else {
			content = []byte(fileSpec.Content)
		}

		if _, err := fw.Write(content); err != nil {
			return nil, "", fmt.Errorf("write file content %s: %w", fileSpec.Name, err)
		}
	}

	// 写结束边界
	if err := w.Close(); err != nil {
		return nil, "", fmt.Errorf("close multipart writer: %w", err)
	}
	// 返回 body 字节和带 boundary 的 Content-Type
	return buf.Bytes(), w.FormDataContentType(), nil
}
