// Package negotiation implements the AIP negotiation state machine.
// It manages offer/counter-offer flows between agents to form binding contracts.
package negotiation

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/empirelabs/aip/internal/crypto"
	"github.com/empirelabs/aip/pkg/contract"
)

// SessionState represents the current state in a negotiation session.
type SessionState string

const (
	StateInitiated SessionState = "initiated"
	StateOffered   SessionState = "offered"
	StateCountered SessionState = "countered"
	StateAccepted  SessionState = "accepted"
	StateDeclined  SessionState = "declined"
	StateExpired   SessionState = "expired"
)

// Offer is a proposal from one agent to another.
type Offer struct {
	SessionID string          `json:"session_id" yaml:"session_id"`
	From      string          `json:"from" yaml:"from"`
	To        string          `json:"to" yaml:"to"`
	ActionID  string          `json:"action_id" yaml:"action_id"`
	Params    json.RawMessage `json:"params,omitempty" yaml:"params,omitempty"`
	Signature string          `json:"signature,omitempty" yaml:"signature,omitempty"`
	Timestamp string          `json:"timestamp" yaml:"timestamp"`
	Nonce     string          `json:"nonce" yaml:"nonce"`
}

// Session tracks a single negotiation conversation.
type Session struct {
	mu sync.RWMutex

	ID        SessionState `json:"id"`
	State     SessionState `json:"state"`
	Provider  string       `json:"provider"`
	Consumer  string       `json:"consumer"`
	ActionID  string       `json:"action_id"`
	Offers    []Offer      `json:"offers"`
	Binding   *contract.Binding `json:"binding,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
	ExpiresAt time.Time    `json:"expires_at"`
}

// Engine manages multiple concurrent negotiation sessions.
type Engine struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	keyPair  *crypto.KeyPair
}

// NewEngine creates a new negotiation engine with the given identity key pair.
func NewEngine(kp *crypto.KeyPair) *Engine {
	return &Engine{
		sessions: make(map[string]*Session),
		keyPair:  kp,
	}
}

// NewSession creates a new negotiation session.
func (e *Engine) NewSession(provider, consumer, actionID string, ttl time.Duration) *Session {
	e.mu.Lock()
	defer e.mu.Unlock()

	id := fmt.Sprintf("neg_%s_%s_%d", provider, consumer, time.Now().UnixNano())
	session := &Session{
		mu:        sync.RWMutex{},
		ID:        SessionState(id),
		State:     StateInitiated,
		Provider:  provider,
		Consumer:  consumer,
		ActionID:  actionID,
		Offers:    []Offer{},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}
	e.sessions[id] = session
	return session
}

// SubmitOffer submits a new offer into the negotiation flow.
// Returns an updated Binding if the offer results in acceptance.
func (e *Engine) SubmitOffer(offer Offer) (*contract.Binding, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	session, ok := e.sessions[offer.SessionID]
	if !ok {
		return nil, fmt.Errorf("aip/negotiation: session not found: %s", offer.SessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if time.Now().After(session.ExpiresAt) {
		session.State = StateExpired
		return nil, fmt.Errorf("aip/negotiation: session expired")
	}

	// Append the offer
	session.Offers = append(session.Offers, offer)

	// Simple state machine:
	// First offer from consumer → StateOffered
	// Counter-offer from provider → StateCountered
	// If terms match template → auto-accept for v0.1
	switch session.State {
	case StateInitiated:
		session.State = StateOffered
	case StateOffered:
		// Provider is responding (counter-offer or acceptance)
		// For v0.1: auto-accept and create binding
		binding := &contract.Binding{
			ContractID: fmt.Sprintf("ct_%s_%d", session.ID, time.Now().UnixNano()),
			TemplateID: "", // Will be populated from template reference
			Status:     contract.StatusActive,
			Provider:   session.Provider,
			Consumer:   session.Consumer,
			ActionID:   session.ActionID,
			Created:    time.Now().UTC().Format(time.RFC3339),
			Nonce:      offer.Nonce,
		}
		session.Binding = binding
		session.State = StateAccepted
		return binding, nil
	case StateCountered:
		session.State = StateAccepted
	default:
		return nil, fmt.Errorf("aip/negotiation: invalid state transition: %s", session.State)
	}

	return nil, nil
}

// SignOffer signs an offer with the engine's private key.
func (e *Engine) SignOffer(offer *Offer) error {
	data, err := json.Marshal(offer)
	if err != nil {
		return fmt.Errorf("aip/negotiation: marshal offer: %w", err)
	}
	offer.Signature = e.keyPair.Sign(data)
	return nil
}

// VerifyOffer verifies an offer's signature.
func VerifyOffer(offer *Offer, publicKey ed25519.PublicKey) bool {
	data, _ := json.Marshal(struct {
		SessionID string          `json:"session_id"`
		From      string          `json:"from"`
		To        string          `json:"to"`
		ActionID  string          `json:"action_id"`
		Params    json.RawMessage `json:"params,omitempty"`
		Timestamp string          `json:"timestamp"`
		Nonce     string          `json:"nonce"`
	}{
		SessionID: offer.SessionID,
		From:      offer.From,
		To:        offer.To,
		ActionID:  offer.ActionID,
		Params:    offer.Params,
		Timestamp: offer.Timestamp,
		Nonce:     offer.Nonce,
	})
	return crypto.Verify(publicKey, data, offer.Signature)
}
