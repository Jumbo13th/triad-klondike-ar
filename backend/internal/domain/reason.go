// Package domain holds the commands the game may send, the rules that accept or
// refuse them, and the stable reason codes the game shows to players.
package domain

import "github.com/Jumbo13th/triad-klondike-ar/backend/internal/store"

// ContractVersion is announced by /v1/health; the game refuses any other value.
const ContractVersion = "1"

// Status is the domain outcome of a command; transport failures never produce one.
type Status string

const (
	StatusAccepted       Status = "accepted"
	StatusRefused        Status = "refused"
	StatusAlreadyApplied Status = "already_applied"
)

// Reason is a stable refusal code; the client translates it as TK-Reason_<code>.
type Reason string

const (
	ReasonUnauthorized        Reason = "unauthorized"
	ReasonPlayerUnknown       Reason = "player_unknown"
	ReasonInvalidAmount       Reason = "invalid_amount"
	ReasonInsufficientFunds   Reason = "insufficient_funds"
	ReasonStaleRevision       Reason = "stale_revision"
	ReasonReasonRequired      Reason = "reason_required"
	ReasonOpIDPayloadMismatch Reason = "op_id_payload_mismatch"
	ReasonInvalidConfigValue  Reason = "invalid_config_value"
)

// Command types the backend knows; anything else is a contract violation (HTTP 400).
const (
	TypeWalletCompensate = "wallet.compensate"
	TypeConfigSet        = "config.set"
	TypeSecurity         = "security"
)

// Roles a player record can carry.
const (
	RolePlayer   = "player"
	RoleOperator = "operator"
)

// Audit outcomes.
const (
	OutcomeAccepted = "accepted"
	OutcomeRefused  = "refused"
	OutcomeSecurity = "security"
)

// Runtime configuration bounds live with the seed that must respect them too.
const (
	MinAnswerTimeoutS   = store.MinAnswerTimeoutS
	MaxAnswerTimeoutS   = store.MaxAnswerTimeoutS
	MinRecheckIntervalS = store.MinRecheckIntervalS
	MaxRecheckIntervalS = store.MaxRecheckIntervalS
)

// Money range shared with the game, whose int is 32-bit: an amount or a total
// outside it could not be shown or carried in an RPC (contracts/backend-http.md).
const (
	MinMoney = -2147483648
	MaxMoney = 2147483647
)

// InMoneyRange reports whether the game can represent v.
func InMoneyRange(v int64) bool {
	return v >= MinMoney && v <= MaxMoney
}
