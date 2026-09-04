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
// delay is added to every command answer; both exist for the demo world. Health and
// reads stay instant: the game runs its requests one after another on one context,
// so a delayed health poll would push every queued request past the answer limit.
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
	log.Printf("connect %s %q role=%s receipts=%d", res.Player.UUID, req.DisplayName, res.Player.Role, len(res.Receipts))
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
	// After the read, not before: the flag simulates a backend that has the command
	// and answers late, so a caller that dies meanwhile still gets its row applied.
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	answer, err := s.svc.Execute(cmd)
	if err != nil {
		fail(w, err)
		return
	}
	outcome := string(answer.Status)
	if answer.ReasonCode != "" {
		outcome += " " + string(answer.ReasonCode)
	}
	log.Printf("command %s op=%s actor=%s target=%s -> %s revision=%d", cmd.Type, cmd.OpID, cmd.Actor, cmd.Target, outcome, answer.Revision)
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
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			log.Printf("%s %s: body too large", r.Method, r.URL.Path)
			failWith(w, http.StatusRequestEntityTooLarge, "body too large")
			return false
		}
		log.Printf("%s %s: body not received: %v", r.Method, r.URL.Path, err)
		failWith(w, http.StatusBadRequest, "body not received")
		return false
	}
	if len(body) == 0 {
		return true
	}
	if err := json.Unmarshal(body, into); err != nil {
		log.Printf("%s %s: malformed JSON: %v", r.Method, r.URL.Path, err)
		failWith(w, http.StatusBadRequest, "malformed JSON: "+err.Error())
		return false
	}
	return true
}

// The engine logs a failed request with the apiCode, uid and message fields of the
// error body; answering in that shape puts the reason into the game's own log line.
type errorBody struct {
	APICode int    `json:"apiCode"`
	UID     string `json:"uid"`
	Message string `json:"message"`
}

func failWith(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorBody{APICode: status, Message: message})
}

func reply(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("reply: %v", err)
	}
}

func fail(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrBadRequest) {
		log.Printf("bad request: %v", err)
		failWith(w, http.StatusBadRequest, err.Error())
		return
	}
	log.Printf("internal error: %v", err)
	failWith(w, http.StatusInternalServerError, "internal error")
}
