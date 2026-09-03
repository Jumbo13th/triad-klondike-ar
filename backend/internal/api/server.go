// Package api exposes the domain over HTTP for the game server. Domain outcomes are
// always HTTP 200 with a status field; any other status code means the boundary
// did not answer and the game fails closed.
package api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Jumbo13th/triad-klondike-ar/backend/internal/domain"
)

// The engine's REST client refuses bodies of 1 MB and more; nothing here comes close.
const maxBody = 256 * 1024

type server struct {
	svc      *domain.Service
	announce string
	delay    time.Duration
}

// New builds the handler. announce is the contract version /v1/health reports and
// delay is added to every answer; both exist for the demo world.
func New(svc *domain.Service, announce string, delay time.Duration) http.Handler {
	s := &server{svc: svc, announce: announce, delay: delay}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/health", s.health)
	mux.HandleFunc("POST /v1/players/{uuid}/connect", s.connect)
	mux.HandleFunc("GET /v1/players/{uuid}/wallet", s.wallet)
	mux.HandleFunc("POST /v1/commands", s.commands)
	mux.HandleFunc("GET /v1/audit", s.audit)
	return s.wrap(mux)
}

func (s *server) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.delay > 0 {
			time.Sleep(s.delay)
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		next.ServeHTTP(w, r)
	})
}

type healthAnswer struct {
	Contract         string `json:"contract"`
	ConfigRevision   int64  `json:"config_revision"`
	AnswerTimeoutS   int64  `json:"answer_timeout_s"`
	RecheckIntervalS int64  `json:"recheck_interval_s"`
	Time             string `json:"time"`
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.svc.Health()
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, healthAnswer{Contract: s.announce, ConfigRevision: cfg.Revision, AnswerTimeoutS: cfg.AnswerTimeoutS,
		RecheckIntervalS: cfg.RecheckIntervalS, Time: time.Now().UTC().Format(time.RFC3339)})
}

type connectRequest struct {
	DisplayName string `json:"display_name"`
}

func (s *server) connect(w http.ResponseWriter, r *http.Request) {
	var req connectRequest
	if !decode(w, r, &req) {
		return
	}
	res, err := s.svc.Connect(r.PathValue("uuid"), req.DisplayName)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, res)
}

func (s *server) wallet(w http.ResponseWriter, r *http.Request) {
	wallet, err := s.svc.Wallet(r.PathValue("uuid"))
	if err != nil {
		fail(w, err)
		return
	}
	if wallet == nil {
		reply(w, map[string]string{"error": string(domain.ReasonPlayerUnknown)})
		return
	}
	reply(w, wallet)
}

func (s *server) commands(w http.ResponseWriter, r *http.Request) {
	var cmd domain.Command
	if !decode(w, r, &cmd) {
		return
	}
	answer, err := s.svc.Execute(cmd)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, answer)
}

type auditAnswer struct {
	Entries []domain.AuditRow `json:"entries"`
}

func (s *server) audit(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	before, _ := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
	entries, err := s.svc.Audit(limit, before)
	if err != nil {
		fail(w, err)
		return
	}
	if entries == nil {
		entries = []domain.AuditRow{}
	}
	reply(w, auditAnswer{Entries: entries})
}

func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
		return false
	}
	if len(body) == 0 {
		return true
	}
	if err := json.Unmarshal(body, into); err != nil {
		http.Error(w, "malformed JSON", http.StatusBadRequest)
		return false
	}
	return true
}

func reply(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("reply: %v", err)
	}
}

func fail(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrBadRequest) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("internal error: %v", err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}
