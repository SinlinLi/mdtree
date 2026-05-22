package auth

import (
	"net/http"
	"time"
)

// CookieName is the name of the mdtree session cookie.
const CookieName = "mdtree_session"

// Authenticator validates session cookies and guards HTTP handlers.
type Authenticator struct {
	store  *SessionStore
	secure bool
}

// NewAuthenticator builds an Authenticator backed by store. When secure is
// true the session cookie is only transmitted over HTTPS.
func NewAuthenticator(store *SessionStore, secure bool) *Authenticator {
	return &Authenticator{store: store, secure: secure}
}

// SetCookie writes the session cookie for sess to the response.
func (a *Authenticator) SetCookie(w http.ResponseWriter, sess Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    sess.Token,
		Path:     "/",
		Expires:  sess.Expires,
		MaxAge:   int(time.Until(sess.Expires).Seconds()),
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// ClearCookie expires the session cookie on the client.
func (a *Authenticator) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// Token extracts the session token from the request cookie.
func (a *Authenticator) Token(r *http.Request) string {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

// Authenticated reports whether the request carries a valid session.
func (a *Authenticator) Authenticated(r *http.Request) bool {
	return a.store.Validate(a.Token(r))
}

// Middleware rejects requests without a valid session with a 401 response.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.Authenticated(r) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"authentication required"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
