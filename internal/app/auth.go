package app

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const (
	adminCookieName   = "webdown_admin_session"
	adminSessionTTL   = 24 * time.Hour
	adminCookieMaxAge = 86400
)

type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]time.Time
}

func newSessionStore() *sessionStore {
	return &sessionStore{sessions: map[string]time.Time{}}
}

func (ss *sessionStore) create() (string, error) {
	var tokenBytes [32]byte
	if _, err := rand.Read(tokenBytes[:]); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenBytes[:])
	expiresAt := time.Now().Add(adminSessionTTL)

	ss.mu.Lock()
	ss.sessions[token] = expiresAt
	ss.mu.Unlock()
	return token, nil
}

func (ss *sessionStore) valid(token string) bool {
	if token == "" {
		return false
	}

	now := time.Now()
	ss.mu.Lock()
	defer ss.mu.Unlock()

	expiresAt, ok := ss.sessions[token]
	if !ok || now.After(expiresAt) {
		delete(ss.sessions, token)
		return false
	}
	return true
}

func (ss *sessionStore) delete(token string) {
	ss.mu.Lock()
	delete(ss.sessions, token)
	ss.mu.Unlock()
}

func (s *Server) isAdmin(r *http.Request) bool {
	cookie, err := r.Cookie(adminCookieName)
	if err != nil {
		return false
	}
	return s.sessions.valid(cookie.Value)
}

func (s *Server) setAdminSession(w http.ResponseWriter) error {
	token, err := s.sessions.create()
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     adminCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   adminCookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (s *Server) clearAdminSession(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(adminCookieName); err == nil {
		s.sessions.delete(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     adminCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if s.isAdmin(r) {
		return true
	}
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
	return false
}
