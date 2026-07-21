// Package negotiation implements the AIP negotiation state machine.
// It manages offer/counter-offer flows between agents to form binding contracts.
package negotiation

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/narko4u/aip-spec/internal/crypto"
	"github.com/narko4u/aip-spec/pkg/contract"
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

// PartialAcceptance allows accepting some terms while countering others.
type PartialAcceptance struct {
	SessionID      string          `json:"session_id"`
	Party          string          `json:"party"`
	AcceptedTerms  json.RawMessage `json:"accepted_terms,omitempty"`
	CounteredTerms json.RawMessage `json:"countered_terms,omitempty"`
	Timestamp      string          `json:"timestamp"`
	Nonce          string          `json:"nonce"`
}

// TransitionRule defines a valid state transition.
type TransitionRule struct {
	From SessionState
	To   SessionState
	Desc string
}

// validTransitions defines the complete state machine.
var validTransitions = map[SessionState]map[SessionState]string{
	StateInitiated: {
		StateOffered: "consumer submits first offer",
		StateExpired: "session timeout",
		StateDeclined: "session rejected before first offer",
	},
	StateOffered: {
		StateAccepted:  "provider accepts offer",
		StateCountered: "provider submits counter-offer",
		StateDeclined:  "provider declines offer",
		StateExpired:   "session timeout",
	},
	StateCountered: {
		StateAccepted:  "consumer accepts counter-offer",
		StateCountered: "consumer counters back",
		StateDeclined:  "consumer declines offer",
		StateExpired:   "session timeout",
	},
	StateAccepted: {},
	StateDeclined: {},
	StateExpired:  {},
}

// Session tracks a single negotiation conversation.
type Session struct {
	mu sync.RWMutex

	ID        SessionState       `json:"id"`
	State     SessionState       `json:"state"`
	Provider  string             `json:"provider"`
	Consumer  string             `json:"consumer"`
	ActionID  string             `json:"action_id"`
	Offers    []Offer            `json:"offers"`
	Binding   *contract.Binding  `json:"binding,omitempty"`
	CreatedAt time.Time          `json:"created_at"`
	ExpiresAt time.Time          `json:"expires_at"`
}

// IsExpired returns true if the session has passed its expiration time.
func (s *Session) IsExpired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Now().After(s.ExpiresAt)
}

// checkExpired transitions the session to StateExpired if expired, and returns true.
func (s *Session) checkExpired() bool {
	if time.Now().After(s.ExpiresAt) && s.State != StateExpired && s.State != StateAccepted && s.State != StateDeclined {
		s.State = StateExpired
		return true
	}
	return false
}

// Engine manages multiple concurrent negotiation sessions.
type Engine struct {
	mu            sync.RWMutex
	sessions      map[string]*Session
	providerKey   *crypto.KeyPair
	consumerKey   *crypto.KeyPair
	currentAction string
}

// NewEngine creates a new negotiation engine with the given identity key pair.
// The key pair is used as the engine operator's identity (provider by default).
func NewEngine(kp *crypto.KeyPair) *Engine {
	return &Engine{
		sessions:    make(map[string]*Session),
		providerKey: kp,
	}
}

// SetConsumerKey sets the consumer key pair for signing consumer-side offers.
func (e *Engine) SetConsumerKey(kp *crypto.KeyPair) {
	e.providerKey = kp
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

// GetSession retrieves a session by ID.
func (e *Engine) GetSession(sessionID string) *Session {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.sessions[sessionID]
}

// checkTransition validates whether a state transition is allowed.
func checkTransition(from, to SessionState) error {
	allowed, ok := validTransitions[from]
	if !ok {
		return fmt.Errorf("aip/negotiation: unknown source state %q", from)
	}
	if _, ok := allowed[to]; !ok {
		return fmt.Errorf("aip/negotiation: invalid transition %s -> %s", from, to)
	}
	return nil
}

// SubmitOffer submits a new offer into the negotiation flow.
// - StateInitiated → StateOffered (first consumer offer)
// - StateOffered → StateCountered (provider counter-offer)
// - StateCountered → StateCountered (consumer counter-counter-offer)
// Returns an updated Binding if the offer results in immediate acceptance.
func (e *Engine) SubmitOffer(offer Offer) (*contract.Binding, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	session, ok := e.sessions[offer.SessionID]
	if !ok {
		return nil, fmt.Errorf("aip/negotiation: session not found: %s", offer.SessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Check expiration first
	if session.checkExpired() {
		return nil, fmt.Errorf("aip/negotiation: session expired")
	}

	// Validate offer direction based on current state
	consumerFirst := session.State == StateInitiated
	if consumerFirst && offer.From != session.Consumer {
		return nil, fmt.Errorf("aip/negotiation: first offer must come from consumer %q, got %q", session.Consumer, offer.From)
	}
	if session.State == StateOffered && offer.From != session.Provider {
		return nil, fmt.Errorf("aip/negotiation: counter-offer must come from provider %q, got %q", session.Provider, offer.From)
	}
	if session.State == StateCountered && offer.From != session.Consumer {
		return nil, fmt.Errorf("aip/negotiation: counter-response must come from consumer %q, got %q", session.Consumer, offer.From)
	}

	// Determine target state
	var target SessionState
	switch session.State {
	case StateInitiated:
		target = StateOffered
	case StateOffered:
		target = StateCountered
	case StateCountered:
		target = StateCountered
	default:
		return nil, fmt.Errorf("aip/negotiation: cannot submit offer in state %s", session.State)
	}

	if err := checkTransition(session.State, target); err != nil {
		return nil, err
	}

	// Append the offer
	session.Offers = append(session.Offers, offer)
	session.State = target

	return nil, nil
}

// AcceptOffer accepts the current offer, creating a binding contract.
// Provider accepts in StateOffered, Consumer accepts in StateCountered.
func (e *Engine) AcceptOffer(sessionID, party string) (*contract.Binding, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	session, ok := e.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("aip/negotiation: session not found: %s", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Check expiration first
	if session.checkExpired() {
		return nil, fmt.Errorf("aip/negotiation: session expired")
	}

	// Validate who can accept based on state
	var target SessionState
	switch session.State {
	case StateOffered:
		if party != session.Provider {
			return nil, fmt.Errorf("aip/negotiation: only provider %q can accept in state %s, got %q", session.Provider, session.State, party)
		}
		target = StateAccepted
	case StateCountered:
		if party != session.Consumer {
			return nil, fmt.Errorf("aip/negotiation: only consumer %q can accept in state %s, got %q", session.Consumer, session.State, party)
		}
		target = StateAccepted
	default:
		return nil, fmt.Errorf("aip/negotiation: cannot accept in state %s", session.State)
	}

	if err := checkTransition(session.State, target); err != nil {
		return nil, err
	}

	// Use the most recent offer's terms
	lastOffer := session.Offers[len(session.Offers)-1]

	binding := &contract.Binding{
		ContractID: fmt.Sprintf("ct_%s_%d", session.ID, time.Now().UnixNano()),
		TemplateID: "",
		Status:     contract.StatusActive,
		Provider:   session.Provider,
		Consumer:   session.Consumer,
		ActionID:   session.ActionID,
		Created:    time.Now().UTC().Format(time.RFC3339),
		Nonce:      lastOffer.Nonce,
	}

	// Parse offer params into pricing/SLA if available
	if len(lastOffer.Params) > 0 {
		var params struct {
			Pricing *contract.Pricing `json:"pricing,omitempty"`
			SLA     *contract.SLA     `json:"sla,omitempty"`
		}
		if err := json.Unmarshal(lastOffer.Params, &params); err == nil {
			binding.Pricing = params.Pricing
			binding.SLA = params.SLA
		}
	}

	session.Binding = binding
	session.State = StateAccepted

	return binding, nil
}

// DeclineOffer declines the current offer, moving the session to StateDeclined.
func (e *Engine) DeclineOffer(sessionID, party string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	session, ok := e.sessions[sessionID]
	if !ok {
		return fmt.Errorf("aip/negotiation: session not found: %s", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Check expiration first
	if session.checkExpired() {
		return fmt.Errorf("aip/negotiation: session expired")
	}

	// Validate who can decline based on state
	switch session.State {
	case StateOffered:
		if party != session.Provider {
			return fmt.Errorf("aip/negotiation: only provider %q can decline in state %s, got %q", session.Provider, session.State, party)
		}
	case StateCountered:
		if party != session.Consumer {
			return fmt.Errorf("aip/negotiation: only consumer %q can decline in state %s, got %q", session.Consumer, session.State, party)
		}
	case StateInitiated:
		// Either party can decline an initiated session
	default:
		return fmt.Errorf("aip/negotiation: cannot decline in state %s", session.State)
	}

	if err := checkTransition(session.State, StateDeclined); err != nil {
		return err
	}

	session.State = StateDeclined
	return nil
}

// SignOffer signs an offer with the engine's private key.
func (e *Engine) SignOffer(offer *Offer) error {
	data, err := json.Marshal(offer)
	if err != nil {
		return fmt.Errorf("aip/negotiation: marshal offer: %w", err)
	}
	offer.Signature = e.providerKey.Sign(data)
	return nil
}

// SignOfferWith signs an offer with a specific key pair.
func (e *Engine) SignOfferWith(offer *Offer, kp *crypto.KeyPair) error {
	data, err := json.Marshal(offer)
	if err != nil {
		return fmt.Errorf("aip/negotiation: marshal offer: %w", err)
	}
	offer.Signature = kp.Sign(data)
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

// SignBindingOnAccept signs the binding on the session with the appropriate party's key.
// This should be called after AcceptOffer to populate ProviderSig and ConsumerSig.
func (e *Engine) SignBindingOnAccept(sessionID string, providerKP, consumerKP *crypto.KeyPair) error {
	e.mu.RLock()
	session, ok := e.sessions[sessionID]
	e.mu.RUnlock()
	if !ok {
		return fmt.Errorf("aip/negotiation: session not found: %s", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.Binding == nil {
		return fmt.Errorf("aip/negotiation: no binding to sign")
	}

	// Sign with provider key
	if providerKP != nil {
		if err := session.Binding.SignBinding(providerKP, "provider"); err != nil {
			return fmt.Errorf("aip/negotiation: provider signature: %w", err)
		}
	}

	// Sign with consumer key
	if consumerKP != nil {
		if err := session.Binding.SignBinding(consumerKP, "consumer"); err != nil {
			return fmt.Errorf("aip/negotiation: consumer signature: %w", err)
		}
	}

	return nil
}
