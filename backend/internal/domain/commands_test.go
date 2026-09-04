package domain

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/Jumbo13th/triad-klondike-ar/backend/internal/store"
)

const (
	operator = "00000000-0000-4000-8000-000000000001"
	player   = "00000000-0000-4000-8000-000000000002"
	stranger = "00000000-0000-4000-8000-000000000003"
)

func newService(t *testing.T) (*Service, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	seed := filepath.Join(dir, "klondiked.json")
	if err := os.WriteFile(seed, []byte(`{"operators":["`+operator+`"],"answer_timeout_s":5,"recheck_interval_s":15}`), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(dir, "klondike.db"), seed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	svc := NewService(st)
	if _, err := svc.Connect(player, "Ivanov"); err != nil {
		t.Fatal(err)
	}
	return svc, st
}

func compensate(opID string, amount, revision int64, reason string) Command {
	return Command{OpID: opID, Type: TypeWalletCompensate, Actor: operator, Target: player, ExpectedRevision: revision, ConfigRevision: 1,
		Reason: reason, Payload: json.RawMessage(fmt.Sprintf(`{"amount":%d}`, amount))}
}

func TestConnectIsIdempotentAndClaimsReceiptsOnce(t *testing.T) {
	svc, _ := newService(t)
	first, err := svc.Connect(player, "Ivanov")
	if err != nil {
		t.Fatal(err)
	}
	if first.Player.Role != RolePlayer || first.Wallet.Total != 0 || first.Wallet.Revision != 1 || len(first.Receipts) != 0 {
		t.Fatalf("unexpected connect result %+v", first)
	}
	if a, err := svc.Execute(compensate("op-1", 200, 1, "missing payout")); err != nil || a.Status != StatusAccepted {
		t.Fatalf("compensate: %v %+v", err, a)
	}
	second, err := svc.Connect(player, "Ivanov")
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Receipts) != 1 || second.Receipts[0].Amount != 200 || second.Wallet.Total != 200 {
		t.Fatalf("receipt not claimed: %+v", second)
	}
	third, _ := svc.Connect(player, "Ivanov")
	if len(third.Receipts) != 0 {
		t.Fatalf("receipt claimed twice: %+v", third)
	}
	op, _ := svc.Connect(operator, "Admin")
	if op.Player.Role != RoleOperator {
		t.Fatalf("seeded operator lost its role: %+v", op)
	}
}

func TestRepeatedOperationReplaysOrRefuses(t *testing.T) {
	svc, st := newService(t)
	first, err := svc.Execute(compensate("op-1", 100, 1, "test"))
	if err != nil || first.Status != StatusAccepted || first.Wallet.Total != 100 || first.Receipt == nil {
		t.Fatalf("first: %v %+v", err, first)
	}
	again, err := svc.Execute(compensate("op-1", 100, 1, "test"))
	if err != nil || again.Status != StatusAlreadyApplied || again.Wallet.Total != 100 || again.Receipt == nil || again.Receipt.OpID != "op-1" {
		t.Fatalf("repeat: %v %+v", err, again)
	}
	changed, err := svc.Execute(compensate("op-1", 999, 1, "test"))
	if err != nil || changed.Status != StatusRefused || changed.ReasonCode != ReasonOpIDPayloadMismatch {
		t.Fatalf("mismatch: %v %+v", err, changed)
	}
	// A refused command is recorded too, so its repeat is the same refusal.
	refused, _ := svc.Execute(compensate("op-2", -5000, 2, "overdraw"))
	if refused.Status != StatusRefused || refused.ReasonCode != ReasonInsufficientFunds || refused.Wallet.Total != 100 {
		t.Fatalf("refusal: %+v", refused)
	}
	replay, _ := svc.Execute(compensate("op-2", -5000, 2, "overdraw"))
	if replay.Status != StatusRefused || replay.ReasonCode != ReasonInsufficientFunds {
		t.Fatalf("refusal replay: %+v", replay)
	}
	st.View(func(tx *store.Tx) error {
		n, _ := tx.CountLedger(player)
		if n != 1 {
			t.Fatalf("ledger rows = %d, want 1", n)
		}
		rows, _ := tx.ListAudit(50, 0)
		var security int
		for _, a := range rows {
			if a.Outcome == OutcomeSecurity && a.ReasonCode == string(ReasonOpIDPayloadMismatch) {
				security++
			}
		}
		if security != 1 {
			t.Fatalf("security audit rows = %d, want 1", security)
		}
		return nil
	})
}

func TestCompensateRefusals(t *testing.T) {
	svc, _ := newService(t)
	cases := []struct {
		name string
		cmd  Command
		code Reason
	}{
		{"unauthorized", Command{OpID: "u", Type: TypeWalletCompensate, Actor: player, Target: player, ExpectedRevision: 1, Reason: "x", Payload: json.RawMessage(`{"amount":1}`)}, ReasonUnauthorized},
		{"stranger", Command{OpID: "s", Type: TypeWalletCompensate, Actor: stranger, Target: player, ExpectedRevision: 1, Reason: "x", Payload: json.RawMessage(`{"amount":1}`)}, ReasonUnauthorized},
		{"reason", compensate("r", 10, 1, ""), ReasonReasonRequired},
		{"zero", compensate("z", 0, 1, "x"), ReasonInvalidAmount},
		{"garbage", Command{OpID: "g", Type: TypeWalletCompensate, Actor: operator, Target: player, ExpectedRevision: 1, Reason: "x", Payload: json.RawMessage(`{"amount":"ten"}`)}, ReasonInvalidAmount},
		{"unknown", Command{OpID: "k", Type: TypeWalletCompensate, Actor: operator, Target: stranger, ExpectedRevision: 1, Reason: "x", Payload: json.RawMessage(`{"amount":1}`)}, ReasonPlayerUnknown},
		{"stale", compensate("st", 10, 7, "x"), ReasonStaleRevision},
		{"funds", compensate("f", -1, 1, "x"), ReasonInsufficientFunds},
	}
	for _, c := range cases {
		a, err := svc.Execute(c.cmd)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if a.Status != StatusRefused || a.ReasonCode != c.code {
			t.Fatalf("%s: got %+v, want refused %s", c.name, a, c.code)
		}
	}
	if a, _ := svc.Execute(compensate("st", 10, 1, "x")); a.Status != StatusRefused || a.ReasonCode != ReasonStaleRevision {
		t.Fatalf("stale refusal must replay, got %+v", a)
	}
	if w, _ := svc.Wallet(player); w.Total != 0 || w.Revision != 1 {
		t.Fatalf("refusals changed the wallet: %+v", w)
	}
}

func TestLedgerInvariantUnderRetryDrill(t *testing.T) {
	svc, st := newService(t)
	rng := rand.New(rand.NewSource(1))
	revision := int64(1)
	var expectedTotal int64
	applied := map[string]bool{}
	for i := 0; i < 100; i++ {
		opID := fmt.Sprintf("drill-%d", rng.Intn(60))
		amount := int64(rng.Intn(400) - 150)
		if amount == 0 {
			amount = 25
		}
		a, err := svc.Execute(compensate(opID, amount, revision, "drill"))
		if err != nil {
			t.Fatal(err)
		}
		switch a.Status {
		case StatusAccepted:
			if applied[opID] {
				t.Fatalf("%s applied twice", opID)
			}
			applied[opID] = true
			expectedTotal += amount
			revision = a.Revision
		case StatusAlreadyApplied:
			if !applied[opID] {
				t.Fatalf("%s reported already applied without applying", opID)
			}
		case StatusRefused:
			if a.ReasonCode == ReasonStaleRevision {
				revision = a.Revision
			}
		}
	}
	st.View(func(tx *store.Tx) error {
		w, _ := tx.GetWallet(player)
		sum, _ := tx.SumLedger(player)
		n, _ := tx.CountLedger(player)
		if w.Total != sum || w.Total != expectedTotal {
			t.Fatalf("total %d, ledger sum %d, expected %d", w.Total, sum, expectedTotal)
		}
		if int(n) != len(applied) {
			t.Fatalf("ledger rows %d, applied ops %d", n, len(applied))
		}
		return nil
	})
}

func TestConfigSet(t *testing.T) {
	svc, _ := newService(t)
	set := func(opID string, revision int64, payload string) Answer {
		a, err := svc.Execute(Command{OpID: opID, Type: TypeConfigSet, Actor: operator, Target: "config", ConfigRevision: revision, Reason: "tuning", Payload: json.RawMessage(payload)})
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	if a := set("c1", 1, `{"answer_timeout_s":2}`); a.Status != StatusAccepted || a.Revision != 2 {
		t.Fatalf("accept: %+v", a)
	}
	cfg, _ := svc.Health()
	if cfg.Revision != 2 || cfg.AnswerTimeoutS != 2 || cfg.RecheckIntervalS != 15 {
		t.Fatalf("config after set: %+v", cfg)
	}
	if a := set("c2", 1, `{"recheck_interval_s":5}`); a.Status != StatusRefused || a.ReasonCode != ReasonStaleRevision || a.Revision != 2 {
		t.Fatalf("stale: %+v", a)
	}
	if a := set("c3", 2, `{"answer_timeout_s":0}`); a.ReasonCode != ReasonInvalidConfigValue {
		t.Fatalf("range low: %+v", a)
	}
	if a := set("c4", 2, `{"recheck_interval_s":601}`); a.ReasonCode != ReasonInvalidConfigValue {
		t.Fatalf("range high: %+v", a)
	}
	if a := set("c5", 2, `{}`); a.ReasonCode != ReasonInvalidConfigValue {
		t.Fatalf("empty: %+v", a)
	}
	a, _ := svc.Execute(Command{OpID: "c6", Type: TypeConfigSet, Actor: player, Target: "config", ConfigRevision: 2, Reason: "x", Payload: json.RawMessage(`{"answer_timeout_s":3}`)})
	if a.ReasonCode != ReasonUnauthorized {
		t.Fatalf("player set config: %+v", a)
	}
}

func TestSecurityAndBadRequests(t *testing.T) {
	svc, st := newService(t)
	a, err := svc.Execute(Command{OpID: "sec-1", Type: TypeSecurity, Actor: player, Target: player, Payload: json.RawMessage(`{"kind":"unauthorized_cockpit_rpc","detail":"RpcAsk_Compensate"}`)})
	if err != nil || a.Status != StatusAccepted {
		t.Fatalf("security: %v %+v", err, a)
	}
	st.View(func(tx *store.Tx) error {
		rows, _ := tx.ListAudit(1, 0)
		if len(rows) != 1 || rows[0].Outcome != OutcomeSecurity || rows[0].Type != TypeSecurity {
			t.Fatalf("security audit: %+v", rows)
		}
		return nil
	})
	for _, cmd := range []Command{
		{Type: TypeWalletCompensate, Actor: operator},
		{OpID: "x", Actor: operator, Type: "wallet.steal"},
		{OpID: "x", Type: TypeWalletCompensate},
		{OpID: "x", Type: TypeWalletCompensate, Actor: operator, Payload: json.RawMessage(`{`)},
		{OpID: "x", Type: TypeSecurity, Actor: player, Payload: json.RawMessage(`{}`)},
	} {
		if _, err := svc.Execute(cmd); err == nil {
			t.Fatalf("expected bad request for %+v", cmd)
		}
	}
	if _, err := svc.Connect("", "nobody"); err == nil {
		t.Fatal("connect with empty uuid must fail")
	}
}

func TestPayloadHashIsCanonical(t *testing.T) {
	a, _ := PayloadHash(json.RawMessage(`{"amount": 1, "note": "x"}`))
	b, _ := PayloadHash(json.RawMessage(`{"note":"x","amount":1}`))
	c, _ := PayloadHash(json.RawMessage(`{"amount":2,"note":"x"}`))
	if a != b || a == c {
		t.Fatalf("hashes: %s %s %s", a, b, c)
	}
}

func TestLivePushedReceiptIsNotReplayedOnConnect(t *testing.T) {
	svc, _ := newService(t)

	online := compensate("op-online", 100, 1, "seen live")
	online.TargetOnline = true
	if answer, err := svc.Execute(online); err != nil || answer.Status != StatusAccepted {
		t.Fatalf("online credit: %+v, %v", answer, err)
	}
	if answer, err := svc.Execute(compensate("op-away", 50, 2, "while away")); err != nil || answer.Status != StatusAccepted {
		t.Fatalf("away credit: %+v, %v", answer, err)
	}

	res, err := svc.Connect(player, "Ivanov")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Receipts) != 1 || res.Receipts[0].OpID != "op-away" {
		t.Fatalf("connect should replay only the receipt created while away, got %+v", res.Receipts)
	}
}

func TestTargetOnlineAcceptsEngineNumbers(t *testing.T) {
	for raw, want := range map[string]bool{"true": true, "1": true, "false": false, "0": false} {
		var cmd Command
		if err := json.Unmarshal([]byte(`{"op_id":"x","target_online":`+raw+`}`), &cmd); err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
		if bool(cmd.TargetOnline) != want {
			t.Fatalf("%s: got %v", raw, cmd.TargetOnline)
		}
	}
}

func TestRepeatedOpIDWithAnotherTargetIsRefused(t *testing.T) {
	svc, _ := newService(t)
	if a, err := svc.Execute(compensate("op-id-bound", 100, 1, "first")); err != nil || a.Status != StatusAccepted {
		t.Fatalf("first: %+v %v", a, err)
	}
	other := compensate("op-id-bound", 100, 1, "first")
	other.Target = stranger
	a, err := svc.Execute(other)
	if err != nil || a.Status != StatusRefused || a.ReasonCode != ReasonOpIDPayloadMismatch {
		t.Fatalf("same op_id, other target: %+v %v", a, err)
	}
}

func TestMoneyStaysInTheGameRange(t *testing.T) {
	svc, _ := newService(t)
	if a, err := svc.Execute(compensate("op-range-1", 3000000000, 1, "too big")); err != nil || a.ReasonCode != ReasonInvalidAmount {
		t.Fatalf("amount beyond the range: %+v %v", a, err)
	}
	if a, err := svc.Execute(compensate("op-range-2", 2000000000, 1, "fits")); err != nil || a.Status != StatusAccepted {
		t.Fatalf("amount within the range: %+v %v", a, err)
	}
	a, err := svc.Execute(compensate("op-range-3", 2000000000, 2, "overflows"))
	if err != nil || a.ReasonCode != ReasonInvalidAmount || a.Wallet == nil || a.Wallet.Total != 2000000000 {
		t.Fatalf("total beyond the range: %+v %v", a, err)
	}
}
