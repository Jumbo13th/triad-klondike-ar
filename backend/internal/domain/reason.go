// Package domain holds the commands the game may send, the rules that accept or
// refuse them, and the stable reason codes the game shows to players.
package domain

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

// Runtime configuration bounds (contracts/backend-http.md).
const (
	MinAnswerTimeoutS   = 1
	MaxAnswerTimeoutS   = 120
	MinRecheckIntervalS = 1
	MaxRecheckIntervalS = 600
)
