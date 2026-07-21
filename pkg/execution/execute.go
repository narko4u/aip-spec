// Package execution defines the action invocation lifecycle:
// validate input → invoke transport → validate output → record evidence.
package execution

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/narko4u/aip-spec/pkg/action"
	"github.com/narko4u/aip-spec/pkg/contract"
)

// Result captures the outcome of an action execution.
type Result struct {
	Success   bool            `json:"success"`
	ActionID  string          `json:"action_id"`
	Output    json.RawMessage `json:"output,omitempty"`
	Error     *action.ActionError `json:"error,omitempty"`
	ReceiptID string          `json:"receipt_id,omitempty"`
	DurationMs int64          `json:"duration_ms"`
	Timestamp string          `json:"timestamp"`
}

// Engine executes action invocations against their declared transport.
type Engine struct {
	client *http.Client
}

// NewEngine creates an execution engine.
func NewEngine(timeout time.Duration) *Engine {
	return &Engine{
		client: &http.Client{Timeout: timeout},
	}
}

// Execute invokes an action against a validated schema, using the binding's
// transport and auth context.
func (e *Engine) Execute(schema *action.Schema, req *action.Request, binding *contract.Binding) (*Result, error) {
	start := time.Now()

	// 1. Validate input against schema (basic structural check for v0.1)
	if err := validateInput(schema, req.Input); err != nil {
		return nil, fmt.Errorf("aip/execution: input validation failed: %w", err)
	}

	// 2. Dispatch via transport
	var output json.RawMessage
	switch schema.Transport.Protocol {
	case "https", "http":
		resp, err := e.httpInvoke(schema, req)
		if err != nil {
			return e.errorResult(req.ActionID, "transport_error", err.Error(), start), nil
		}
		output = resp
	case "mcp":
		// MCP transport — placeholder for future implementation
		return nil, fmt.Errorf("aip/execution: MCP transport not yet implemented")
	case "grpc":
		return nil, fmt.Errorf("aip/execution: gRPC transport not yet implemented")
	default:
		return nil, fmt.Errorf("aip/execution: unsupported transport: %s", schema.Transport.Protocol)
	}

	// 3. Validate output against schema
	if err := validateOutput(schema, output); err != nil {
		return nil, fmt.Errorf("aip/execution: output validation failed: %w", err)
	}

	duration := time.Since(start)

	return &Result{
		Success:    true,
		ActionID:   req.ActionID,
		Output:     output,
		ReceiptID:  fmt.Sprintf("rec_%s_%d", req.ActionID, start.UnixNano()),
		DurationMs: duration.Milliseconds(),
		Timestamp:  start.UTC().Format(time.RFC3339),
	}, nil
}

func (e *Engine) httpInvoke(schema *action.Schema, req *action.Request) (json.RawMessage, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(schema.Transport.Method, schema.Transport.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range schema.Transport.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := e.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func validateInput(schema *action.Schema, input json.RawMessage) error {
	if input == nil {
		return fmt.Errorf("input is required")
	}
	// v0.1: basic JSON validity check.
	// v0.2+: full JSON Schema validation against schema.InputSchema.
	var v interface{}
	if err := json.Unmarshal(input, &v); err != nil {
		return fmt.Errorf("invalid JSON input: %w", err)
	}
	return nil
}

func validateOutput(schema *action.Schema, output json.RawMessage) error {
	if output == nil {
		return fmt.Errorf("output is required")
	}
	var v interface{}
	if err := json.Unmarshal(output, &v); err != nil {
		return fmt.Errorf("invalid JSON output: %w", err)
	}
	return nil
}

func (e *Engine) errorResult(actionID, code, message string, start time.Time) *Result {
	return &Result{
		Success:    false,
		ActionID:   actionID,
		Error:      &action.ActionError{Code: code, Message: message, Retry: true},
		ReceiptID:  fmt.Sprintf("rec_%s_%d", actionID, start.UnixNano()),
		DurationMs: time.Since(start).Milliseconds(),
		Timestamp:  start.UTC().Format(time.RFC3339),
	}
}
