// Package store is the SQLite persistence of the backend. It knows rows and
// transactions, not rules; the domain package composes the rules on top of it.
package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Store owns one database connection; every access goes through a transaction so
// that a command's rows commit together or not at all.
type Store struct {
	db *sql.DB
}

// Seed is the content of klondiked.json, read at every start: the operator list is
// applied each time, the runtime values only when the database is new.
type Seed struct {
	Operators        []string `json:"operators"`
	AnswerTimeoutS   int64    `json:"answer_timeout_s"`
	RecheckIntervalS int64    `json:"recheck_interval_s"`
}

// Runtime configuration bounds (contracts/backend-http.md); the seed and
// config.set are both held to them, so a stored revision is always in effect.
const (
	MinAnswerTimeoutS   = 1
	MaxAnswerTimeoutS   = 120
	MinRecheckIntervalS = 1
	MaxRecheckIntervalS = 600
)

type Player struct {
	UUID          string
	Role          string
	DisplayName   string
	CreatedAt     string
	LastConnectAt string
}

type Wallet struct {
	PlayerUUID string
	Total      int64
	Reserved   int64
	Revision   int64
}

type Operation struct {
	OpID             string
	Type             string
	ActorUUID        string
	SubjectUUID      string
	Target           string
	PayloadHash      string
	ExpectedRevision int64
	ConfigRevision   int64
	Reason           string
	Status           string
	ReasonCode       string
	ResultJSON       string
	CreatedAt        string
}

type Ledger struct {
	PlayerUUID string
	OpID       string
	Kind       string
	Amount     int64
	TotalAfter int64
	ActorUUID  string
	Reason     string
	CreatedAt  string
}

type Receipt struct {
	OpID       string
	PlayerUUID string
	Kind       string
	Amount     int64
	TotalAfter int64
	Reason     string
	CreatedAt  string
}

type AuditEntry struct {
	ID               int64
	OpID             string
	ActorUUID        string
	ActorRole        string
	Type             string
	Target           string
	Reason           string
	ExpectedRevision int64
	ActualRevision   int64
	BeforeJSON       string
	AfterJSON        string
	Outcome          string
	ReasonCode       string
	Amount           int64
	CreatedAt        string
}

type Config struct {
	Revision         int64
	AnswerTimeoutS   int64
	RecheckIntervalS int64
	ChangedBy        string
	Reason           string
	CreatedAt        string
}

const schema = `
CREATE TABLE IF NOT EXISTS players (
	id INTEGER PRIMARY KEY,
	uuid TEXT NOT NULL UNIQUE,
	role TEXT NOT NULL,
	display_name TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	last_connect_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS wallets (
	id INTEGER PRIMARY KEY,
	player_uuid TEXT NOT NULL UNIQUE REFERENCES players(uuid),
	total INTEGER NOT NULL DEFAULT 0,
	reserved INTEGER NOT NULL DEFAULT 0,
	revision INTEGER NOT NULL DEFAULT 1,
	CHECK (reserved >= 0 AND reserved <= total)
);
CREATE TABLE IF NOT EXISTS operations (
	id INTEGER PRIMARY KEY,
	op_id TEXT NOT NULL UNIQUE,
	type TEXT NOT NULL,
	actor_uuid TEXT NOT NULL,
	subject_uuid TEXT NOT NULL,
	target TEXT NOT NULL,
	payload_hash TEXT NOT NULL,
	expected_revision INTEGER NOT NULL,
	config_revision INTEGER NOT NULL,
	reason TEXT NOT NULL,
	status TEXT NOT NULL,
	reason_code TEXT NOT NULL,
	result_json TEXT NOT NULL,
	created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS ledger (
	id INTEGER PRIMARY KEY,
	player_uuid TEXT NOT NULL,
	op_id TEXT NOT NULL REFERENCES operations(op_id) DEFERRABLE INITIALLY DEFERRED,
	kind TEXT NOT NULL,
	amount INTEGER NOT NULL,
	total_after INTEGER NOT NULL,
	actor_uuid TEXT NOT NULL,
	reason TEXT NOT NULL,
	created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS receipts (
	id INTEGER PRIMARY KEY,
	op_id TEXT NOT NULL UNIQUE,
	player_uuid TEXT NOT NULL,
	kind TEXT NOT NULL,
	amount INTEGER NOT NULL,
	total_after INTEGER NOT NULL,
	reason TEXT NOT NULL,
	created_at TEXT NOT NULL,
	delivered_at TEXT
);
CREATE TABLE IF NOT EXISTS audit (
	id INTEGER PRIMARY KEY,
	op_id TEXT NOT NULL,
	actor_uuid TEXT NOT NULL,
	actor_role TEXT NOT NULL,
	type TEXT NOT NULL,
	target TEXT NOT NULL,
	reason TEXT NOT NULL,
	expected_revision INTEGER NOT NULL,
	actual_revision INTEGER NOT NULL,
	before_json TEXT NOT NULL,
	after_json TEXT NOT NULL,
	outcome TEXT NOT NULL,
	reason_code TEXT NOT NULL,
	amount INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS config (
	revision INTEGER PRIMARY KEY,
	answer_timeout_s INTEGER NOT NULL,
	recheck_interval_s INTEGER NOT NULL,
	changed_by TEXT NOT NULL,
	reason TEXT NOT NULL,
	created_at TEXT NOT NULL
);
`

// Open creates or opens the database and applies the seed at seedPath: the operator
// list every time, the runtime values only on first run. A missing seed file means no
// operators and the default runtime values.
func Open(dbPath, seedPath string) (*Store, error) {
	// Pragmas travel in the DSN so that a connection the pool replaces after an
	// error enforces them too; foreign_keys and busy_timeout are per connection.
	path := filepath.ToSlash(dbPath)
	if filepath.IsAbs(dbPath) && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	dsn := "file:" + (&url.URL{Path: path}).EscapedPath() +
		"?_pragma=journal_mode(WAL)&_pragma=synchronous(FULL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	s := &Store{db: db}
	seed, err := readSeed(seedPath)
	if err != nil {
		db.Close()
		return nil, err
	}
	if err := s.seedConfigIfEmpty(seed); err != nil {
		db.Close()
		return nil, err
	}
	if err := s.applyOperators(seed); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func readSeed(seedPath string) (Seed, error) {
	seed := Seed{AnswerTimeoutS: 5, RecheckIntervalS: 15}
	data, err := os.ReadFile(seedPath)
	switch {
	case err == nil:
		if err := json.Unmarshal(data, &seed); err != nil {
			return seed, fmt.Errorf("%s: %w", seedPath, err)
		}
	case errors.Is(err, os.ErrNotExist):
		log.Printf("no seed file at %s: starting with no operators", seedPath)
	default:
		return seed, err
	}
	if seed.AnswerTimeoutS < MinAnswerTimeoutS || seed.AnswerTimeoutS > MaxAnswerTimeoutS {
		return seed, fmt.Errorf("%s: answer_timeout_s %d is outside %d..%d", seedPath, seed.AnswerTimeoutS, MinAnswerTimeoutS, MaxAnswerTimeoutS)
	}
	if seed.RecheckIntervalS < MinRecheckIntervalS || seed.RecheckIntervalS > MaxRecheckIntervalS {
		return seed, fmt.Errorf("%s: recheck_interval_s %d is outside %d..%d", seedPath, seed.RecheckIntervalS, MinRecheckIntervalS, MaxRecheckIntervalS)
	}
	return seed, nil
}

// The runtime values are seeded once: after that they change only through
// config.set, which is what gives them a revision.
func (s *Store) seedConfigIfEmpty(seed Seed) error {
	return s.Update(func(t *Tx) error {
		var count int
		if err := t.tx.QueryRow("SELECT COUNT(*) FROM config").Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		return t.InsertConfig(Config{Revision: 1, AnswerTimeoutS: seed.AnswerTimeoutS, RecheckIntervalS: seed.RecheckIntervalS, ChangedBy: "seed", Reason: "seed", CreatedAt: Now()})
	})
}

// The operator list is an allowlist, so every start makes the roles match it:
// listed identities are operators (created if unknown), everyone else a player.
func (s *Store) applyOperators(seed Seed) error {
	return s.Update(func(t *Tx) error {
		now := Now()
		if _, err := t.tx.Exec("UPDATE players SET role = 'player' WHERE role = 'operator'"); err != nil {
			return err
		}
		for _, uuid := range seed.Operators {
			if _, err := t.tx.Exec("INSERT OR IGNORE INTO players (uuid, role, created_at) VALUES (?, 'operator', ?)", uuid, now); err != nil {
				return err
			}
			if _, err := t.tx.Exec("INSERT OR IGNORE INTO wallets (player_uuid) VALUES (?)", uuid); err != nil {
				return err
			}
			if _, err := t.tx.Exec("UPDATE players SET role = 'operator' WHERE uuid = ?", uuid); err != nil {
				return err
			}
		}
		log.Printf("operators from seed: %d", len(seed.Operators))
		return nil
	})
}

// Now is the single clock of the backend: UTC, RFC 3339.
func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// Tx is one transaction; the store hands out no other access.
type Tx struct {
	tx *sql.Tx
}

// Update runs fn in a transaction and commits when it returns nil.
func (s *Store) Update(fn func(*Tx) error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if err := fn(&Tx{tx: tx}); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// View runs fn in a transaction that is always rolled back.
func (s *Store) View(fn func(*Tx) error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	return fn(&Tx{tx: tx})
}

func (t *Tx) GetPlayer(uuid string) (*Player, error) {
	p := &Player{}
	err := t.tx.QueryRow("SELECT uuid, role, display_name, created_at, last_connect_at FROM players WHERE uuid = ?", uuid).
		Scan(&p.UUID, &p.Role, &p.DisplayName, &p.CreatedAt, &p.LastConnectAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

// ConnectPlayer creates the record with the player role when absent and stamps the
// connection either way.
func (t *Tx) ConnectPlayer(uuid, displayName, now string) (Player, error) {
	p, err := t.GetPlayer(uuid)
	if err != nil {
		return Player{}, err
	}
	if p == nil {
		if _, err := t.tx.Exec("INSERT INTO players (uuid, role, display_name, created_at, last_connect_at) VALUES (?, 'player', ?, ?, ?)", uuid, displayName, now, now); err != nil {
			return Player{}, err
		}
		return Player{UUID: uuid, Role: "player", DisplayName: displayName, CreatedAt: now, LastConnectAt: now}, nil
	}
	if _, err := t.tx.Exec("UPDATE players SET display_name = ?, last_connect_at = ? WHERE uuid = ?", displayName, now, uuid); err != nil {
		return Player{}, err
	}
	p.DisplayName = displayName
	p.LastConnectAt = now
	return *p, nil
}

func (t *Tx) GetWallet(uuid string) (*Wallet, error) {
	w := &Wallet{}
	err := t.tx.QueryRow("SELECT player_uuid, total, reserved, revision FROM wallets WHERE player_uuid = ?", uuid).
		Scan(&w.PlayerUUID, &w.Total, &w.Reserved, &w.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return w, err
}

func (t *Tx) EnsureWallet(uuid string) (Wallet, error) {
	w, err := t.GetWallet(uuid)
	if err != nil {
		return Wallet{}, err
	}
	if w != nil {
		return *w, nil
	}
	if _, err := t.tx.Exec("INSERT INTO wallets (player_uuid) VALUES (?)", uuid); err != nil {
		return Wallet{}, err
	}
	return Wallet{PlayerUUID: uuid, Revision: 1}, nil
}

func (t *Tx) UpdateWallet(w Wallet) error {
	_, err := t.tx.Exec("UPDATE wallets SET total = ?, reserved = ?, revision = ? WHERE player_uuid = ?", w.Total, w.Reserved, w.Revision, w.PlayerUUID)
	return err
}

func (t *Tx) GetOperation(opID string) (*Operation, error) {
	o := &Operation{}
	err := t.tx.QueryRow(`SELECT op_id, type, actor_uuid, subject_uuid, target, payload_hash, expected_revision, config_revision,
		reason, status, reason_code, result_json, created_at FROM operations WHERE op_id = ?`, opID).
		Scan(&o.OpID, &o.Type, &o.ActorUUID, &o.SubjectUUID, &o.Target, &o.PayloadHash, &o.ExpectedRevision, &o.ConfigRevision,
			&o.Reason, &o.Status, &o.ReasonCode, &o.ResultJSON, &o.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return o, err
}

func (t *Tx) InsertOperation(o Operation) error {
	_, err := t.tx.Exec(`INSERT INTO operations (op_id, type, actor_uuid, subject_uuid, target, payload_hash, expected_revision,
		config_revision, reason, status, reason_code, result_json, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		o.OpID, o.Type, o.ActorUUID, o.SubjectUUID, o.Target, o.PayloadHash, o.ExpectedRevision,
		o.ConfigRevision, o.Reason, o.Status, o.ReasonCode, o.ResultJSON, o.CreatedAt)
	return err
}

func (t *Tx) InsertLedger(l Ledger) error {
	_, err := t.tx.Exec(`INSERT INTO ledger (player_uuid, op_id, kind, amount, total_after, actor_uuid, reason, created_at)
		VALUES (?,?,?,?,?,?,?,?)`, l.PlayerUUID, l.OpID, l.Kind, l.Amount, l.TotalAfter, l.ActorUUID, l.Reason, l.CreatedAt)
	return err
}

// SumLedger backs the invariant total == sum(ledger).
func (t *Tx) SumLedger(uuid string) (int64, error) {
	var sum int64
	err := t.tx.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM ledger WHERE player_uuid = ?", uuid).Scan(&sum)
	return sum, err
}

func (t *Tx) CountLedger(uuid string) (int64, error) {
	var n int64
	err := t.tx.QueryRow("SELECT COUNT(*) FROM ledger WHERE player_uuid = ?", uuid).Scan(&n)
	return n, err
}

func (t *Tx) InsertReceipt(r Receipt) error {
	_, err := t.tx.Exec(`INSERT INTO receipts (op_id, player_uuid, kind, amount, total_after, reason, created_at)
		VALUES (?,?,?,?,?,?,?)`, r.OpID, r.PlayerUUID, r.Kind, r.Amount, r.TotalAfter, r.Reason, r.CreatedAt)
	return err
}

// ClaimReceipts returns the player's undelivered receipts and marks them delivered.
func (t *Tx) ClaimReceipts(uuid, now string) ([]Receipt, error) {
	rows, err := t.tx.Query(`SELECT op_id, player_uuid, kind, amount, total_after, reason, created_at FROM receipts
		WHERE player_uuid = ? AND delivered_at IS NULL ORDER BY id`, uuid)
	if err != nil {
		return nil, err
	}
	var out []Receipt
	for rows.Next() {
		var r Receipt
		if err := rows.Scan(&r.OpID, &r.PlayerUUID, &r.Kind, &r.Amount, &r.TotalAfter, &r.Reason, &r.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) > 0 {
		if _, err := t.tx.Exec("UPDATE receipts SET delivered_at = ? WHERE player_uuid = ? AND delivered_at IS NULL", now, uuid); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (t *Tx) MarkReceiptDelivered(opID, now string) error {
	_, err := t.tx.Exec("UPDATE receipts SET delivered_at = ? WHERE op_id = ? AND delivered_at IS NULL", now, opID)
	return err
}

func (t *Tx) InsertAudit(a AuditEntry) error {
	_, err := t.tx.Exec(`INSERT INTO audit (op_id, actor_uuid, actor_role, type, target, reason, expected_revision, actual_revision,
		before_json, after_json, outcome, reason_code, amount, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.OpID, a.ActorUUID, a.ActorRole, a.Type, a.Target, a.Reason, a.ExpectedRevision, a.ActualRevision,
		a.BeforeJSON, a.AfterJSON, a.Outcome, a.ReasonCode, a.Amount, a.CreatedAt)
	return err
}

// ListAudit returns the newest entries first; before = 0 starts from the newest.
func (t *Tx) ListAudit(limit int, before int64) ([]AuditEntry, error) {
	if before <= 0 {
		before = 1<<62 - 1
	}
	rows, err := t.tx.Query(`SELECT id, op_id, actor_uuid, actor_role, type, target, reason, expected_revision, actual_revision,
		before_json, after_json, outcome, reason_code, amount, created_at FROM audit WHERE id < ? ORDER BY id DESC LIMIT ?`, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuditEntry
	for rows.Next() {
		var a AuditEntry
		if err := rows.Scan(&a.ID, &a.OpID, &a.ActorUUID, &a.ActorRole, &a.Type, &a.Target, &a.Reason, &a.ExpectedRevision, &a.ActualRevision,
			&a.BeforeJSON, &a.AfterJSON, &a.Outcome, &a.ReasonCode, &a.Amount, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (t *Tx) CountAudit() (int64, error) {
	var n int64
	err := t.tx.QueryRow("SELECT COUNT(*) FROM audit").Scan(&n)
	return n, err
}

func (t *Tx) CurrentConfig() (Config, error) {
	c := Config{}
	err := t.tx.QueryRow(`SELECT revision, answer_timeout_s, recheck_interval_s, changed_by, reason, created_at FROM config
		ORDER BY revision DESC LIMIT 1`).Scan(&c.Revision, &c.AnswerTimeoutS, &c.RecheckIntervalS, &c.ChangedBy, &c.Reason, &c.CreatedAt)
	return c, err
}

func (t *Tx) InsertConfig(c Config) error {
	_, err := t.tx.Exec(`INSERT INTO config (revision, answer_timeout_s, recheck_interval_s, changed_by, reason, created_at)
		VALUES (?,?,?,?,?,?)`, c.Revision, c.AnswerTimeoutS, c.RecheckIntervalS, c.ChangedBy, c.Reason, c.CreatedAt)
	return err
}
