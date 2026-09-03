package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Command is the envelope of contracts/backend-http.md.
type Command struct {
	OpID             string          `json:"op_id"`
	Type             string          `json:"type"`
	Actor            string          `json:"actor"`
	Subject          string          `json:"subject"`
	Target           string          `json:"target"`
	ExpectedRevision int64           `json:"expected_revision"`
	ConfigRevision   int64           `json:"config_revision"`
	Reason           string          `json:"reason"`
	Payload          json.RawMessage `json:"payload"`
}

// Wallet is the current-player view of money.
type Wallet struct {
	Total    int64 `json:"total"`
	Reserved int64 `json:"reserved"`
	Revision int64 `json:"revision"`
}

// Receipt is keyed by the operation that produced it.
type Receipt struct {
	OpID       string `json:"op_id"`
	Kind       string `json:"kind"`
	Amount     int64  `json:"amount"`
	TotalAfter int64  `json:"total_after"`
	Reason     string `json:"reason"`
	CreatedAt  string `json:"created_at"`
}

// Answer is the one definitive reply to a command.
type Answer struct {
	Status     Status   `json:"status"`
	ReasonCode Reason   `json:"reason_code"`
	Revision   int64    `json:"revision"`
	Wallet     *Wallet  `json:"wallet,omitempty"`
	Receipt    *Receipt `json:"receipt,omitempty"`
}

// Player is the connect-time view of a record.
type Player struct {
	UUID      string `json:"uuid"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

// ConnectResult is the answer to /v1/players/{uuid}/connect.
type ConnectResult struct {
	Player   Player    `json:"player"`
	Wallet   Wallet    `json:"wallet"`
	Receipts []Receipt `json:"receipts"`
}

// Config is the newest runtime configuration row.
type Config struct {
	Revision         int64 `json:"config_revision"`
	AnswerTimeoutS   int64 `json:"answer_timeout_s"`
	RecheckIntervalS int64 `json:"recheck_interval_s"`
}

// AuditRow is one entry of /v1/audit.
type AuditRow struct {
	ID         int64  `json:"id"`
	CreatedAt  string `json:"created_at"`
	OpID       string `json:"op_id"`
	ActorUUID  string `json:"actor_uuid"`
	ActorRole  string `json:"actor_role"`
	Type       string `json:"type"`
	Target     string `json:"target"`
	Outcome    string `json:"outcome"`
	ReasonCode string `json:"reason_code"`
	Reason     string `json:"reason"`
	Amount     int64  `json:"amount"`
}

// PayloadHash binds an operation id to its payload. Keys are canonicalised so that
// the same JSON value with a different key order hashes identically.
func PayloadHash(raw json.RawMessage) (string, error) {
	var value any
	if len(raw) == 0 {
		raw = json.RawMessage("null")
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("payload: %w", err)
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}
