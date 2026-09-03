package domain

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Jumbo13th/triad-klondike-ar/backend/internal/store"
)

// ErrBadRequest marks a contract violation: the game sent something the backend
// cannot interpret. It becomes HTTP 400, which the game treats as "no answer".
var ErrBadRequest = errors.New("bad request")

// Service applies the rules of the boundary on top of the store.
type Service struct {
	st  *store.Store
	now func() string
}

func NewService(st *store.Store) *Service {
	return &Service{st: st, now: store.Now}
}

// Health returns the newest runtime configuration.
func (s *Service) Health() (Config, error) {
	var c Config
	err := s.st.View(func(t *store.Tx) error {
		row, err := t.CurrentConfig()
		if err != nil {
			return err
		}
		c = Config{Revision: row.Revision, AnswerTimeoutS: row.AnswerTimeoutS, RecheckIntervalS: row.RecheckIntervalS}
		return nil
	})
	return c, err
}

// Connect creates the player record and wallet when absent, claims undelivered
// receipts, and records the connection; one transaction, idempotent.
func (s *Service) Connect(uuid, displayName string) (ConnectResult, error) {
	if uuid == "" {
		return ConnectResult{}, fmt.Errorf("%w: empty uuid", ErrBadRequest)
	}
	var out ConnectResult
	err := s.st.Update(func(t *store.Tx) error {
		now := s.now()
		p, err := t.ConnectPlayer(uuid, displayName, now)
		if err != nil {
			return err
		}
		w, err := t.EnsureWallet(uuid)
		if err != nil {
			return err
		}
		receipts, err := t.ClaimReceipts(uuid, now)
		if err != nil {
			return err
		}
		out = ConnectResult{
			Player:   Player{UUID: p.UUID, Role: p.Role, CreatedAt: p.CreatedAt},
			Wallet:   Wallet{Total: w.Total, Reserved: w.Reserved, Revision: w.Revision},
			Receipts: make([]Receipt, 0, len(receipts)),
		}
		for _, r := range receipts {
			out.Receipts = append(out.Receipts, Receipt{OpID: r.OpID, Kind: r.Kind, Amount: r.Amount, TotalAfter: r.TotalAfter, Reason: r.Reason, CreatedAt: r.CreatedAt})
		}
		return t.InsertAudit(store.AuditEntry{ActorUUID: uuid, ActorRole: p.Role, Type: "connect", Target: uuid, ActualRevision: w.Revision, Outcome: OutcomeAccepted, CreatedAt: now})
	})
	return out, err
}

// Wallet returns nil when the player has no record.
func (s *Service) Wallet(uuid string) (*Wallet, error) {
	var out *Wallet
	err := s.st.View(func(t *store.Tx) error {
		w, err := t.GetWallet(uuid)
		if err != nil || w == nil {
			return err
		}
		out = &Wallet{Total: w.Total, Reserved: w.Reserved, Revision: w.Revision}
		return nil
	})
	return out, err
}

// Audit returns up to limit entries, newest first, older than before (0 = newest).
func (s *Service) Audit(limit int, before int64) ([]AuditRow, error) {
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	var out []AuditRow
	err := s.st.View(func(t *store.Tx) error {
		entries, err := t.ListAudit(limit, before)
		if err != nil {
			return err
		}
		out = make([]AuditRow, 0, len(entries))
		for _, a := range entries {
			out = append(out, AuditRow{ID: a.ID, CreatedAt: a.CreatedAt, OpID: a.OpID, ActorUUID: a.ActorUUID, ActorRole: a.ActorRole,
				Type: a.Type, Target: a.Target, Outcome: a.Outcome, ReasonCode: a.ReasonCode, Reason: a.Reason, Amount: a.Amount})
		}
		return nil
	})
	return out, err
}

type compensatePayload struct {
	Amount int64 `json:"amount"`
}

type configPayload struct {
	AnswerTimeoutS   *int64 `json:"answer_timeout_s"`
	RecheckIntervalS *int64 `json:"recheck_interval_s"`
}

type securityPayload struct {
	Kind   string `json:"kind"`
	Detail string `json:"detail"`
}

// Execute gives a command its one definitive answer. A repeated operation id with
// the same payload replays the recorded answer; with a different payload it is
// refused and recorded as a security event.
func (s *Service) Execute(cmd Command) (Answer, error) {
	switch {
	case cmd.OpID == "":
		return Answer{}, fmt.Errorf("%w: empty op_id", ErrBadRequest)
	case cmd.Actor == "":
		return Answer{}, fmt.Errorf("%w: empty actor", ErrBadRequest)
	case cmd.Type != TypeWalletCompensate && cmd.Type != TypeConfigSet && cmd.Type != TypeSecurity:
		return Answer{}, fmt.Errorf("%w: unknown type %q", ErrBadRequest, cmd.Type)
	}
	if cmd.Subject == "" {
		cmd.Subject = cmd.Actor
	}
	hash, err := PayloadHash(cmd.Payload)
	if err != nil {
		return Answer{}, fmt.Errorf("%w: %v", ErrBadRequest, err)
	}

	var answer Answer
	err = s.st.Update(func(t *store.Tx) error {
		now := s.now()
		actor, err := t.GetPlayer(cmd.Actor)
		if err != nil {
			return err
		}
		actorRole := RolePlayer
		if actor != nil {
			actorRole = actor.Role
		}
		ctx := &execution{tx: t, cmd: cmd, hash: hash, now: now, actorRole: actorRole}

		prior, err := t.GetOperation(cmd.OpID)
		if err != nil {
			return err
		}
		if prior != nil {
			if prior.PayloadHash != hash {
				answer = Answer{Status: StatusRefused, ReasonCode: ReasonOpIDPayloadMismatch}
				return ctx.audit(OutcomeSecurity, ReasonOpIDPayloadMismatch, "", "", 0, 0)
			}
			if err := json.Unmarshal([]byte(prior.ResultJSON), &answer); err != nil {
				return err
			}
			if prior.Status == string(StatusAccepted) {
				answer.Status = StatusAlreadyApplied
			}
			return nil
		}

		switch cmd.Type {
		case TypeWalletCompensate:
			answer, err = ctx.compensate()
		case TypeConfigSet:
			answer, err = ctx.setConfig()
		case TypeSecurity:
			answer, err = ctx.security()
		}
		return err
	})
	return answer, err
}

// execution carries one command through its transaction.
type execution struct {
	tx        *store.Tx
	cmd       Command
	hash      string
	now       string
	actorRole string
}

func (e *execution) record(answer Answer, before, after string, actualRevision, amount int64) (Answer, error) {
	result, err := json.Marshal(answer)
	if err != nil {
		return answer, err
	}
	status := string(answer.Status)
	outcome := OutcomeAccepted
	if answer.Status == StatusRefused {
		outcome = OutcomeRefused
	}
	if e.cmd.Type == TypeSecurity {
		outcome = OutcomeSecurity
	}
	if err := e.tx.InsertOperation(store.Operation{
		OpID: e.cmd.OpID, Type: e.cmd.Type, ActorUUID: e.cmd.Actor, SubjectUUID: e.cmd.Subject, Target: e.cmd.Target,
		PayloadHash: e.hash, ExpectedRevision: e.cmd.ExpectedRevision, ConfigRevision: e.cmd.ConfigRevision, Reason: e.cmd.Reason,
		Status: status, ReasonCode: string(answer.ReasonCode), ResultJSON: string(result), CreatedAt: e.now,
	}); err != nil {
		return answer, err
	}
	return answer, e.audit(outcome, answer.ReasonCode, before, after, actualRevision, amount)
}

func (e *execution) audit(outcome string, code Reason, before, after string, actualRevision, amount int64) error {
	return e.tx.InsertAudit(store.AuditEntry{
		OpID: e.cmd.OpID, ActorUUID: e.cmd.Actor, ActorRole: e.actorRole, Type: e.cmd.Type, Target: e.cmd.Target, Reason: e.cmd.Reason,
		ExpectedRevision: e.cmd.ExpectedRevision, ActualRevision: actualRevision, BeforeJSON: before, AfterJSON: after,
		Outcome: outcome, ReasonCode: string(code), Amount: amount, CreatedAt: e.now,
	})
}

func (e *execution) refuse(code Reason, w *store.Wallet) (Answer, error) {
	answer := Answer{Status: StatusRefused, ReasonCode: code}
	var revision int64
	if w != nil {
		answer.Wallet = &Wallet{Total: w.Total, Reserved: w.Reserved, Revision: w.Revision}
		answer.Revision = w.Revision
		revision = w.Revision
	}
	return e.record(answer, "", "", revision, 0)
}

func (e *execution) compensate() (Answer, error) {
	if e.actorRole != RoleOperator {
		return e.refuse(ReasonUnauthorized, nil)
	}
	if e.cmd.Reason == "" {
		return e.refuse(ReasonReasonRequired, nil)
	}
	var p compensatePayload
	if err := json.Unmarshal(e.cmd.Payload, &p); err != nil || p.Amount == 0 {
		return e.refuse(ReasonInvalidAmount, nil)
	}
	w, err := e.tx.GetWallet(e.cmd.Target)
	if err != nil {
		return Answer{}, err
	}
	if w == nil {
		return e.refuse(ReasonPlayerUnknown, nil)
	}
	if e.cmd.ExpectedRevision != w.Revision {
		return e.refuse(ReasonStaleRevision, w)
	}
	if p.Amount < 0 && w.Total-w.Reserved < -p.Amount {
		return e.refuse(ReasonInsufficientFunds, w)
	}
	before, _ := json.Marshal(Wallet{Total: w.Total, Reserved: w.Reserved, Revision: w.Revision})
	w.Total += p.Amount
	w.Revision++
	if err := e.tx.UpdateWallet(*w); err != nil {
		return Answer{}, err
	}
	if err := e.tx.InsertLedger(store.Ledger{PlayerUUID: w.PlayerUUID, OpID: e.cmd.OpID, Kind: "compensation", Amount: p.Amount,
		TotalAfter: w.Total, ActorUUID: e.cmd.Actor, Reason: e.cmd.Reason, CreatedAt: e.now}); err != nil {
		return Answer{}, err
	}
	receipt := Receipt{OpID: e.cmd.OpID, Kind: "compensation", Amount: p.Amount, TotalAfter: w.Total, Reason: e.cmd.Reason, CreatedAt: e.now}
	if err := e.tx.InsertReceipt(store.Receipt{OpID: receipt.OpID, PlayerUUID: w.PlayerUUID, Kind: receipt.Kind, Amount: receipt.Amount,
		TotalAfter: receipt.TotalAfter, Reason: receipt.Reason, CreatedAt: e.now}); err != nil {
		return Answer{}, err
	}
	wallet := Wallet{Total: w.Total, Reserved: w.Reserved, Revision: w.Revision}
	after, _ := json.Marshal(wallet)
	answer := Answer{Status: StatusAccepted, Revision: w.Revision, Wallet: &wallet, Receipt: &receipt}
	return e.record(answer, string(before), string(after), w.Revision, p.Amount)
}

func (e *execution) setConfig() (Answer, error) {
	if e.actorRole != RoleOperator {
		return e.refuse(ReasonUnauthorized, nil)
	}
	if e.cmd.Reason == "" {
		return e.refuse(ReasonReasonRequired, nil)
	}
	current, err := e.tx.CurrentConfig()
	if err != nil {
		return Answer{}, err
	}
	if e.cmd.ConfigRevision != current.Revision {
		answer := Answer{Status: StatusRefused, ReasonCode: ReasonStaleRevision, Revision: current.Revision}
		return e.record(answer, "", "", current.Revision, 0)
	}
	var p configPayload
	if err := json.Unmarshal(e.cmd.Payload, &p); err != nil || (p.AnswerTimeoutS == nil && p.RecheckIntervalS == nil) {
		return e.record(Answer{Status: StatusRefused, ReasonCode: ReasonInvalidConfigValue, Revision: current.Revision}, "", "", current.Revision, 0)
	}
	next := current
	if p.AnswerTimeoutS != nil {
		next.AnswerTimeoutS = *p.AnswerTimeoutS
	}
	if p.RecheckIntervalS != nil {
		next.RecheckIntervalS = *p.RecheckIntervalS
	}
	if next.AnswerTimeoutS < MinAnswerTimeoutS || next.AnswerTimeoutS > MaxAnswerTimeoutS ||
		next.RecheckIntervalS < MinRecheckIntervalS || next.RecheckIntervalS > MaxRecheckIntervalS {
		return e.record(Answer{Status: StatusRefused, ReasonCode: ReasonInvalidConfigValue, Revision: current.Revision}, "", "", current.Revision, 0)
	}
	next.Revision = current.Revision + 1
	next.ChangedBy = e.cmd.Actor
	next.Reason = e.cmd.Reason
	next.CreatedAt = e.now
	if err := e.tx.InsertConfig(next); err != nil {
		return Answer{}, err
	}
	before, _ := json.Marshal(Config{Revision: current.Revision, AnswerTimeoutS: current.AnswerTimeoutS, RecheckIntervalS: current.RecheckIntervalS})
	after, _ := json.Marshal(Config{Revision: next.Revision, AnswerTimeoutS: next.AnswerTimeoutS, RecheckIntervalS: next.RecheckIntervalS})
	return e.record(Answer{Status: StatusAccepted, Revision: next.Revision}, string(before), string(after), next.Revision, 0)
}

func (e *execution) security() (Answer, error) {
	var p securityPayload
	if err := json.Unmarshal(e.cmd.Payload, &p); err != nil || p.Kind == "" {
		return Answer{}, fmt.Errorf("%w: security payload", ErrBadRequest)
	}
	return e.record(Answer{Status: StatusAccepted}, "", p.Kind+": "+p.Detail, 0, 0)
}
