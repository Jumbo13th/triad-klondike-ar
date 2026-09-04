package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Jumbo13th/triad-klondike-ar/backend/internal/domain"
	"github.com/Jumbo13th/triad-klondike-ar/backend/internal/store"
)

const (
	operator = "00000000-0000-4000-8000-0000000000a1"
	player   = "00000000-0000-4000-8000-0000000000a2"
)

func newServer(t *testing.T, announce string, delay time.Duration) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	seed := filepath.Join(dir, "klondiked.json")
	os.WriteFile(seed, []byte(`{"operators":["`+operator+`"],"answer_timeout_s":5,"recheck_interval_s":15}`), 0o644)
	st, err := store.Open(filepath.Join(dir, "klondike.db"), seed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	srv := httptest.NewServer(New(domain.NewService(st), announce, delay))
	t.Cleanup(srv.Close)
	return srv
}

func call(t *testing.T, srv *httptest.Server, method, path string, body any, into any) int {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, srv.URL+path, &buf)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if into != nil && res.StatusCode == http.StatusOK {
		if err := json.NewDecoder(res.Body).Decode(into); err != nil {
			t.Fatal(err)
		}
	}
	return res.StatusCode
}

func TestHealthAnnouncesContractAndConfig(t *testing.T) {
	srv := newServer(t, domain.ContractVersion, 0)
	var h healthAnswer
	if code := call(t, srv, "GET", "/v1/health", nil, &h); code != 200 {
		t.Fatalf("status %d", code)
	}
	if h.Contract != "1" || h.ConfigRevision != 1 || h.AnswerTimeoutS != 5 || h.RecheckIntervalS != 15 || h.Time == "" {
		t.Fatalf("health: %+v", h)
	}
	other := newServer(t, "2", 0)
	call(t, other, "GET", "/v1/health", nil, &h)
	if h.Contract != "2" {
		t.Fatalf("announce override ignored: %+v", h)
	}
}

func TestDelayFlagDelaysCommandsOnly(t *testing.T) {
	srv := newServer(t, domain.ContractVersion, 150*time.Millisecond)
	start := time.Now()
	call(t, srv, "GET", "/v1/health", nil, nil)
	if time.Since(start) >= 150*time.Millisecond {
		t.Fatal("health answer was delayed")
	}
	start = time.Now()
	call(t, srv, "POST", "/v1/commands", map[string]any{"op_id": "x", "type": "wallet.compensate", "actor": "a", "payload": map[string]any{"amount": 1}}, nil)
	if time.Since(start) < 150*time.Millisecond {
		t.Fatal("command answer was not delayed")
	}
}

func TestConnectWalletCommandsAndAudit(t *testing.T) {
	srv := newServer(t, domain.ContractVersion, 0)

	var conn domain.ConnectResult
	if code := call(t, srv, "POST", "/v1/players/"+player+"/connect", map[string]string{"display_name": "Ivanov"}, &conn); code != 200 {
		t.Fatalf("connect status %d", code)
	}
	if conn.Player.Role != domain.RolePlayer || conn.Wallet.Revision != 1 || len(conn.Receipts) != 0 {
		t.Fatalf("connect: %+v", conn)
	}

	var wallet domain.Wallet
	call(t, srv, "GET", "/v1/players/"+player+"/wallet", nil, &wallet)
	if wallet.Total != 0 || wallet.Revision != 1 {
		t.Fatalf("wallet: %+v", wallet)
	}
	var unknown map[string]string
	call(t, srv, "GET", "/v1/players/nobody/wallet", nil, &unknown)
	if unknown["error"] != "player_unknown" {
		t.Fatalf("unknown wallet: %+v", unknown)
	}

	cmd := domain.Command{OpID: "op-1", Type: domain.TypeWalletCompensate, Actor: operator, Target: player, ExpectedRevision: 1,
		ConfigRevision: 1, Reason: "missing payout", Payload: json.RawMessage(`{"amount":200}`)}
	var answer domain.Answer
	if code := call(t, srv, "POST", "/v1/commands", cmd, &answer); code != 200 {
		t.Fatalf("command status %d", code)
	}
	if answer.Status != domain.StatusAccepted || answer.Wallet.Total != 200 || answer.Receipt.OpID != "op-1" {
		t.Fatalf("answer: %+v", answer)
	}
	call(t, srv, "POST", "/v1/commands", cmd, &answer)
	if answer.Status != domain.StatusAlreadyApplied {
		t.Fatalf("repeat: %+v", answer)
	}

	bad := domain.Command{OpID: "op-2", Type: "wallet.steal", Actor: operator}
	if code := call(t, srv, "POST", "/v1/commands", bad, nil); code != 400 {
		t.Fatalf("unknown type status %d, want 400", code)
	}
	req, _ := http.NewRequest("POST", srv.URL+"/v1/commands", bytes.NewBufferString("{not json"))
	res, _ := http.DefaultClient.Do(req)
	if res.StatusCode != 400 {
		t.Fatalf("malformed JSON status %d, want 400", res.StatusCode)
	}

	call(t, srv, "POST", "/v1/players/"+player+"/connect", map[string]string{"display_name": "Ivanov"}, &conn)
	if len(conn.Receipts) != 1 || conn.Receipts[0].TotalAfter != 200 {
		t.Fatalf("receipt on reconnect: %+v", conn)
	}

	var page auditAnswer
	call(t, srv, "GET", "/v1/audit?limit=2", nil, &page)
	if len(page.Entries) != 2 || page.Entries[0].ID <= page.Entries[1].ID {
		t.Fatalf("audit page: %+v", page)
	}
	var older auditAnswer
	call(t, srv, "GET", "/v1/audit?limit=50&before="+itoa(page.Entries[1].ID), nil, &older)
	if len(older.Entries) == 0 || older.Entries[0].ID >= page.Entries[1].ID {
		t.Fatalf("audit paging: %+v", older)
	}
	var all auditAnswer
	call(t, srv, "GET", "/v1/audit?limit=500", nil, &all)
	if len(all.Entries) > 50 {
		t.Fatalf("limit cap: %d", len(all.Entries))
	}
}

func itoa(v int64) string {
	b, _ := json.Marshal(v)
	return string(b)
}
