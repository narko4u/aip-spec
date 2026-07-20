// Package action defines Action Schemas — typed input/output contracts
// that agents use to invoke capabilities discovered through ACI manifests.
package action

import (
	"encoding/json"
	"fmt"
)

// Schema defines a single AIP Action Schema: the contract for invoking
// a specific capability on an agent or service.
type Schema struct {
	// AIPVersion identifies the AIP protocol version.
	AIPVersion string `json:"aip_version" yaml:"aip_version"`

	// ActionID uniquely identifies this action schema (e.g. "policy.evaluate.v1").
	ActionID string `json:"action_id" yaml:"action_id"`

	// Version is the schema version for this action.
	Version string `json:"version" yaml:"version"`

	// DisplayName is a human-readable name.
	DisplayName string `json:"display_name,omitempty" yaml:"display_name,omitempty"`

	// Description explains what this action does.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// InputSchema is a JSON Schema fragment describing valid inputs.
	InputSchema json.RawMessage `json:"input_schema" yaml:"input_schema"`

	// OutputSchema is a JSON Schema fragment describing valid outputs.
	OutputSchema json.RawMessage `json:"output_schema" yaml:"output_schema"`

	// Transport specifies which protocol to use for this action.
	Transport Transport `json:"transport" yaml:"transport"`

	// Timeout is the maximum duration (seconds) for this action.
	Timeout int `json:"timeout_seconds,omitempty" yaml:"timeout_seconds,omitempty"`

	// Idempotent indicates this action can be safely retried.
	Idempotent bool `json:"idempotent,omitempty" yaml:"idempotent,omitempty"`

	// Cost specifies the per-invocation cost, if any.
	Cost *Cost `json:"cost,omitempty" yaml:"cost,omitempty"`

	// RequiredEvidence lists the evidence types this action produces.
	RequiredEvidence []string `json:"required_evidence,omitempty" yaml:"required_evidence,omitempty"`
}

// Transport describes how to reach the action endpoint.
type Transport struct {
	Protocol string `json:"protocol" yaml:"protocol"`
	Endpoint string `json:"endpoint" yaml:"endpoint"`
	Method   string `json:"method,omitempty" yaml:"method,omitempty"`
	Headers  map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// Cost represents the price to invoke this action once.
type Cost struct {
	Amount   int64  `json:"amount" yaml:"amount"`
	Currency string `json:"currency" yaml:"currency"`
	// PerInvocation is true if the cost is per-call; false if flat.
	PerInvocation bool `json:"per_invocation,omitempty" yaml:"per_invocation,omitempty"`
}

// Request is an action invocation request.
type Request struct {
	ActionID  string          `json:"action_id" yaml:"action_id"`
	SchemaRef string          `json:"schema_ref,omitempty" yaml:"schema_ref,omitempty"`
	Input     json.RawMessage `json:"input" yaml:"input"`
	Nonce     string          `json:"nonce,omitempty" yaml:"nonce,omitempty"`
	Timestamp string          `json:"timestamp" yaml:"timestamp"`
}

// Response is an action invocation response.
type Response struct {
	ActionID  string          `json:"action_id" yaml:"action_id"`
	Output    json.RawMessage `json:"output,omitempty" yaml:"output,omitempty"`
	Error     *ActionError    `json:"error,omitempty" yaml:"error,omitempty"`
	ReceiptID string          `json:"receipt_id,omitempty" yaml:"receipt_id,omitempty"`
	Timestamp string          `json:"timestamp" yaml:"timestamp"`
}

// ActionError describes a failed action invocation.
type ActionError struct {
	Code    string `json:"code" yaml:"code"`
	Message string `json:"message" yaml:"message"`
	Retry   bool   `json:"retry,omitempty" yaml:"retry,omitempty"`
}

// ParseSchema parses a JSON-encoded Action Schema.
func ParseSchema(data []byte) (*Schema, error) {
	var s Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("aip/action: parse schema: %w", err)
	}
	if s.ActionID == "" {
		return nil, fmt.Errorf("aip/action: schema missing action_id")
	}
	if s.InputSchema == nil {
		return nil, fmt.Errorf("aip/action: schema missing input_schema")
	}
	if s.OutputSchema == nil {
		return nil, fmt.Errorf("aip/action: schema missing output_schema")
	}
	return &s, nil
}

// MustParseSchema parses a schema and panics on error.
func MustParseSchema(data []byte) *Schema {
	s, err := ParseSchema(data)
	if err != nil {
		panic(err)
	}
	return s
}
