package contract

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/narko4u/aip-spec/internal/crypto"
)

// signableBinding contains only the fields that are covered by signatures,
// excluding ProviderSig and ConsumerSig.
type signableBinding struct {
	ContractID string   `json:"contract_id"`
	TemplateID string   `json:"template_id"`
	Status     Status   `json:"status"`
	Provider   string   `json:"provider"`
	Consumer   string   `json:"consumer"`
	ActionID   string   `json:"action_id"`
	Pricing    *Pricing `json:"pricing"`
	SLA        *SLA     `json:"sla,omitempty"`
	Created    string   `json:"created"`
	Expires    *string  `json:"expires,omitempty"`
	Nonce      string   `json:"nonce"`
}

// CanonicalBytes returns a deterministic, canonical JSON representation
// of the binding excluding ProviderSig and ConsumerSig fields.
// This is what gets signed and verified.
func (b *Binding) CanonicalBytes() ([]byte, error) {
	sb := signableBinding{
		ContractID: b.ContractID,
		TemplateID: b.TemplateID,
		Status:     b.Status,
		Provider:   b.Provider,
		Consumer:   b.Consumer,
		ActionID:   b.ActionID,
		Pricing:    b.Pricing,
		SLA:        b.SLA,
		Created:    b.Created,
		Expires:    b.Expires,
		Nonce:      b.Nonce,
	}
	return json.Marshal(sb)
}

// SignBinding signs the binding with the given KeyPair and populates the
// ProviderSig or ConsumerSig field based on the party identifier.
// party must be either "provider" or "consumer".
func (b *Binding) SignBinding(kp *crypto.KeyPair, party string) error {
	canonical, err := b.CanonicalBytes()
	if err != nil {
		return fmt.Errorf("aip/contract: canonical marshal: %w", err)
	}

	sig := kp.Sign(canonical)

	switch strings.ToLower(party) {
	case "provider":
		b.ProviderSig = sig
	case "consumer":
		b.ConsumerSig = sig
	default:
		return fmt.Errorf("aip/contract: unknown party %q, expected 'provider' or 'consumer'", party)
	}
	return nil
}

// VerifyBindingSignature verifies a party's signature on a binding.
// party must be either "provider" or "consumer".
// pubKeyHex is the hex-encoded Ed25519 public key of the signing party.
func VerifyBindingSignature(pubKeyHex string, b *Binding, party string) bool {
	pubKey, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return false
	}

	canonical, err := b.CanonicalBytes()
	if err != nil {
		return false
	}

	var sig string
	switch strings.ToLower(party) {
	case "provider":
		sig = b.ProviderSig
	case "consumer":
		sig = b.ConsumerSig
	default:
		return false
	}

	return crypto.Verify(ed25519.PublicKey(pubKey), canonical, sig)
}

// VerifyBindingSignatureWithKey verifies using an ed25519.PublicKey directly.
func VerifyBindingSignatureWithKey(pubKey ed25519.PublicKey, b *Binding, party string) bool {
	canonical, err := b.CanonicalBytes()
	if err != nil {
		return false
	}

	var sig string
	switch strings.ToLower(party) {
	case "provider":
		sig = b.ProviderSig
	case "consumer":
		sig = b.ConsumerSig
	default:
		return false
	}

	return crypto.Verify(pubKey, canonical, sig)
}

// HasBothSignatures returns true if both ProviderSig and ConsumerSig are set.
func (b *Binding) HasBothSignatures() bool {
	return b.ProviderSig != "" && b.ConsumerSig != ""
}

// Signatures returns the signing parties for which signatures exist.
func (b *Binding) Signatures() []string {
	var parties []string
	if b.ProviderSig != "" {
		parties = append(parties, "provider")
	}
	if b.ConsumerSig != "" {
		parties = append(parties, "consumer")
	}
	sort.Strings(parties)
	return parties
}
