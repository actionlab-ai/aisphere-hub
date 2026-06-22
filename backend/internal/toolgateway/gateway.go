package toolgateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
)

type HTTPGateway struct {
	client *http.Client
}

func NewHTTPGateway(client *http.Client) *HTTPGateway {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &HTTPGateway{client: client}
}

func (g *HTTPGateway) ListTools(ctx context.Context, endpoint string) (*model.SandboxToolListResponse, error) {
	endpoint, err := normalizeEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"/v1/tools", nil)
	if err != nil {
		return nil, err
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("sandbox tools list failed: status=%d body=%s", resp.StatusCode, string(body))
	}
	var out model.SandboxToolListResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	out.Endpoint = endpoint
	return &out, nil
}

func (g *HTTPGateway) Call(ctx context.Context, endpoint string, in model.SandboxToolCallRequest) (*model.SandboxToolCallResult, error) {
	endpoint, err := normalizeEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Tool) == "" {
		return nil, errors.New("tool is required")
	}
	payload := map[string]interface{}{
		"tool":  in.Tool,
		"input": in.Input,
	}
	if in.TraceID != "" {
		payload["traceId"] = in.TraceID
	}
	if in.RunID != "" {
		payload["runId"] = in.RunID
	}
	if in.Attempt > 0 {
		payload["attempt"] = in.Attempt
	}
	if in.Metadata != nil {
		payload["metadata"] = in.Metadata
	}
	b, _ := json.Marshal(payload)
	timeout := in.TimeoutMillis
	if timeout <= 0 {
		timeout = 60_000
	}
	callCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, endpoint+"/v1/tools/call", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("sandbox tool response decode failed: status=%d err=%w body=%s", resp.StatusCode, err, string(body))
	}
	out := decodeCallResult(raw)
	out.Tool = firstNonEmpty(out.Tool, in.Tool)
	out.TraceID = in.TraceID
	out.Raw = raw
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if out.Error == nil {
			out.Error = &model.SandboxToolError{Code: fmt.Sprintf("HTTP_%d", resp.StatusCode), Message: string(body)}
		}
		out.OK = false
		return out, nil
	}
	return out, nil
}

func decodeCallResult(raw map[string]interface{}) *model.SandboxToolCallResult {
	out := &model.SandboxToolCallResult{OK: boolValue(raw["ok"]), Tool: stringValue(raw["tool"]), Trace: stringValue(raw["trace"])}
	if v, ok := raw["durationMillis"]; ok {
		out.DurationMillis = int64Value(v)
	}
	if result, ok := raw["result"].(map[string]interface{}); ok {
		out.Result = result
	}
	if errObj, ok := raw["error"].(map[string]interface{}); ok {
		out.Error = &model.SandboxToolError{Code: stringValue(errObj["code"]), Message: stringValue(errObj["message"])}
	} else if s := stringValue(raw["error"]); s != "" {
		out.Error = &model.SandboxToolError{Message: s}
	}
	return out
}

func normalizeEndpoint(endpoint string) (string, error) {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		return "", errors.New("sandbox tools endpoint is empty")
	}
	if !(strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://")) {
		return "", fmt.Errorf("sandbox tools endpoint must be http(s): %s", endpoint)
	}
	return endpoint, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func boolValue(v interface{}) bool {
	b, _ := v.(bool)
	return b
}
func stringValue(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
func int64Value(v interface{}) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	case int:
		return int64(x)
	default:
		return 0
	}
}
