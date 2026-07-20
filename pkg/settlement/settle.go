// Package settlement handles economic settlement between agents:
// payment processing, receipt generation, and reconciliation.
package settlement

import (
	"fmt"
	"time"
)

// Model represents the settlement model used.
type Model string

const (
	ModelPrepay        Model = "prepay"
	ModelPostpay       Model = "postpay"
	ModelSubscription  Model = "subscription"
	ModelRevenueShare  Model = "revenue_share"
)

// Status represents the settlement lifecycle.
type Status string

const (
	StatusPending   Status = "pending"
	StatusSettled   Status = "settled"
	StatusFailed    Status = "failed"
	StatusDisputed  Status = "disputed"
)

// Transaction records a single settlement transaction.
type Transaction struct {
	TransactionID string `json:"transaction_id" yaml:"transaction_id"`
	ContractID    string `json:"contract_id" yaml:"contract_id"`
	ActionID      string `json:"action_id" yaml:"action_id"`
	Provider      string `json:"provider" yaml:"provider"`
	Consumer      string `json:"consumer" yaml:"consumer"`
	Amount        int64  `json:"amount" yaml:"amount"`
	Currency      string `json:"currency" yaml:"currency"`
	Model         Model  `json:"model" yaml:"model"`
	Status        Status `json:"status" yaml:"status"`
	Created       string `json:"created" yaml:"created"`
	SettledAt     string `json:"settled_at,omitempty" yaml:"settled_at,omitempty"`
	Reference     string `json:"reference,omitempty" yaml:"reference,omitempty"`
}

// Receipt is a settlement receipt, verifiable by both parties.
type Receipt struct {
	TransactionID string `json:"transaction_id" yaml:"transaction_id"`
	ContractID    string `json:"contract_id" yaml:"contract_id"`
	TotalAmount   int64  `json:"total_amount" yaml:"total_amount"`
	Currency      string `json:"currency" yaml:"currency"`
	ActionCount   int    `json:"action_count" yaml:"action_count"`
	From          string `json:"from" yaml:"from"`
	To            string `json:"to" yaml:"to"`
	Timestamp     string `json:"timestamp" yaml:"timestamp"`
	ProviderSig   string `json:"provider_sig,omitempty" yaml:"provider_sig,omitempty"`
	ConsumerSig   string `json:"consumer_sig,omitempty" yaml:"consumer_sig,omitempty"`
}

// Ledger tracks settlement transactions.
type Ledger struct {
	transactions []Transaction
	receipts     []Receipt
}

// NewLedger creates a new settlement ledger.
func NewLedger() *Ledger {
	return &Ledger{
		transactions: make([]Transaction, 0),
		receipts:     make([]Receipt, 0),
	}
}

// RecordTransaction records a new settlement transaction.
func (l *Ledger) RecordTransaction(provider, consumer, contractID, actionID string, amount int64, currency string, model Model) *Transaction {
	t := &Transaction{
		TransactionID: fmt.Sprintf("txn_%s_%d", contractID, time.Now().UnixNano()),
		ContractID:    contractID,
		ActionID:      actionID,
		Provider:      provider,
		Consumer:      consumer,
		Amount:        amount,
		Currency:      currency,
		Model:         model,
		Status:        StatusPending,
		Created:       time.Now().UTC().Format(time.RFC3339),
	}
	l.transactions = append(l.transactions, *t)
	return t
}

// Settle marks a transaction as settled and generates a receipt.
func (l *Ledger) Settle(txnID string) (*Receipt, error) {
	for i, txn := range l.transactions {
		if txn.TransactionID == txnID {
			now := time.Now().UTC().Format(time.RFC3339)
			l.transactions[i].Status = StatusSettled
			l.transactions[i].SettledAt = now

			receipt := &Receipt{
				TransactionID: txn.TransactionID,
				ContractID:    txn.ContractID,
				TotalAmount:   txn.Amount,
				Currency:      txn.Currency,
				ActionCount:   1,
				From:          txn.Consumer,
				To:            txn.Provider,
				Timestamp:     now,
			}
			l.receipts = append(l.receipts, *receipt)
			return receipt, nil
		}
	}
	return nil, fmt.Errorf("aip/settlement: transaction not found: %s", txnID)
}
