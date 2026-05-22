package auth

import (
	"context"
	"sync"
	"time"
)

// sessionTokenBytes is the entropy size of a session token.
const sessionTokenBytes = 32

// Session represents an authenticated login session.
type Session struct {
	Token   string
	Created time.Time
	Expires time.Time
}

// SessionStore is an in-memory store of active sessions. All methods are safe
// for concurrent use. Sessions do not survive a process restart.
type SessionStore struct {
	mu       sync.RWMutex
	ttl      time.Duration
	sessions map[string]Session
}

// NewSessionStore creates a store whose sessions live for ttl.
func NewSessionStore(ttl time.Duration) *SessionStore {
	return &SessionStore{ttl: ttl, sessions: make(map[string]Session)}
}

// Create issues a new session with a CSPRNG-generated token.
func (s *SessionStore) Create() (Session, error) {
	token, err := randomToken(sessionTokenBytes)
	if err != nil {
		return Session{}, err
	}
	now := time.Now()
	sess := Session{Token: token, Created: now, Expires: now.Add(s.ttl)}
	s.mu.Lock()
	s.sessions[token] = sess
	s.mu.Unlock()
	return sess, nil
}

// Validate reports whether token maps to a live (non-expired) session.
func (s *SessionStore) Validate(token string) bool {
	if token == "" {
		return false
	}
	s.mu.RLock()
	sess, ok := s.sessions[token]
	s.mu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().After(sess.Expires) {
		s.Delete(token)
		return false
	}
	return true
}

// Delete removes a session by token.
func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

// Count returns the number of stored sessions.
func (s *SessionStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

// TTL returns the configured session lifetime.
func (s *SessionStore) TTL() time.Duration { return s.ttl }

// StartGC runs a background loop that evicts expired sessions on the given
// interval until ctx is cancelled.
func (s *SessionStore) StartGC(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.evictExpired()
			}
		}
	}()
}

// evictExpired removes all sessions whose expiry has passed.
func (s *SessionStore) evictExpired() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for token, sess := range s.sessions {
		if now.After(sess.Expires) {
			delete(s.sessions, token)
		}
	}
}
