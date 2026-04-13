package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"html/template"
	"net/http"
	"sync"
	"time"
)

// --- Session Store ---

type Session struct {
	Token     string
	CreatedAt time.Time
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
	maxAge   time.Duration
}

func NewSessionStore(maxAge time.Duration) *SessionStore {
	return &SessionStore{
		sessions: make(map[string]Session),
		maxAge:   maxAge,
	}
}

func (s *SessionStore) Create() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.URLEncoding.EncodeToString(b)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = Session{Token: token, CreatedAt: time.Now()}
	return token, nil
}

func (s *SessionStore) Valid(token string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[token]
	if !ok {
		return false
	}
	return time.Since(sess.CreatedAt) < s.maxAge
}

func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

// --- Auth Config ---

type Auth struct {
	Username   string // expected username
	Password   string // expected password (hash in production!)
	Store      *SessionStore
	CookieName string
}

func NewAuth(username, password string) *Auth {
	return &Auth{
		Username:   username,
		Password:   password,
		Store:      NewSessionStore(24 * time.Hour),
		CookieName: "session",
	}
}

// --- Login Handler ---

func (a *Auth) loginGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	t, err := template.ParseFS(templatesFS, "templates/base.html", "templates/dark.html", "templates/auth.html")
	if err != nil {
		log.Error().Err(err).Msg("parsing error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, "base", nil); err != nil {
		log.Error().Err(err).Msg("execution error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

}

func (a *Auth) loginPost(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Constant-time comparison to prevent timing attacks
	usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(a.Username)) == 1
	passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(a.Password)) == 1

	if !usernameMatch || !passwordMatch {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := a.Store.Create()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     a.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400,
	})

	// 303 is technically more correct for POST→GET redirects,
	// because it explicitly tells the client to follow up with a
	// GET regardless of the original method.
	// 302 does the same in practice (browsers always switch to GET),
	// but the spec originally said 302 should preserve the method.
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// --- Logout Handler ---

func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(a.CookieName); err == nil {
		a.Store.Delete(c.Value)
	}

	// Clear the cookie
	http.SetCookie(w, &http.Cookie{
		Name:   a.CookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// w.WriteHeader(http.StatusAccepted)
	http.Redirect(w, r, LOGIN_ROUTE, http.StatusSeeOther)
}

func (a *Auth) Login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.loginGet(w, r)
	case http.MethodPost:
		a.loginPost(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// --- Middleware ---

func (a *Auth) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ignore auth for login
		if r.URL.Path == LOGIN_ROUTE {
			next.ServeHTTP(w, r)
			return
		}
		// ignore assets
		if len(r.URL.Path) >= 8 && r.URL.Path[:8] == "/assets/" {
			next.ServeHTTP(w, r)
			return
		}
		c, err := r.Cookie(a.CookieName)
		if err != nil || !a.Store.Valid(c.Value) {
			http.Redirect(w, r, LOGIN_ROUTE, http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}
