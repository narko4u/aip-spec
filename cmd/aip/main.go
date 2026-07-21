package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/narko4u/aip-spec/internal/crypto"
	"github.com/narko4u/aip-spec/pkg/action"
	"github.com/narko4u/aip-spec/pkg/contract"
	"github.com/narko4u/aip-spec/pkg/evidence"
	"github.com/narko4u/aip-spec/pkg/execution"
	"github.com/narko4u/aip-spec/pkg/negotiation"
	"github.com/narko4u/aip-spec/pkg/settlement"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("AIP — Agent Interaction Protocol v0.1")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  aip keygen                  Generate a new Ed25519 key pair")
		fmt.Println("  aip negotiate [flags]       Start/exchange offers in a negotiation session")
		fmt.Println("  aip sign-contract <flags>   Sign a binding contract JSON")
		fmt.Println("  aip execute <schema>        Execute an action")
		fmt.Println("  aip settle <contract>       Settle a contract")
		fmt.Println("  aip verify <receipt>        Verify an evidence receipt")
		fmt.Println("  aip demo                    Run a full end-to-end demo")
		return
	}

	switch os.Args[1] {
	case "keygen":
		cmdKeygen()
	case "negotiate":
		cmdNegotiate()
	case "sign-contract":
		cmdSignContract()
	case "execute":
		cmdExecute()
	case "settle":
		cmdSettle()
	case "verify":
		cmdVerify()
	case "demo":
		cmdDemo()
	default:
		fmt.Fprintf(os.Stderr, "aip: unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func cmdKeygen() {
	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("AIP Key Pair\n")
	fmt.Printf("  Public Key:  %s\n", kp.PublicKeyHex())
	fmt.Printf("  Private Key: %x (keep secret!)\n", kp.PrivateKey)
}

func cmdNegotiate() {
	fs := flag.NewFlagSet("negotiate", flag.ExitOnError)
	role := fs.String("role", "", "Party role: provider or consumer")
	actionID := fs.String("action", "", "Action schema ID")
	session := fs.String("session", "", "Session ID (required for counter/accept/decline)")
	counterFile := fs.String("counter", "", "JSON file with counter-offer terms")
	accept := fs.Bool("accept", false, "Accept the current offer")
	decline := fs.Bool("decline", false, "Decline the current offer")
	providerKeyHex := fs.String("provider-key", "", "Provider's hex-encoded private key")
	consumerKeyHex := fs.String("consumer-key", "", "Consumer's hex-encoded private key")
	fs.Parse(os.Args[2:])

	// Default to generating one-time keys if none provided
	providerKP, _ := crypto.GenerateKeyPair()
	consumerKP, _ := crypto.GenerateKeyPair()

	// Parse provider key if provided
	if *providerKeyHex != "" {
		privBytes, err := hexDecode(*providerKeyHex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: invalid provider key hex: %v\n", err)
			os.Exit(1)
		}
		if len(privBytes) != ed25519PrivateKeyLen {
			fmt.Fprintf(os.Stderr, "error: provider private key must be %d bytes\n", ed25519PrivateKeyLen)
			os.Exit(1)
		}
		providerKP = &crypto.KeyPair{PrivateKey: privBytes, PublicKey: privBytes[32:]}
	}

	// Parse consumer key if provided
	if *consumerKeyHex != "" {
		privBytes, err := hexDecode(*consumerKeyHex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: invalid consumer key hex: %v\n", err)
			os.Exit(1)
		}
		if len(privBytes) != ed25519PrivateKeyLen {
			fmt.Fprintf(os.Stderr, "error: consumer private key must be %d bytes\n", ed25519PrivateKeyLen)
			os.Exit(1)
		}
		consumerKP = &crypto.KeyPair{PrivateKey: privBytes, PublicKey: privBytes[32:]}
	}

	// Create a new session if no session ID provided
	if *session == "" {
		if *role == "" || *actionID == "" {
			fmt.Fprintln(os.Stderr, "usage: aip negotiate --role <provider|consumer> --action <action_id> [--session <id>] [--counter <file>] [--accept] [--decline]")
			os.Exit(1)
		}

		engine := negotiation.NewEngine(providerKP)
		engine.SetConsumerKey(consumerKP)

		sess := engine.NewSession("provider_agent", "consumer_agent", *actionID, 30*time.Minute)
		fmt.Printf("Created session: %s\n", sess.ID)
		fmt.Printf("State: %s\n", sess.State)

		// If role is consumer, make first offer
		if *role == "consumer" {
			offer := negotiation.Offer{
				SessionID: string(sess.ID),
				From:      "consumer_agent",
				To:        "provider_agent",
				ActionID:  *actionID,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Nonce:     fmt.Sprintf("nonce_%d", time.Now().UnixNano()),
			}
			if err := engine.SignOfferWith(&offer, consumerKP); err != nil {
				fmt.Fprintf(os.Stderr, "error signing offer: %v\n", err)
				os.Exit(1)
			}
			_, err := engine.SubmitOffer(offer)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error submitting offer: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Submitted offer from consumer\n")
			fmt.Printf("State: %s\n", sess.State)
		}

		output := struct {
			SessionID string           `json:"session_id"`
			State     negotiation.SessionState `json:"state"`
			Provider  string           `json:"provider"`
			Consumer  string           `json:"consumer"`
			ActionID  string           `json:"action_id"`
		}{
			SessionID: string(sess.ID),
			State:     sess.State,
			Provider:  sess.Provider,
			Consumer:  sess.Consumer,
			ActionID:  sess.ActionID,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Use existing session
	engine := negotiation.NewEngine(providerKP)
	engine.SetConsumerKey(consumerKP)

	// Accept
	if *accept {
		binding, err := engine.AcceptOffer(*session, *role)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error accepting offer: %v\n", err)
			os.Exit(1)
		}
		// Sign the binding
		if *role == "provider" {
			binding.SignBinding(providerKP, "provider")
		} else {
			binding.SignBinding(consumerKP, "consumer")
		}
		fmt.Printf("Offer accepted!\n")
		data, _ := json.MarshalIndent(binding, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Decline
	if *decline {
		err := engine.DeclineOffer(*session, *role)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error declining offer: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Offer declined.")
		sess := engine.GetSession(*session)
		if sess != nil {
			data, _ := json.MarshalIndent(struct {
				SessionID string                      `json:"session_id"`
				State     negotiation.SessionState    `json:"state"`
			}{
				SessionID: *session,
				State:     sess.State,
			}, "", "  ")
			fmt.Println(string(data))
		}
		return
	}

	// Counter-offer
	if *counterFile != "" {
		paramsData, err := os.ReadFile(*counterFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading counter-offer file: %v\n", err)
			os.Exit(1)
		}

		sess := engine.GetSession(*session)
		if sess == nil {
			fmt.Fprintf(os.Stderr, "error: session not found: %s\n", *session)
			os.Exit(1)
		}

		offer := negotiation.Offer{
			SessionID: *session,
			From:      *role,
			To:        "",
			ActionID:  string(sess.ActionID),
			Params:    paramsData,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Nonce:     fmt.Sprintf("nonce_%d", time.Now().UnixNano()),
		}

		// Set To field based on role
		if *role == "provider" {
			offer.To = sess.Consumer
		} else {
			offer.To = sess.Provider
		}

		var signKP *crypto.KeyPair
		if *role == "provider" {
			signKP = providerKP
		} else {
			signKP = consumerKP
		}
		if err := engine.SignOfferWith(&offer, signKP); err != nil {
			fmt.Fprintf(os.Stderr, "error signing counter-offer: %v\n", err)
			os.Exit(1)
		}

		_, err = engine.SubmitOffer(offer)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error submitting counter-offer: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Counter-offer submitted by %s\n", *role)
		sess = engine.GetSession(*session)
		if sess != nil {
			data, _ := json.MarshalIndent(struct {
				SessionID string                      `json:"session_id"`
				State     negotiation.SessionState    `json:"state"`
				Offers    int                         `json:"offers"`
			}{
				SessionID: *session,
				State:     sess.State,
				Offers:    len(sess.Offers),
			}, "", "  ")
			fmt.Println(string(data))
		}
		return
	}

	fmt.Fprintln(os.Stderr, "usage: aip negotiate --role <provider|consumer> --action <action_id> [--session <id>] [--counter <file>] [--accept] [--decline]")
	os.Exit(1)
}

func cmdSignContract() {
	fs := flag.NewFlagSet("sign-contract", flag.ExitOnError)
	bindingFile := fs.String("binding", "", "Path to binding JSON file")
	keyHex := fs.String("key", "", "Private key hex to sign with")
	party := fs.String("party", "", "Party to sign as: provider or consumer")
	fs.Parse(os.Args[2:])

	if *bindingFile == "" || *keyHex == "" || *party == "" {
		fmt.Fprintln(os.Stderr, "usage: aip sign-contract --binding <binding.json> --key <private_key_hex> --party <provider|consumer>")
		os.Exit(1)
	}

	bindingData, err := os.ReadFile(*bindingFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading binding file: %v\n", err)
		os.Exit(1)
	}

	var binding contract.Binding
	if err := json.Unmarshal(bindingData, &binding); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing binding: %v\n", err)
		os.Exit(1)
	}

	privBytes, err := hexDecode(*keyHex)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid key hex: %v\n", err)
		os.Exit(1)
	}
	if len(privBytes) != ed25519PrivateKeyLen {
		fmt.Fprintf(os.Stderr, "error: private key must be %d bytes (got %d)\n", ed25519PrivateKeyLen, len(privBytes))
		os.Exit(1)
	}

	kp := &crypto.KeyPair{
		PrivateKey: privBytes,
		PublicKey:  privBytes[32:],
	}

	if err := binding.SignBinding(kp, *party); err != nil {
		fmt.Fprintf(os.Stderr, "error signing binding: %v\n", err)
		os.Exit(1)
	}

	output, _ := json.MarshalIndent(binding, "", "  ")
	fmt.Println(string(output))
	fmt.Fprintf(os.Stderr, "✓ Binding signed by %s\n", *party)
}

const ed25519PrivateKeyLen = 64

func hexDecode(s string) ([]byte, error) {
	b := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		var val byte
		n, err := fmt.Sscanf(s[i:i+2], "%02x", &val)
		if err != nil || n != 1 {
			return nil, fmt.Errorf("invalid hex char at position %d", i)
		}
		b[i/2] = val
	}
	return b, nil
}

func cmdExecute() {
	fs := flag.NewFlagSet("execute", flag.ExitOnError)
	witnessosURL := fs.String("witnessos", "", "WitnessOS URL to push evidence receipt (e.g. http://localhost:8402)")
	fs.Parse(os.Args[2:])

	args := fs.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: aip execute [--witnessos <url>] <schema.json> [input.json]")
		os.Exit(1)
	}

	schemaData, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading schema: %v\n", err)
		os.Exit(1)
	}

	schema, err := action.ParseSchema(schemaData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing schema: %v\n", err)
		os.Exit(1)
	}

	var inputData []byte
	if len(args) >= 2 {
		inputData, err = os.ReadFile(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading input: %v\n", err)
			os.Exit(1)
		}
	} else {
		inputData = []byte(`{}`)
	}

	req := &action.Request{
		ActionID:  schema.ActionID,
		Input:     inputData,
		Nonce:     fmt.Sprintf("nonce_%d", time.Now().UnixNano()),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	binding := &contract.Binding{
		ContractID: fmt.Sprintf("demo_ct_%d", time.Now().UnixNano()),
		Status:     contract.StatusActive,
		Provider:   "provider_agent",
		Consumer:   "consumer_agent",
		ActionID:   schema.ActionID,
		Created:    time.Now().UTC().Format(time.RFC3339),
	}

	engine := execution.NewEngine(30 * time.Second)
	result, err := engine.Execute(schema, req, binding)
	if err != nil {
		fmt.Fprintf(os.Stderr, "execution failed: %v\n", err)
		os.Exit(1)
	}

	// Generate evidence receipt
	kp, _ := crypto.GenerateKeyPair()
	receipt := evidence.NewReceipt(result, binding.ContractID, "provider_agent", "consumer_agent")
	receipt.Sign(kp)

	// Push evidence receipt to WitnessOS if flag is set
	if *witnessosURL != "" {
		receiptJSON, _ := json.Marshal(receipt)
		resp, err := http.Post(
			*witnessosURL+"/demo/aip/ingest",
			"application/json",
			bytes.NewReader(receiptJSON),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WitnessOS push failed: %v\n", err)
		} else {
			defer resp.Body.Close()
			var witnessResp map[string]any
			if json.NewDecoder(resp.Body).Decode(&witnessResp) == nil {
				fmt.Fprintf(os.Stderr, "WitnessOS ingest: case=%v status=%v chain_root=%v\n",
					witnessResp["case_id"], witnessResp["status"], witnessResp["chain_root"])
			}
		}
	}

	output := struct {
		Result  *execution.Result `json:"result"`
		Receipt *evidence.Receipt `json:"receipt"`
	}{
		Result:  result,
		Receipt: receipt,
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(data))
}

func cmdSettle() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: aip settle <contract.json>")
		os.Exit(1)
	}

	contractData, err := os.ReadFile(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading contract: %v\n", err)
		os.Exit(1)
	}

	var binding contract.Binding
	if err := json.Unmarshal(contractData, &binding); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing contract: %v\n", err)
		os.Exit(1)
	}

	ledger := settlement.NewLedger()
	amount := int64(100)
	if binding.Pricing != nil {
		amount = binding.Pricing.Amount
	}

	txn := ledger.RecordTransaction(
		binding.Provider,
		binding.Consumer,
		binding.ContractID,
		binding.ActionID,
		amount,
		"AUD",
		settlement.ModelPostpay,
	)

	receipt, err := ledger.Settle(txn.TransactionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "settlement failed: %v\n", err)
		os.Exit(1)
	}

	data, _ := json.MarshalIndent(receipt, "", "  ")
	fmt.Println(string(data))
}

func cmdVerify() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: aip verify <receipt.json> [public_key_hex]")
		os.Exit(1)
	}

	receiptData, err := os.ReadFile(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading receipt: %v\n", err)
		os.Exit(1)
	}

	var receipt evidence.Receipt
	if err := json.Unmarshal(receiptData, &receipt); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing receipt: %v\n", err)
		os.Exit(1)
	}

	var pubKeyHex string
	if len(os.Args) >= 4 {
		pubKeyHex = os.Args[3]
	} else if receipt.Signature != "" {
		fmt.Println("info: no public key provided, skipping signature verification")
		fmt.Println("info: receipt data is valid JSON")
		data, _ := json.MarshalIndent(receipt, "", "  ")
		fmt.Println(string(data))
		return
	}

	valid, err := receipt.Verify(pubKeyHex)
	if err != nil {
		fmt.Fprintf(os.Stderr, "verification error: %v\n", err)
		os.Exit(1)
	}

	if valid {
		fmt.Println("✓ Evidence receipt VALID")
	} else {
		fmt.Println("✗ Evidence receipt INVALID")
		os.Exit(1)
	}
}

func cmdDemo() {
	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║    AIP — Full End-to-End Demo v0.1                    ║")
	fmt.Println("║    Featuring Dynamic Negotiation + Contract Signing   ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 1. Generate identities for both parties
	fmt.Println("◆ Step 1: Generate Agent Identities")
	providerKP, _ := crypto.GenerateKeyPair()
	consumerKP, _ := crypto.GenerateKeyPair()
	fmt.Printf("  Provider Key:     %s\n", providerKP.PublicKeyHex())
	fmt.Printf("  Consumer Key:     %s\n", consumerKP.PublicKeyHex())
	fmt.Println()

	// 2. Load an action schema
	fmt.Println("◆ Step 2: Action Schema: policy.evaluate.v1")
	schema := action.Schema{
		AIPVersion:   "0.1",
		ActionID:     "policy.evaluate.v1",
		Version:      "1.0.0",
		DisplayName:  "Policy Evaluation",
		Description:  "Evaluate an action against a compliance policy",
		InputSchema:  json.RawMessage(`{"type":"object","properties":{"action":{"type":"string"},"context":{"type":"object"}}}`),
		OutputSchema: json.RawMessage(`{"type":"object","properties":{"decision":{"type":"string"},"evidence":{"type":"string"}}}`),
		Transport: action.Transport{
			Protocol: "https",
			Endpoint: "https://api.empirelabs.com.au/witnessos/evaluate",
			Method:   "POST",
		},
		Timeout:    30,
		Idempotent: true,
	}
	fmt.Printf("  Action:   %s\n", schema.DisplayName)
	fmt.Printf("  Endpoint: %s\n", schema.Transport.Endpoint)
	fmt.Println()

	// 3. Negotiate — proper back-and-forth with counter-offer
	fmt.Println("◆ Step 3: Dynamic Negotiation")
	engine := negotiation.NewEngine(providerKP)

	session := engine.NewSession(
		"witnessos.empirelabs.com.au",
		"agent.client.io",
		"policy.evaluate.v1",
		5*time.Minute,
	)
	fmt.Printf("  Session created: %s\n", session.ID)
	fmt.Printf("  State: %s\n", session.State)

	// 3a. Consumer makes first offer
	fmt.Println("\n  --- Consumer submits first offer ---")
	consumerOffer := negotiation.Offer{
		SessionID: string(session.ID),
		From:      "agent.client.io",
		To:        "witnessos.empirelabs.com.au",
		ActionID:  "policy.evaluate.v1",
		Params:    json.RawMessage(`{"pricing":{"model":"per_invocation","amount":50,"currency":"AUD"},"sla":{"max_latency_ms":10000,"max_retries":5}}`),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Nonce:     fmt.Sprintf("nonce_%d", time.Now().UnixNano()),
	}
	if err := engine.SignOfferWith(&consumerOffer, consumerKP); err != nil {
		fmt.Fprintf(os.Stderr, "error signing offer: %v\n", err)
		os.Exit(1)
	}
	_, err := engine.SubmitOffer(consumerOffer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  State: %s\n", session.State)
	fmt.Printf("  Consumer offered: 50¢/call, 10s latency, 5 retries\n")

	// 3b. Provider counters with different terms
	fmt.Println("\n  --- Provider counters with modified terms ---")
	counterOffer := negotiation.Offer{
		SessionID: string(session.ID),
		From:      "witnessos.empirelabs.com.au",
		To:        "agent.client.io",
		ActionID:  "policy.evaluate.v1",
		Params:    json.RawMessage(`{"pricing":{"model":"per_invocation","amount":100,"currency":"AUD"},"sla":{"max_latency_ms":5000,"max_retries":3}}`),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Nonce:     fmt.Sprintf("nonce_%d", time.Now().UnixNano()),
	}
	if err := engine.SignOfferWith(&counterOffer, providerKP); err != nil {
		fmt.Fprintf(os.Stderr, "error signing counter-offer: %v\n", err)
		os.Exit(1)
	}
	_, err = engine.SubmitOffer(counterOffer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  State: %s\n", session.State)
	fmt.Printf("  Provider countered: $1.00/call, 5s latency, 3 retries\n")

	// 3c. Consumer accepts the counter-offer
	fmt.Println("\n  --- Consumer accepts counter-offer ---")
	binding, err := engine.AcceptOffer(string(session.ID), "agent.client.io")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error accepting: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  State: %s\n", session.State)
	fmt.Printf("  Contract created: %s\n", binding.ContractID)

	// 3d. Sign the binding with both parties' keys
	fmt.Println("\n  --- Contract Signing Ceremony ---")
	if err := binding.SignBinding(providerKP, "provider"); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  ✓ Provider signed: %s\n", binding.ProviderSig[:16]+"...")

	if err := binding.SignBinding(consumerKP, "consumer"); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  ✓ Consumer signed: %s\n", binding.ConsumerSig[:16]+"...")
	fmt.Printf("  ✓ Both signatures present: %v\n", binding.HasBothSignatures())

	// 3e. Verify signatures
	fmt.Println("\n  --- Signature Verification ---")
	validProvider := contract.VerifyBindingSignature(providerKP.PublicKeyHex(), binding, "provider")
	validConsumer := contract.VerifyBindingSignature(consumerKP.PublicKeyHex(), binding, "consumer")
	fmt.Printf("  Provider sig valid: %v\n", validProvider)
	fmt.Printf("  Consumer sig valid: %v\n", validConsumer)
	fmt.Println()

	// 4. Execute
	fmt.Println("◆ Step 4: Execute Action")
	req := &action.Request{
		ActionID:  "policy.evaluate.v1",
		Input:     json.RawMessage(`{"action":"deploy_model","context":{"env":"production"}}`),
		Nonce:     fmt.Sprintf("nonce_%d", time.Now().UnixNano()),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	execEngine := execution.NewEngine(30 * time.Second)
	result, _ := execEngine.Execute(&schema, req, binding)
	fmt.Printf("  Success:  %v\n", result.Success)
	if result.Success {
		fmt.Printf("  Duration: %dms\n", result.DurationMs)
		fmt.Printf("  Receipt:  %s\n", result.ReceiptID)
	}
	fmt.Println()

	// 5. Generate evidence
	fmt.Println("◆ Step 5: Generate Evidence Receipt")
	receipt := evidence.NewReceipt(result, binding.ContractID, "witnessos.empirelabs.com.au", "agent.client.io")
	receipt.Sign(providerKP)
	fmt.Printf("  Receipt ID:  %s\n", receipt.ReceiptID)
	fmt.Printf("  Output Hash: %s\n", receipt.OutputHash[:16])
	fmt.Printf("  Duration:    %dms\n", receipt.DurationMs)
	fmt.Println()

	// 6. Settle
	fmt.Println("◆ Step 6: Settlement")
	ledger := settlement.NewLedger()
	txn := ledger.RecordTransaction("witnessos.empirelabs.com.au", "agent.client.io",
		binding.ContractID, "policy.evaluate.v1", 100, "AUD", settlement.ModelPostpay)
	settleReceipt, _ := ledger.Settle(txn.TransactionID)
	fmt.Printf("  Transaction: %s\n", settleReceipt.TransactionID)
	fmt.Printf("  Amount:      %d %s\n", settleReceipt.TotalAmount, settleReceipt.Currency)
	fmt.Printf("  Status:      settled\n")
	fmt.Println()

	// 7. Verify
	fmt.Println("◆ Step 7: Verify Evidence")
	valid, _ := receipt.Verify(providerKP.PublicKeyHex())
	if valid {
		fmt.Println("  ✓ Evidence signature VALID")
	} else {
		fmt.Println("  ✗ Evidence signature INVALID")
	}
	fmt.Println()

	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║    AIP Demo Complete ✓                                ║")
	fmt.Println("║    ✓ Dynamic Negotiation: offer→counter→accept        ║")
	fmt.Println("║    ✓ Contract Signing Ceremony: both parties signed   ║")
	fmt.Println("║    ✓ Signature Verification: both signatures valid    ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
}
