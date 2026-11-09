package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	cfg    Config
	store  *Store
	auth   *Auth
	assets assets
}

func NewServer(cfg Config, store *Store, auth *Auth) *Server {
	return &Server{cfg: cfg, store: store, auth: auth, assets: loadAssets()}
}

// ---------- helpers ----------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func readJSON(w http.ResponseWriter, r *http.Request, v any, limit int64) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit)).Decode(v); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body")
		return false
	}
	return true
}

func (s *Server) storeErr(w http.ResponseWriter, err error) {
	var inv invalidErr
	var fb forbiddenErr
	switch {
	case errors.Is(err, ErrNotFound):
		writeErr(w, http.StatusNotFound, "Not found")
	case errors.Is(err, ErrConflict):
		writeErr(w, http.StatusConflict, "This was changed somewhere else. Reload to get the latest version.")
	case errors.As(err, &inv):
		writeErr(w, http.StatusBadRequest, inv.msg)
	case errors.As(err, &fb):
		writeErr(w, http.StatusForbidden, fb.msg)
	default:
		logf("error: %v", err)
		writeErr(w, http.StatusInternalServerError, "Could not save. Check the server logs.")
	}
}

// ---------- who is asking ----------

type gate int

const (
	gatePublic gate = iota
	gateSignedIn
	gateAdmin
)

type actorKey struct{}

func actor(r *http.Request) *Actor {
	a, _ := r.Context().Value(actorKey{}).(*Actor)
	return a
}

// guard resolves who the request acts for and enforces the gate:
//   - public: anyone, including visitors (handlers filter what they return);
//   - signed in: a valid session (always true when sign-in is off);
//   - admin: a signed-in admin.
func (s *Server) guard(g gate, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a := s.auth.Actor(r)
		if g != gatePublic && a == nil {
			writeErr(w, http.StatusUnauthorized, "Sign in required")
			return
		}
		if g == gateAdmin && !a.Admin() {
			writeErr(w, http.StatusForbidden, "Only admins can do that")
			return
		}
		h(w, r.WithContext(context.WithValue(r.Context(), actorKey{}, a)))
	}
}

// ---------- routing ----------

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	pub := func(h http.HandlerFunc) http.HandlerFunc { return s.guard(gatePublic, h) }
	in := func(h http.HandlerFunc) http.HandlerFunc { return s.guard(gateSignedIn, h) }
	admin := func(h http.HandlerFunc) http.HandlerFunc { return s.guard(gateAdmin, h) }

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "ok") })
	mux.HandleFunc("GET /{$}", pub(s.handleIndex))
	mux.HandleFunc("GET /favicon", s.handleFavicon)
	mux.HandleFunc("GET /favicon.ico", s.handleFavicon)
	mux.HandleFunc("GET /assets/", s.handleAsset)
	mux.HandleFunc("GET /api/icons", s.handleIcons)

	mux.HandleFunc("GET /api/session", pub(s.handleSession))
	mux.HandleFunc("POST /api/login", s.auth.handleLogin)
	mux.HandleFunc("POST /api/logout", s.auth.handleLogout)
	mux.HandleFunc("GET /api/state", pub(s.handleState))
	mux.HandleFunc("GET /api/search", pub(s.handleSearch))

	mux.HandleFunc("PUT /api/settings", in(s.handleSettings))
	mux.HandleFunc("PUT /api/layout", in(s.handleLayout))
	mux.HandleFunc("PUT /api/visibility", in(s.handleVisibility))

	mux.HandleFunc("GET /api/notes", in(s.handleNotes))
	mux.HandleFunc("GET /api/notes/{id}", in(s.handleNoteGet))
	mux.HandleFunc("POST /api/notes", in(s.handleNoteCreate))
	mux.HandleFunc("PUT /api/notes/{id}", in(s.handleNoteUpdate))
	mux.HandleFunc("DELETE /api/notes/{id}", in(s.handleNoteDelete))

	mux.HandleFunc("GET /api/tasks", in(s.handleTasks))
	mux.HandleFunc("POST /api/tasks", in(s.handleListCreate))
	mux.HandleFunc("PUT /api/tasks/order", in(s.handleListOrder))
	mux.HandleFunc("PUT /api/tasks/{id}", in(s.handleListUpdate))
	mux.HandleFunc("DELETE /api/tasks/{id}", in(s.handleListDelete))
	mux.HandleFunc("POST /api/tasks/{id}/items", in(s.handleItemCreate))
	mux.HandleFunc("PUT /api/items/{id}", in(s.handleItemUpdate))
	mux.HandleFunc("DELETE /api/items/{id}", in(s.handleItemDelete))

	mux.HandleFunc("POST /api/directory", in(s.handleEntryCreate))
	mux.HandleFunc("PUT /api/directory/{id}", in(s.handleEntryUpdate))
	mux.HandleFunc("DELETE /api/directory/{id}", in(s.handleEntryDelete))

	mux.HandleFunc("PUT /api/scratchpad", in(s.handleScratchpad))

	mux.HandleFunc("GET /api/me", in(s.handleMe))
	mux.HandleFunc("PUT /api/me", in(s.handleMeUpdate))
	mux.HandleFunc("PUT /api/me/password", in(s.handleMePassword))
	mux.HandleFunc("POST /api/me/signout-all", in(s.handleMeSignOutAll))

	mux.HandleFunc("GET /api/export", in(s.handleExport))
	mux.HandleFunc("POST /api/import", in(s.handleImport))
	mux.HandleFunc("GET /api/about", in(s.handleAbout))

	mux.HandleFunc("GET /api/users", admin(s.handleUsers))
	mux.HandleFunc("POST /api/users", admin(s.handleUserCreate))
	mux.HandleFunc("PUT /api/users/{id}", admin(s.handleUserUpdate))
	mux.HandleFunc("DELETE /api/users/{id}", admin(s.handleUserDelete))

	mux.HandleFunc("GET /api/db", admin(s.handleDBDownload))
	mux.HandleFunc("POST /api/db/restore", admin(s.handleDBRestore))
	mux.HandleFunc("GET /api/backups", admin(s.handleBackups))
	mux.HandleFunc("POST /api/backups", admin(s.handleBackupNow))
	mux.HandleFunc("GET /api/backups/{name}", admin(s.handleBackupDownload))
	mux.HandleFunc("POST /api/backups/{name}/restore", admin(s.handleRestore))
	return s.middleware(mux)
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		h.Set("Content-Security-Policy", "default-src 'self'; img-src * data: blob:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; frame-ancestors 'self'; base-uri 'none'; form-action 'self'")
		// Basic CSRF protection: state-changing API calls must come from our
		// own JavaScript, which always sends this header.
		if strings.HasPrefix(r.URL.Path, "/api/") && r.Method != http.MethodGet && r.Method != http.MethodHead {
			if r.Header.Get("X-Requested-With") != "DotSlash" {
				writeErr(w, http.StatusForbidden, "Missing request header")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// ---------- pages ----------

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	st, _ := s.store.Settings(actor(r))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(renderIndex(s.assets, st, s.store.Instance()))
}

func (s *Server) handleAsset(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/assets/")
	if name == "" || strings.Contains(name, "..") || name == "index.html" {
		http.NotFound(w, r)
		return
	}
	if r.URL.Query().Get("v") != "" {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	http.ServeFileFS(w, r, s.assets.files, name)
}

func (s *Server) handleFavicon(w http.ResponseWriter, r *http.Request) {
	in := s.store.Instance()
	w.Header().Set("Cache-Control", "public, max-age=300")
	switch in.Icon.Type {
	case "image":
		if i := strings.Index(in.Icon.Image, ";base64,"); i > 0 {
			b, err := base64.StdEncoding.DecodeString(in.Icon.Image[i+8:])
			if err == nil {
				w.Header().Set("Content-Type", strings.TrimPrefix(in.Icon.Image[:i], "data:"))
				w.Write(b)
				return
			}
		}
	case "library":
		if u := s.libraryIconURL(in.Icon.Name); u != "" {
			http.Redirect(w, r, u, http.StatusFound)
			return
		}
	}
	bg := in.Icon.Color
	if bg == "" {
		bg = in.Accent
	}
	fg := "#0D1220"
	if luminance(bg) < 0.45 {
		fg = "#FFFFFF"
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	if in.Icon.Type == "letter" {
		fmt.Fprintf(w, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64"><rect width="64" height="64" rx="14" fill="%s"/><text x="32" y="43" font-family="system-ui,-apple-system,Segoe UI,sans-serif" font-size="30" font-weight="600" text-anchor="middle" fill="%s">%s</text></svg>`,
			bg, fg, html.EscapeString(in.Icon.Letter))
		return
	}
	w.Write([]byte(logoSVG(bg, fg)))
}

// logoSVG is the DotSlash mark — "./" — on a rounded square.
func logoSVG(bg, fg string) string {
	return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64"><rect width="64" height="64" rx="14" fill="` + bg +
		`"/><circle cx="22.5" cy="43" r="4.6" fill="` + fg + `"/><path d="M42 17 33 47" fill="none" stroke="` + fg + `" stroke-width="5.5" stroke-linecap="round"/></svg>`
}

// handleIcons serves the embedded index of Dashboard Icons names
// ({"svg":[...], "png":[...]} where png lists names that only exist as PNG).
func (s *Server) handleIcons(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Header().Set("Vary", "Accept-Encoding")
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(s.assets.iconsGz)
		return
	}
	w.Write(iconIndex)
}

func (s *Server) libraryIconURL(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://") {
		return name
	}
	clean := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			return r
		}
		return -1
	}, strings.ToLower(name))
	if clean == "" {
		return ""
	}
	return s.cfg.IconCDN + "/svg/" + clean + ".svg"
}

func luminance(hex string) float64 {
	if len(hex) != 7 {
		return 0.5
	}
	v, err := strconv.ParseUint(hex[1:], 16, 32)
	if err != nil {
		return 0.5
	}
	r, g, b := float64(v>>16&0xff)/255, float64(v>>8&0xff)/255, float64(v&0xff)/255
	return 0.2126*r + 0.7152*g + 0.0722*b
}

// ---------- session & state ----------

type meJSON struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Display  string `json:"display"`
	Role     string `json:"role"`
}

func (s *Server) me(a *Actor) *meJSON {
	if a == nil {
		return nil
	}
	u, err := s.store.UserByID(a.ID)
	if err != nil {
		return nil
	}
	return &meJSON{ID: u.ID, Username: u.Username, Name: u.Name, Display: u.Display(), Role: u.Role}
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	a := actor(r)
	out := map[string]any{
		"auth_enabled": s.auth.enabled,
		"signed_in":    a != nil,
		"icon_cdn":     s.cfg.IconCDN,
		"version":      version,
		"session_days": s.cfg.SessionDays,
		"repo":         repoURL,
		"me":           s.me(a),
	}
	if s.auth.enabled && a != nil {
		out["expires"] = s.auth.SessionExpiry(r)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	a := actor(r)
	st, err := s.store.Settings(a)
	if err != nil {
		s.storeErr(w, err)
		return
	}
	pages, err := s.store.Pages(a)
	if err != nil {
		s.storeErr(w, err)
		return
	}
	if a == nil {
		// Visitors get the public pages and the look of the dashboard — nothing else.
		writeJSON(w, http.StatusOK, map[string]any{"settings": st, "pages": pages, "signed_in": false})
		return
	}
	notes, err1 := s.store.NoteMetas(a)
	tasks, err2 := s.store.TaskLists(a)
	dir, err3 := s.store.Entries(a)
	pad, err4 := s.store.Scratchpad(a)
	if err := errors.Join(err1, err2, err3, err4); err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"settings": st, "pages": pages, "signed_in": true, "me": s.me(a),
		"notes": notes, "tasks": tasks, "directory": dir, "scratchpad": pad,
	})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if len(q) > 200 {
		q = q[:200]
	}
	res, err := s.store.Search(actor(r), q, 30)
	if err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	var st Settings
	if !readJSON(w, r, &st, 1<<20) {
		return
	}
	out, err := s.store.SaveSettings(actor(r), st)
	if err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleLayout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Pages []PageIn `json:"pages"`
	}
	if !readJSON(w, r, &body, 8<<20) {
		return
	}
	pages, conflicts, err := s.store.SaveLayout(actor(r), body.Pages)
	if err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"pages": pages, "conflicts": conflicts})
}

func (s *Server) handleVisibility(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Kind       string `json:"kind"` // page | note | list | entry
		ID         string `json:"id"`
		Visibility string `json:"visibility"`
	}
	if !readJSON(w, r, &b, 4096) {
		return
	}
	acc, err := s.store.SetVisibility(actor(r), b.Kind, b.ID, b.Visibility)
	if err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, acc)
}

// ---------- notes ----------

func (s *Server) handleNotes(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.NoteMetas(actor(r))
	if err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleNoteGet(w http.ResponseWriter, r *http.Request) {
	n, err := s.store.Note(actor(r), r.PathValue("id"))
	if err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (s *Server) handleNoteCreate(w http.ResponseWriter, r *http.Request) {
	var p NotePatch
	if !readJSON(w, r, &p, int64(s.cfg.MaxNoteKB+16)*1024) {
		return
	}
	n, err := s.store.CreateNote(actor(r), p)
	if err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, n)
}

func (s *Server) handleNoteUpdate(w http.ResponseWriter, r *http.Request) {
	var p NotePatch
	if !readJSON(w, r, &p, int64(s.cfg.MaxNoteKB+16)*1024) {
		return
	}
	n, err := s.store.UpdateNote(actor(r), r.PathValue("id"), p)
	if errors.Is(err, ErrConflict) {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "This note was changed somewhere else.", "note": n})
		return
	}
	if err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (s *Server) handleNoteDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteNote(actor(r), r.PathValue("id")); err != nil {
		s.storeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- tasks ----------

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	l, err := s.store.TaskLists(actor(r))
	if err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, l)
}

func (s *Server) handleListCreate(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Title      string `json:"title"`
		Visibility string `json:"visibility"`
	}
	if !readJSON(w, r, &b, 4096) {
		return
	}
	l, err := s.store.CreateList(actor(r), b.Title, b.Visibility)
	if err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, l)
}

func (s *Server) handleListUpdate(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Title string `json:"title"`
	}
	if !readJSON(w, r, &b, 4096) {
		return
	}
	l, err := s.store.RenameList(actor(r), r.PathValue("id"), b.Title)
	if err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, l)
}

func (s *Server) handleListDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteList(actor(r), r.PathValue("id")); err != nil {
		s.storeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListOrder(w http.ResponseWriter, r *http.Request) {
	var b struct {
		IDs []string `json:"ids"`
	}
	if !readJSON(w, r, &b, 1<<20) {
		return
	}
	l, err := s.store.OrderLists(actor(r), b.IDs)
	if err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, l)
}

func (s *Server) handleItemCreate(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Text string `json:"text"`
		Due  string `json:"due"`
	}
	if !readJSON(w, r, &b, 8192) {
		return
	}
	it, err := s.store.AddItem(actor(r), r.PathValue("id"), b.Text, b.Due)
	if err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, it)
}

func (s *Server) handleItemUpdate(w http.ResponseWriter, r *http.Request) {
	var p ItemPatch
	if !readJSON(w, r, &p, 8192) {
		return
	}
	it, err := s.store.UpdateItem(actor(r), r.PathValue("id"), p)
	if err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, it)
}

func (s *Server) handleItemDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteItem(actor(r), r.PathValue("id")); err != nil {
		s.storeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- directory, scratchpad ----------

func (s *Server) handleEntryCreate(w http.ResponseWriter, r *http.Request) {
	var p EntryPatch
	if !readJSON(w, r, &p, 16384) {
		return
	}
	e, err := s.store.CreateEntry(actor(r), p)
	if err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (s *Server) handleEntryUpdate(w http.ResponseWriter, r *http.Request) {
	var p EntryPatch
	if !readJSON(w, r, &p, 16384) {
		return
	}
	e, err := s.store.UpdateEntry(actor(r), r.PathValue("id"), p)
	if err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (s *Server) handleEntryDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteEntry(actor(r), r.PathValue("id")); err != nil {
		s.storeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleScratchpad(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Content      string `json:"content"`
		BaseModified string `json:"base_modified"`
	}
	if !readJSON(w, r, &b, maxScratchpadLen+4096) {
		return
	}
	pad, err := s.store.SaveScratchpad(actor(r), b.Content, b.BaseModified)
	if errors.Is(err, ErrConflict) {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "The scratchpad was changed somewhere else.", "scratchpad": pad})
		return
	}
	if err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pad)
}

// ---------- me ----------

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.me(actor(r)))
}

func (s *Server) handleMeUpdate(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Name string `json:"name"`
	}
	if !readJSON(w, r, &b, 4096) {
		return
	}
	if _, err := s.store.UpdateUser(actor(r).ID, UserPatch{Name: &b.Name}); err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.me(actor(r)))
}

func (s *Server) handleMePassword(w http.ResponseWriter, r *http.Request) {
	if !s.auth.enabled {
		writeErr(w, http.StatusBadRequest, "Sign-in is off. Set DS_PASSWORD on the server to use passwords.")
		return
	}
	var b struct {
		Current  string `json:"current"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &b, 4096) {
		return
	}
	u, err := s.store.ChangeOwnPassword(actor(r), b.Current, b.Password)
	if err != nil {
		s.storeErr(w, err)
		return
	}
	s.auth.Reissue(w, r, u)
	writeJSON(w, http.StatusOK, map[string]any{"changed": true})
}

func (s *Server) handleMeSignOutAll(w http.ResponseWriter, r *http.Request) {
	u, err := s.store.SignOutEverywhere(actor(r).ID)
	if err != nil {
		s.storeErr(w, err)
		return
	}
	s.auth.Reissue(w, r, u)
	writeJSON(w, http.StatusOK, map[string]any{"done": true})
}

// ---------- users (admin) ----------

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.Users()
	if err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleUserCreate(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Username string `json:"username"`
		Name     string `json:"name"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if !readJSON(w, r, &b, 4096) {
		return
	}
	u, err := s.store.CreateUser(b.Username, b.Name, b.Password, b.Role)
	if err != nil {
		s.storeErr(w, err)
		return
	}
	logf("user %q created by %q", u.Username, actor(r).Username)
	writeJSON(w, http.StatusCreated, u)
}

func (s *Server) handleUserUpdate(w http.ResponseWriter, r *http.Request) {
	var p UserPatch
	if !readJSON(w, r, &p, 4096) {
		return
	}
	a := actor(r)
	u, err := s.store.UpdateUser(r.PathValue("id"), p)
	if err != nil {
		s.storeErr(w, err)
		return
	}
	if u.ID == a.ID {
		s.auth.Reissue(w, r, u) // stay signed in after changing your own role or password
	}
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) handleUserDelete(w http.ResponseWriter, r *http.Request) {
	handover := r.URL.Query().Get("handover") == "1"
	if err := s.store.DeleteUser(r.PathValue("id"), actor(r), handover); err != nil {
		s.storeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- data ----------

func download(w http.ResponseWriter, name, ctype string) {
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("Cache-Control", "no-store")
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	a := actor(r)
	ex, err := s.store.ExportMine(a)
	if err != nil {
		s.storeErr(w, err)
		return
	}
	b, _ := json.MarshalIndent(ex, "", "  ")
	download(w, "dotslash-"+a.Username+"-"+time.Now().Format("2006-01-02")+".json", "application/json")
	w.Write(b)
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 50<<20))
	if err != nil {
		writeErr(w, http.StatusRequestEntityTooLarge, "File is too large (50 MB max)")
		return
	}
	ex, sum, err := ParseExport(raw)
	if err != nil {
		s.storeErr(w, err)
		return
	}
	if r.URL.Query().Get("dry_run") == "1" {
		writeJSON(w, http.StatusOK, map[string]any{"summary": sum, "applied": false})
		return
	}
	a := actor(r)
	if err := s.store.ImportMine(a, ex); err != nil {
		s.storeErr(w, err)
		return
	}
	logf("%q imported data: %+v", a.Username, sum)
	writeJSON(w, http.StatusOK, map[string]any{"summary": sum, "applied": true})
}

func (s *Server) handleDBDownload(w http.ResponseWriter, r *http.Request) {
	download(w, "dotslash-"+time.Now().Format("2006-01-02-150405")+".db", "application/vnd.sqlite3")
	if err := s.store.WriteSnapshot(w); err != nil {
		logf("database download failed: %v", err)
	}
}

func (s *Server) handleDBRestore(w http.ResponseWriter, r *http.Request) {
	if err := s.store.RestoreFile(http.MaxBytesReader(w, r.Body, 512<<20)); err != nil {
		s.storeErr(w, err)
		return
	}
	logf("database restored from an uploaded file by %q", actor(r).Username)
	writeJSON(w, http.StatusOK, map[string]any{"restored": true})
}

func (s *Server) handleBackups(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"backups": s.store.Backups(), "keep": s.cfg.BackupKeep})
}

func (s *Server) handleBackupNow(w http.ResponseWriter, r *http.Request) {
	b, err := s.store.Backup("manual")
	if err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (s *Server) handleBackupDownload(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.BackupPath(r.PathValue("name"))
	if err != nil {
		s.storeErr(w, err)
		return
	}
	download(w, r.PathValue("name"), "application/vnd.sqlite3")
	http.ServeFile(w, r, p)
}

func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Restore(r.PathValue("name")); err != nil {
		s.storeErr(w, err)
		return
	}
	logf("restored backup %s", r.PathValue("name"))
	writeJSON(w, http.StatusOK, map[string]any{"restored": true})
}

func rssBytes() uint64 {
	b, err := os.ReadFile("/proc/self/status")
	if err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "VmRSS:") {
				f := strings.Fields(line)
				if len(f) >= 2 {
					kb, _ := strconv.ParseUint(f[1], 10, 64)
					return kb * 1024
				}
			}
		}
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Sys
}

func (s *Server) handleAbout(w http.ResponseWriter, r *http.Request) {
	a := actor(r)
	var size int64
	for _, suffix := range []string{"", "-wal"} {
		if fi, err := os.Stat(s.store.Path() + suffix); err == nil {
			size += fi.Size()
		}
	}
	out := map[string]any{
		"version": version,
		"memory":  rssBytes(),
		"go":      runtime.Version(),
		"auth":    s.auth.enabled,
		"mine":    s.store.MyCounts(a),
		"backups": s.cfg.BackupKeep,
	}
	if a.Admin() {
		out["data_file"] = s.store.Path()
		out["data_bytes"] = size
	}
	writeJSON(w, http.StatusOK, out)
}
