package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const cookieName = "ds_session"

// Auth signs people in. Sessions are signed cookies carrying the user ID and
// the user's token version: bumping the version (password change, sign out
// everywhere, disabling the account) ends every session at once.
type Auth struct {
	enabled     bool
	store       *Store
	key         []byte
	sessionDays int
	secure      string // auto | true | false
	trustProxy  bool
	dummy       string

	mu       sync.Mutex
	failures map[string][]time.Time
}

func NewAuth(c Config, store *Store) (*Auth, error) {
	a := &Auth{
		enabled:     c.Password != "",
		store:       store,
		sessionDays: c.SessionDays,
		secure:      c.CookieSecure,
		trustProxy:  c.TrustProxy,
		failures:    map[string][]time.Time{},
	}
	secret := c.Secret
	if secret == "" {
		// Keep sessions across restarts without asking for DS_SECRET.
		var err error
		if secret, err = store.Meta("session_secret"); err != nil {
			return nil, err
		}
		if secret == "" {
			b := make([]byte, 32)
			_, _ = rand.Read(b)
			secret = hex.EncodeToString(b)
			if err := store.SetMeta("session_secret", secret); err != nil {
				return nil, err
			}
		}
	}
	h := sha256.Sum256([]byte("dotslash-session|" + secret))
	a.key = h[:]
	a.dummy, _ = hashPassword("not-a-real-password")
	return a, nil
}

func (a *Auth) sign(payload string) string {
	m := hmac.New(sha256.New, a.key)
	m.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

// token: base64(id|ver|exp|remember).sig
func (a *Auth) makeToken(u User, exp time.Time, remember bool) string {
	r := "0"
	if remember {
		r = "1"
	}
	payload := u.ID + "|" + strconv.FormatInt(u.tokenVer, 10) + "|" + strconv.FormatInt(exp.Unix(), 10) + "|" + r
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + a.sign(payload)
}

type session struct {
	userID   string
	ver      int64
	exp      time.Time
	remember bool
}

func (a *Auth) parse(r *http.Request) (session, bool) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return session{}, false
	}
	parts := strings.SplitN(c.Value, ".", 2)
	if len(parts) != 2 {
		return session{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return session{}, false
	}
	payload := string(raw)
	if !hmac.Equal([]byte(a.sign(payload)), []byte(parts[1])) {
		return session{}, false
	}
	f := strings.Split(payload, "|")
	if len(f) != 4 {
		return session{}, false
	}
	ver, err1 := strconv.ParseInt(f[1], 10, 64)
	exp, err2 := strconv.ParseInt(f[2], 10, 64)
	if err1 != nil || err2 != nil || time.Now().Unix() > exp {
		return session{}, false
	}
	return session{userID: f[0], ver: ver, exp: time.Unix(exp, 0), remember: f[3] == "1"}, true
}

// Actor returns who the request acts for, or nil for a visitor. With sign-in
// off, everyone acts as the first admin.
func (a *Auth) Actor(r *http.Request) *Actor {
	if !a.enabled {
		act, err := a.store.OpenModeActor()
		if err != nil {
			return nil
		}
		return act
	}
	s, ok := a.parse(r)
	if !ok {
		return nil
	}
	u, err := a.store.UserByID(s.userID)
	if err != nil || u.Disabled || u.tokenVer != s.ver || u.pwHash == "" {
		return nil
	}
	return actorOf(u)
}

func (a *Auth) SessionExpiry(r *http.Request) time.Time {
	s, _ := a.parse(r)
	return s.exp
}

func (a *Auth) clientIP(r *http.Request) string {
	if a.trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			return strings.TrimSpace(strings.Split(xff, ",")[0])
		}
		if xr := r.Header.Get("X-Real-IP"); xr != "" {
			return strings.TrimSpace(xr)
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (a *Auth) isSecure(r *http.Request) bool {
	switch a.secure {
	case "true":
		return true
	case "false":
		return false
	}
	if r.TLS != nil {
		return true
	}
	return a.trustProxy && strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// ---------- rate limiting: 5 failures a minute per IP and per username ----------

const maxFailures = 5

func (a *Auth) allowed(keys ...string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	cut := time.Now().Add(-time.Minute)
	for _, k := range keys {
		kept := a.failures[k][:0]
		for _, t := range a.failures[k] {
			if t.After(cut) {
				kept = append(kept, t)
			}
		}
		if len(kept) == 0 {
			delete(a.failures, k)
		} else {
			a.failures[k] = kept
		}
		if len(kept) >= maxFailures {
			return false
		}
	}
	return true
}

func (a *Auth) fail(keys ...string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, k := range keys {
		a.failures[k] = append(a.failures[k], time.Now())
	}
}

// ---------- handlers ----------

func (a *Auth) setCookie(w http.ResponseWriter, r *http.Request, u User, remember bool) time.Time {
	ttl := 12 * time.Hour
	if remember {
		ttl = time.Duration(a.sessionDays) * 24 * time.Hour
	}
	exp := time.Now().Add(ttl)
	c := &http.Cookie{
		Name:     cookieName,
		Value:    a.makeToken(u, exp, remember),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   a.isSecure(r),
	}
	if remember {
		c.Expires = exp
		c.MaxAge = int(ttl.Seconds())
	}
	http.SetCookie(w, c)
	return exp
}

// Reissue gives the current session a fresh cookie after the user's token
// version changed (so this tab stays signed in while others are signed out).
func (a *Auth) Reissue(w http.ResponseWriter, r *http.Request, u User) {
	if !a.enabled {
		return
	}
	s, _ := a.parse(r)
	a.setCookie(w, r, u, s.remember)
}

func (a *Auth) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !a.enabled {
		writeJSON(w, http.StatusOK, map[string]any{"signed_in": true})
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Remember bool   `json:"remember"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request")
		return
	}
	ip := "ip:" + a.clientIP(r)
	name := "user:" + strings.ToLower(strings.TrimSpace(body.Username))
	if !a.allowed(ip, name) {
		writeErr(w, http.StatusTooManyRequests, "Too many attempts. Try again in a minute.")
		return
	}
	u, err := a.store.UserByName(body.Username)
	ok := false
	if err == nil && u.pwHash != "" {
		ok = checkPassword(u.pwHash, body.Password)
	} else {
		checkPassword(a.dummy, body.Password) // same timing for unknown names
	}
	if !ok || u.Disabled {
		a.fail(ip, name)
		time.Sleep(300 * time.Millisecond)
		logf("failed sign-in for %q from %s", body.Username, a.clientIP(r))
		writeErr(w, http.StatusUnauthorized, "Wrong username or password")
		return
	}
	exp := a.setCookie(w, r, u, body.Remember)
	writeJSON(w, http.StatusOK, map[string]any{"signed_in": true, "expires": exp})
}

func (a *Auth) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: a.isSecure(r)})
	writeJSON(w, http.StatusOK, map[string]any{"signed_in": false})
}
