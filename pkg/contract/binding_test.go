// AIP (Agent Interaction Protocol)
// Copyright (c) 2026 Empire Labs Pty Ltd
// SPDX-License-Identifier: MIT

package contract

import (
	"bytes"
	"strings"
	"testing"
)

func aBinding() *Binding {
	return &Binding{
		ContractID:  "ctr_01",
		TemplateID:  "tpl_01",
		Status:      StatusActive,
		Provider:    "did:aip:provider",
		Consumer:    "did:aip:consumer",
		ActionID:    "act_01",
		Pricing:     &Pricing{Model: "flat", Amount: 2500, Currency: "AUD"},
		SLA:         &SLA{MaxLatencyMs: 900},
		Created:     "2026-10-31T00:00:00Z",
		Nonce:       "n-1",
		ProviderSig: "provider-signature-that-must-not-be-signed",
		ConsumerSig: "consumer-signature-that-must-not-be-signed",
	}
}

func TestCanonicalBytesIsStable(t *testing.T) {
	b := aBinding()

	first, err := b.CanonicalBytes()
	if err != nil {
		t.Fatalf("CanonicalBytes: %v", err)
	}
	second, err := aBinding().CanonicalBytes()
	if err != nil {
		t.Fatalf("CanonicalBytes: %v", err)
	}

	if !bytes.Equal(first, second) {
		t.Fatalf("two marshals of the same binding differ:\n%s\n%s", first, second)
	}
}

func TestCanonicalBytesChangesWithTheAmount(t *testing.T) {
	cheap, err := aBinding().CanonicalBytes()
	if err != nil {
		t.Fatalf("CanonicalBytes: %v", err)
	}

	expensiveBinding := aBinding()
	expensiveBinding.Pricing.Amount = 9900

	expensive, err := expensiveBinding.CanonicalBytes()
	if err != nil {
		t.Fatalf("CanonicalBytes: %v", err)
	}

	if bytes.Equal(cheap, expensive) {
		t.Fatal("changing the pricing amount did not change the canonical bytes")
	}
}

func TestCanonicalBytesExcludesSignatures(t *testing.T) {
	canonical, err := aBinding().CanonicalBytes()
	if err != nil {
		t.Fatalf("CanonicalBytes: %v", err)
	}

	text := string(canonical)
	for _, field := range []string{"provider_sig", "consumer_sig"} {
		if strings.Contains(text, field) {
			t.Fatalf("canonical bytes contain %q: %s", field, text)
		}
	}
}

func TestCanonicalBytesTable(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Binding)
		same   bool
	}{
		{name: "stable-name", mutate: func(b *Binding) {}, same: true},
		{name: "changed-amount", mutate: func(b *Binding) { b.Pricing.Amount = 9900 }, same: false},
	}

	base, err := aBinding().CanonicalBytes()
	if err != nil {
		t.Fatalf("CanonicalBytes: %v", err)
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			other := aBinding()
			c.mutate(other)
			got, err := other.CanonicalBytes()
			if err != nil {
				t.Fatalf("CanonicalBytes: %v", err)
			}
			if same := bytes.Equal(base, got); same != c.same {
				t.Fatalf("%s: bytes equal = %v, want %v", c.name, same, c.same)
			}
		})
	}
}
