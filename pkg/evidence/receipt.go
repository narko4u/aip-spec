// Package evidence defines evidence receipts — signed attestations
// that an action was executed under a specific contract. Evidence receipts
// are designed to be verifiable by WitnessOS.
package evidence

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/narko4u/aip-spec/internal/crypto"
	"github.com/narko4u/aip-spec/pkg/execution"
)

// Receipt is a signed attestation of an action execution.
type Receipt struct {
	ReceiptID  string `json:"receipt_id" yaml:"receipt_id"`
	AIPVersion string `json:"aip_version" yaml:"aip_version"`
	ContractID string `json:"contract_id" yaml:"contract_id"`
	ActionID   string `json:"action_id" yaml:"action_id"`
	InputHash  string `json:"input_hash" yaml:"input_hash"`
	OutputHash string `json:"output_hash" yaml:"output_hash"`
	Provider   string `json:"provider" yaml:"provider"`
	Consumer   string `json:"consumer" yaml:"consumer"`
	Timestamp  string `json:"timestamp" yaml:"timestamp"`
	DurationMs int64  `json:"duration_ms" yaml:"duration_ms"`
	Signature  string `json:"signature,omitempty" yaml:"signature,omitempty"`
}

// NewReceipt creates an evidence receipt from an execution result.
func NewReceipt(result *execution.Result, contractID, provider, consumer string) *Receipt {
	inputHash := sha256.Sum256(result.Output)
	outputHash := sha256.Sum256(result.Output)

	return &Receipt{
		ReceiptID:  result.ReceiptID,
		AIPVersion: "0.1",
		ContractID: contractID,
		ActionID:   result.ActionID,
		InputHash:  hex.EncodeToString(inputHash[:]),
		OutputHash: hex.EncodeToString(outputHash[:]),
		Provider:   provider,
		Consumer:   consumer,
		Timestamp:  result.Timestamp,
		DurationMs: result.DurationMs,
	}
}

// Sign signs the receipt with the given key pair.
func (r *Receipt) Sign(kp *crypto.KeyPair) error {
	data, err := r.serializeForSigning()
	if err != nil {
		return fmt.Errorf("aip/evidence: serialize for signing: %w", err)
	}
	r.Signature = kp.Sign(data)
	return nil
}

// Verify checks the receipt's signature against a hex-encoded public key.
func (r *Receipt) Verify(pubKeyHex string) (bool, error) {
	pubKeyBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return false, fmt.Errorf("aip/evidence: decode public key: %w", err)
	}
	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return false, fmt.Errorf("aip/evidence: invalid public key length: %d", len(pubKeyBytes))
	}

	data, err := r.serializeForSigning()
	if err != nil {
		return false, err
	}

	pubKey := ed25519.PublicKey(pubKeyBytes)
	return crypto.Verify(pubKey, data, r.Signature), nil
}

func (r *Receipt) serializeForSigning() ([]byte, error) {
	clone := *r
	clone.Signature = ""
	return json.Marshal(clone)
}

// Store is a simple in-memory evidence store.
type Store struct {
	receipts map[string]*Receipt
}

// NewStore creates a new evidence store.
func NewStore() *Store {
	return &Store{
		receipts: make(map[string]*Receipt),
	}
}

// Store saves a receipt.
func (s *Store) Store(r *Receipt) {
	s.receipts[r.ReceiptID] = r
}

// Get retrieves a receipt by ID.
func (s *Store) Get(id string) (*Receipt, bool) {
	r, ok := s.receipts[id]
	return r, ok
}
