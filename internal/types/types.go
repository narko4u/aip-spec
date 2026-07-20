package types

import "time"

// AgentID identifies a negotiating party in the AIP protocol.
type AgentID string

// ActionID uniquely identifies an Action Schema.
type ActionID string

// ContractID uniquely identifies a negotiated contract.
type ContractID string

// SessionID uniquely identifies a negotiation session.
type SessionID string

// Nonce is a unique value for replay protection.
type Nonce string

// Currency represents a monetary unit (e.g. "USD", "AUD", "USDC").
type Currency string

// Amount represents a quantity of currency in the smallest unit (cents/satoshi).
type Amount int64

// Percentage represents a ratio (0.0 – 100.0).
type Percentage float64

// Duration represents a time span in seconds.
type Duration int64

// Timestamp is an RFC 3339 timestamp.
type Timestamp string

func NowTimestamp() Timestamp {
	return Timestamp(time.Now().UTC().Format(time.RFC3339))
}

// URL is a well-known location for a schema or endpoint.
type URL string

// HTTPMethod represents an HTTP verb.
type HTTPMethod string

const (
	MethodGET  HTTPMethod = "GET"
	MethodPOST HTTPMethod = "POST"
	MethodPUT  HTTPMethod = "PUT"
	MethodDEL  HTTPMethod = "DELETE"
)

// ContentType represents a media type.
type ContentType string

const (
	ContentTypeJSON      ContentType = "application/json"
	ContentTypeYAML      ContentType = "application/x-yaml"
	ContentTypeProtobuf  ContentType = "application/protobuf"
)

// TransportProtocol represents the protocol used for action execution.
type TransportProtocol string

const (
	TransportHTTP    TransportProtocol = "https"
	TransportMCP     TransportProtocol = "mcp"
	TransportGRPC    TransportProtocol = "grpc"
	TransportWS      TransportProtocol = "websocket"
)

// AIPVersion is the protocol version.
type AIPVersion string

const (
	AIPVersion0_1 AIPVersion = "0.1"
)
