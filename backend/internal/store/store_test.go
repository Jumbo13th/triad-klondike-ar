package store

import (
	"os"
	"path/filepath"
	"testing"
)

func openWithSeed(t *testing.T, dir, seed string) *Store {
	t.Helper()
	seedPath := filepath.Join(dir, "klondiked.json")
	if err := os.WriteFile(seedPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(filepath.Join(dir, "klondike.db"), seedPath)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func role(t *testing.T, s *Store, uuid string) string {
	t.Helper()
	var r string
	err := s.View(func(tx *Tx) error {
		p, err := tx.GetPlayer(uuid)
		if err != nil {
			return err
		}
		if p != nil {
			r = p.Role
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestOperatorListIsAppliedAtEveryStart(t *testing.T) {
	dir := t.TempDir()
	const a, b = "00000000-0000-4000-8000-00000000000a", "00000000-0000-4000-8000-00000000000b"

	s := openWithSeed(t, dir, `{ "operators": ["`+a+`"], "answer_timeout_s": 7, "recheck_interval_s": 20 }`)
	if got := role(t, s, a); got != "operator" {
		t.Fatalf("first start: %s role = %q, want operator", a, got)
	}
	if got := role(t, s, b); got != "" {
		t.Fatalf("first start: %s should not exist, role = %q", b, got)
	}
	s.Close()

	// A later edit of the seed promotes a known player and demotes a removed operator;
	// the runtime values stay at their first revision.
	s = openWithSeed(t, dir, `{ "operators": ["`+b+`"], "answer_timeout_s": 1, "recheck_interval_s": 1 }`)
	defer s.Close()
	if got := role(t, s, a); got != "player" {
		t.Fatalf("second start: %s role = %q, want player", a, got)
	}
	if got := role(t, s, b); got != "operator" {
		t.Fatalf("second start: %s role = %q, want operator", b, got)
	}
	var cfg Config
	if err := s.View(func(tx *Tx) error {
		c, err := tx.CurrentConfig()
		cfg = c
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if cfg.Revision != 1 || cfg.AnswerTimeoutS != 7 {
		t.Fatalf("config after second start = %+v, want revision 1 with the first seed's values", cfg)
	}
}
