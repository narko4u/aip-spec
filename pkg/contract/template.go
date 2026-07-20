// Package contract defines Contract Templates and Binding Agreements
// between agents discovered through ACI and negotiated via AIP.
package contract

import "time"

// Status represents the lifecycle state of a contract.
type Status string

const (
	StatusPending    Status = "pending"
	StatusActive     Status = "active"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
	StatusDisputed   Status = "disputed"
	StatusCancelled  Status = "cancelled"
)

// Template defines re-usable terms for a class of contracts.
// Templates are published in ACI Capability Manifests and referenced
// by action_id.
type Template struct {
	// TemplateID uniquely identifies this contract template.
	TemplateID string `json:"template_id" yaml:"template_id"`

	// AIPVersion identifies the protocol version.
	AIPVersion string `json:"aip_version" yaml:"aip_version"`

	// ActionID references the Action Schema this template applies to.
	ActionID string `json:"action_id" yaml:"action_id"`

	// DisplayName is a human-readable label.
	DisplayName string `json:"display_name,omitempty" yaml:"display_name,omitempty"`

	// Pricing defines the cost structure.
	Pricing *Pricing `json:"pricing,omitempty" yaml:"pricing,omitempty"`

	// SLA defines service level commitments.
	SLA *SLA `json:"sla,omitempty" yaml:"sla,omitempty"`

	// Evidence specifies what evidence the provider will produce.
	Evidence *EvidenceSpec `json:"evidence,omitempty" yaml:"evidence,omitempty"`

	// Dispute describes how disputes are resolved.
	Dispute *DisputeSpec `json:"dispute,omitempty" yaml:"dispute,omitempty"`

	// Terms is human-readable legal or policy text.
	Terms string `json:"terms,omitempty" yaml:"terms,omitempty"`
}

// Pricing defines the cost model.
type Pricing struct {
	// Model: "flat", "per_invocation", "subscription", "revenue_share"
	Model string `json:"model" yaml:"model"`

	// Amount in smallest currency unit (cents).
	Amount int64 `json:"amount" yaml:"amount"`

	// Currency code (e.g. "USD", "AUD", "USDC").
	Currency string `json:"currency" yaml:"currency"`

	// Interval for subscription model ("monthly", "yearly", "per_action").
	Interval string `json:"interval,omitempty" yaml:"interval,omitempty"`

	// RevenueShare is a percentage (0.0–100.0) for revenue_share model.
	RevenueShare float64 `json:"revenue_share,omitempty" yaml:"revenue_share,omitempty"`
}

// SLA defines service level commitments.
type SLA struct {
	// MaxLatencyMs is the maximum acceptable response time.
	MaxLatencyMs int `json:"max_latency_ms,omitempty" yaml:"max_latency_ms,omitempty"`

	// MinUptimePercent is the minimum uptime guarantee (0.0–100.0).
	MinUptimePercent float64 `json:"min_uptime_percent,omitempty" yaml:"min_uptime_percent,omitempty"`

	// MaxRetries is the maximum number of retries allowed.
	MaxRetries int `json:"max_retries,omitempty" yaml:"max_retries,omitempty"`

	// RateLimit is the maximum requests per time window.
	RateLimit *RateLimit `json:"rate_limit,omitempty" yaml:"rate_limit,omitempty"`
}

// RateLimit defines request throttling.
type RateLimit struct {
	Requests int `json:"requests" yaml:"requests"`
	PerSeconds int `json:"per_seconds" yaml:"per_seconds"`
}

// EvidenceSpec defines what evidence the provider produces.
type EvidenceSpec struct {
	// Types lists the evidence types (e.g. "witnessos_receipt", "log_export").
	Types []string `json:"types" yaml:"types"`

	// RetentionDays is how long evidence is kept.
	RetentionDays int `json:"retention_days,omitempty" yaml:"retention_days,omitempty"`
}

// DisputeSpec defines the dispute resolution process.
type DisputeSpec struct {
	// Arbiter is the entity that resolves disputes (e.g. "empirelabs.com").
	Arbiter string `json:"arbiter,omitempty" yaml:"arbiter,omitempty"`

	// ResolutionTimeoutSeconds is the max time for resolution.
	ResolutionTimeoutSeconds int `json:"resolution_timeout_seconds,omitempty" yaml:"resolution_timeout_seconds,omitempty"`
}

// Binding is a fully negotiated contract between two parties.
type Binding struct {
	// ContractID uniquely identifies this binding.
	ContractID string `json:"contract_id" yaml:"contract_id"`

	// TemplateID is the template this was derived from.
	TemplateID string `json:"template_id" yaml:"template_id"`

	// Status is the current lifecycle status.
	Status Status `json:"status" yaml:"status"`

	// Provider is the agent providing the capability.
	Provider string `json:"provider" yaml:"provider"`

	// Consumer is the agent consuming the capability.
	Consumer string `json:"consumer" yaml:"consumer"`

	// ActionID is the action being contracted.
	ActionID string `json:"action_id" yaml:"action_id"`

	// Pricing is the agreed pricing (may differ from template).
	Pricing *Pricing `json:"pricing" yaml:"pricing"`

	// SLA is the agreed SLA (may differ from template).
	SLA *SLA `json:"sla,omitempty" yaml:"sla,omitempty"`

	// Created is when the contract was created.
	Created string `json:"created" yaml:"created"`

	// Expires is when the contract expires (optional).
	Expires *string `json:"expires,omitempty" yaml:"expires,omitempty"`

	// Nonce provides replay protection.
	Nonce string `json:"nonce" yaml:"nonce"`

	// ProviderSig is the provider's signature of this binding.
	ProviderSig string `json:"provider_sig,omitempty" yaml:"provider_sig,omitempty"`

	// ConsumerSig is the consumer's signature of this binding.
	ConsumerSig string `json:"consumer_sig,omitempty" yaml:"consumer_sig,omitempty"`
}

// Duration returns the time since creation.
func (b *Binding) Duration() time.Duration {
	created, err := time.Parse(time.RFC3339, b.Created)
	if err != nil {
		return 0
	}
	return time.Since(created)
}
