package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// ---------- content ----------

type Link struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	NewTab      bool   `json:"new_tab"`
}

type Group struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Links []Link `json:"links"`
}

// Access describes who owns an item and what the current user may do with it.
type Access struct {
	Visibility string `json:"visibility"` // private | shared | public (pages only)
	Owner      string `json:"owner"`
	OwnerName  string `json:"owner_name"`
	CanEdit    bool   `json:"can_edit"`
	CanManage  bool   `json:"can_manage"` // change visibility, delete
}

type Page struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Groups []Group `json:"groups"`
	Rev    int64   `json:"rev"`
	Access
}

type Note struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"` // note (Markdown) | snippet (plain text to copy)
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Pinned     bool      `json:"pinned"` // per user
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
	Access
}

type NoteMeta struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Title      string    `json:"title"`
	Pinned     bool      `json:"pinned"`
	Snippet    string    `json:"snippet"`
	Content    string    `json:"content,omitempty"` // only for snippets, which are shown as copy cards
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
	Access
}

type TaskItem struct {
	ID         string     `json:"id"`
	Text       string     `json:"text"`
	Done       bool       `json:"done"`
	Due        string     `json:"due"`
	CreatedAt  time.Time  `json:"created_at"`
	DoneAt     *time.Time `json:"done_at,omitempty"`
	DoneByName string     `json:"done_by_name,omitempty"`
}

type TaskList struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	Items      []TaskItem `json:"items"`
	CreatedAt  time.Time  `json:"created_at"`
	ModifiedAt time.Time  `json:"modified_at"`
	Access
}

// Entry is one row of the Directory: a person, a business, a place or a device.
type Entry struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Category   string    `json:"category"`
	Phone      string    `json:"phone"`
	Email      string    `json:"email"`
	Address    string    `json:"address"`
	URL        string    `json:"url"`
	IP         string    `json:"ip"`
	MAC        string    `json:"mac"`
	Notes      string    `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
	Access
}

func (e Entry) summary() string {
	parts := []string{}
	for _, v := range []string{e.Category, e.Phone, e.Email, e.IP, e.Address} {
		if v != "" {
			parts = append(parts, v)
		}
		if len(parts) == 3 {
			break
		}
	}
	return strings.Join(parts, " · ")
}

type Scratchpad struct {
	Content    string     `json:"content"`
	ModifiedAt *time.Time `json:"modified_at"`
}

// ---------- settings ----------

type Icon struct {
	Type   string `json:"type"`   // logo | letter | image | library
	Letter string `json:"letter"` // for letter
	Color  string `json:"color"`  // background for letter/logo ("" = accent)
	Image  string `json:"image"`  // data:image/... for image
	Name   string `json:"name"`   // dashboard-icons name or URL for library
}

// Instance holds settings shared by everyone; only admins change them.
type Instance struct {
	Title             string `json:"title"`
	Icon              Icon   `json:"icon"`
	Accent            string `json:"accent"` // default accent, also used for visitors and the favicon
	Layout            string `json:"layout"` // default link layout, also used for visitors
	HeaderBg          string `json:"header_bg"`
	HeaderText        string `json:"header_text"`
	PublicStart       string `json:"public_start"` // "" = first public page | links:<id> | blank
	MembersEditShared bool   `json:"members_edit_shared"`
}

// Prefs are each person's own choices.
type Prefs struct {
	Theme          string   `json:"theme"`  // system | dark | light
	Accent         string   `json:"accent"` // "" = instance default
	Layout         string   `json:"layout"` // "" = instance default
	StartOn        string   `json:"start_on"`
	IdleMinutes    int      `json:"idle_minutes"`
	NewTab         bool     `json:"new_tab"`
	HiddenSections []string `json:"hidden_sections"` // any of notes, tasks, directory, scratchpad
}

func (p Prefs) Hidden(section string) bool {
	for _, h := range p.HiddenSections {
		if h == section {
			return true
		}
	}
	return false
}

// Settings is the merged view the browser works with: instance + the user's prefs.
type Settings struct {
	// instance (admins only)
	Title             string `json:"title"`
	Icon              Icon   `json:"icon"`
	HeaderBg          string `json:"header_bg"`
	HeaderText        string `json:"header_text"`
	DefaultAccent     string `json:"default_accent"`
	DefaultLayout     string `json:"default_layout"`
	PublicStart       string `json:"public_start"`
	MembersEditShared bool   `json:"members_edit_shared"`
	// the user's own
	Theme          string   `json:"theme"`
	PrefAccent     string   `json:"pref_accent"`
	PrefLayout     string   `json:"pref_layout"`
	StartOn        string   `json:"start_on"`
	IdleMinutes    int      `json:"idle_minutes"`
	NewTab         bool     `json:"new_tab"`
	HiddenSections []string `json:"hidden_sections"`
	// resolved
	Accent string `json:"accent"`
	Layout string `json:"layout"`
}

func mergeSettings(in Instance, p Prefs) Settings {
	s := Settings{
		Title: in.Title, Icon: in.Icon, HeaderBg: in.HeaderBg, HeaderText: in.HeaderText,
		DefaultAccent: in.Accent, DefaultLayout: in.Layout, PublicStart: in.PublicStart, MembersEditShared: in.MembersEditShared,
		Theme: p.Theme, PrefAccent: p.Accent, PrefLayout: p.Layout, StartOn: p.StartOn, IdleMinutes: p.IdleMinutes,
		NewTab: p.NewTab, HiddenSections: p.HiddenSections,
		Accent: in.Accent, Layout: in.Layout,
	}
	if p.Accent != "" {
		s.Accent = p.Accent
	}
	if p.Layout != "" {
		s.Layout = p.Layout
	}
	if s.HiddenSections == nil {
		s.HiddenSections = []string{}
	}
	return s
}

func (s Settings) split() (Instance, Prefs) {
	return Instance{Title: s.Title, Icon: s.Icon, Accent: s.DefaultAccent, Layout: s.DefaultLayout, HeaderBg: s.HeaderBg, HeaderText: s.HeaderText,
			PublicStart: s.PublicStart, MembersEditShared: s.MembersEditShared},
		Prefs{Theme: s.Theme, Accent: s.PrefAccent, Layout: s.PrefLayout, StartOn: s.StartOn, IdleMinutes: s.IdleMinutes, NewTab: s.NewTab, HiddenSections: s.HiddenSections}
}

func defaultInstance(title string) Instance {
	if title == "" {
		title = appName
	}
	return Instance{Title: title, Icon: Icon{Type: "logo", Letter: strings.ToUpper(firstRune(title)), Color: "#F0B45A"}, Accent: "#F0B45A", Layout: "cards"}
}

func defaultPrefs() Prefs {
	return Prefs{Theme: "system", StartOn: "links", NewTab: true, HiddenSections: []string{}}
}

// ---------- errors ----------

var (
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("conflict")
	ErrInvalid   = errors.New("invalid")
	ErrForbidden = errors.New("forbidden")
)

type invalidErr struct{ msg string }

func (e invalidErr) Error() string { return e.msg }
func (e invalidErr) Unwrap() error { return ErrInvalid }

func invalid(format string, a ...any) error { return invalidErr{fmt.Sprintf(format, a...)} }

type forbiddenErr struct{ msg string }

func (e forbiddenErr) Error() string { return e.msg }
func (e forbiddenErr) Unwrap() error { return ErrForbidden }

func forbidden(format string, a ...any) error { return forbiddenErr{fmt.Sprintf(format, a...)} }

// ---------- validation ----------

var (
	hexColor   = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	dateRe     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	startOnRe  = regexp.MustCompile(`^(links|links:[A-Za-z0-9_-]{1,40}|notes|tasks|directory|last|blank)$`)
	pubStartRe = regexp.MustCompile(`^(|links:[A-Za-z0-9_-]{1,40}|blank)$`)
	usernameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._@-]{0,39}$`)
	idRe       = regexp.MustCompile(`^[A-Za-z0-9_-]{1,40}$`)
)

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func firstRune(s string) string {
	for _, r := range strings.TrimSpace(s) {
		return string(r)
	}
	return "D"
}

// clip trims spaces and cuts s to at most n runes.
func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return strings.TrimSpace(string([]rune(s)[:n]))
}

func safeURL(u string) string {
	u = strings.TrimSpace(u)
	l := strings.ToLower(strings.Join(strings.Fields(u), ""))
	for _, bad := range []string{"javascript:", "data:", "vbscript:", "file:"} {
		if strings.HasPrefix(l, bad) {
			return ""
		}
	}
	return clip(u, 2048)
}

func validIconImage(s string) bool {
	if len(s) > 400*1024 {
		return false
	}
	for _, p := range []string{"data:image/png;base64,", "data:image/jpeg;base64,", "data:image/webp;base64,", "data:image/gif;base64,", "data:image/x-icon;base64,", "data:image/vnd.microsoft.icon;base64,"} {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func normalizeInstance(in *Instance) {
	def := defaultInstance(appName)
	in.Title = clip(in.Title, 60)
	if in.Title == "" {
		in.Title = def.Title
	}
	switch in.Icon.Type {
	case "logo", "letter", "image", "library":
	default:
		in.Icon.Type = "logo"
	}
	in.Icon.Letter = clip(in.Icon.Letter, 2)
	if in.Icon.Letter == "" {
		in.Icon.Letter = strings.ToUpper(firstRune(in.Title))
	}
	if in.Icon.Color != "" && !hexColor.MatchString(in.Icon.Color) {
		in.Icon.Color = ""
	}
	if in.Icon.Image != "" && !validIconImage(in.Icon.Image) {
		in.Icon.Image = ""
	}
	if in.Icon.Type == "image" && in.Icon.Image == "" {
		in.Icon.Type = "logo"
	}
	in.Icon.Name = clip(in.Icon.Name, 300)
	if !hexColor.MatchString(in.Accent) {
		in.Accent = def.Accent
	}
	if !validLayout(in.Layout) {
		in.Layout = def.Layout
	}
	if in.HeaderBg != "" && !hexColor.MatchString(in.HeaderBg) {
		in.HeaderBg = ""
	}
	if in.HeaderText != "" && !hexColor.MatchString(in.HeaderText) {
		in.HeaderText = ""
	}
	if !pubStartRe.MatchString(in.PublicStart) {
		in.PublicStart = ""
	}
}

func validLayout(l string) bool {
	switch l {
	case "cards", "tiles", "compact", "rows":
		return true
	}
	return false
}

func normalizePrefs(p *Prefs) {
	def := defaultPrefs()
	switch p.Theme {
	case "system", "dark", "light":
	default:
		p.Theme = def.Theme
	}
	if p.Accent != "" && !hexColor.MatchString(p.Accent) {
		p.Accent = ""
	}
	if p.Layout != "" && !validLayout(p.Layout) {
		p.Layout = ""
	}
	if !startOnRe.MatchString(p.StartOn) {
		p.StartOn = def.StartOn
	}
	p.IdleMinutes = max(0, min(p.IdleMinutes, 1440))
	hidden := []string{}
	seen := map[string]bool{}
	for _, h := range p.HiddenSections {
		switch h {
		case "notes", "tasks", "directory", "scratchpad":
			if !seen[h] {
				seen[h] = true
				hidden = append(hidden, h)
			}
		}
	}
	p.HiddenSections = hidden
}

// normalizeGroups validates a page's groups and links. IDs that are missing or
// repeated are replaced; links with neither a name nor a URL are dropped.
func normalizeGroups(groups []Group) []Group {
	seen := map[string]bool{}
	id := func(v string) string {
		if !idRe.MatchString(v) || seen[v] {
			v = newID()
		}
		seen[v] = true
		return v
	}
	out := make([]Group, 0, len(groups))
	for _, g := range groups {
		g.ID = id(g.ID)
		g.Name = clip(g.Name, 80)
		if g.Name == "" {
			g.Name = "Untitled"
		}
		links := make([]Link, 0, len(g.Links))
		for _, l := range g.Links {
			l.ID = id(l.ID)
			l.Name = clip(l.Name, 120)
			l.URL = safeURL(l.URL)
			l.Icon = clip(l.Icon, 500)
			if low := strings.ToLower(l.Icon); strings.HasPrefix(low, "javascript:") || strings.HasPrefix(low, "data:") {
				l.Icon = ""
			}
			l.Description = clip(l.Description, 300)
			if l.Name == "" && l.URL == "" {
				continue
			}
			if l.Name == "" {
				l.Name = l.URL
			}
			links = append(links, l)
		}
		g.Links = links
		out = append(out, g)
	}
	return out
}

func noteType(t string) string {
	if t == "snippet" {
		return t
	}
	return "note"
}

func normalizeEntry(e *Entry) {
	e.Name = clip(e.Name, 120)
	e.Category = clip(e.Category, 40)
	e.Phone = clip(e.Phone, 60)
	e.Email = clip(e.Email, 200)
	e.Address = clip(e.Address, 300)
	e.URL = safeURL(e.URL)
	e.IP = clip(e.IP, 60)
	e.MAC = clip(e.MAC, 60)
	e.Notes = clip(e.Notes, 1000)
}

// ---------- text helpers ----------

func snippetOf(content string, n int) string {
	var b strings.Builder
	inFence := false
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "```") {
			inFence = !inFence
			continue
		}
		if t == "" || inFence {
			continue
		}
		t = strings.TrimLeft(t, "#>-*+ ")
		t = strings.NewReplacer("**", "", "__", "", "`", "", "[ ]", "", "[x]", "").Replace(t)
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString(t)
		if b.Len() > n {
			break
		}
	}
	return clip(b.String(), n)
}

func firstLine(s string, n int) string {
	for _, l := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(l); t != "" {
			return clip(t, n)
		}
	}
	return ""
}

func contextAround(content, term string, n int) string {
	flat := strings.Join(strings.Fields(content), " ")
	i := strings.Index(strings.ToLower(flat), term)
	if i < 0 {
		return snippetOf(content, n)
	}
	start := max(0, i-n/3)
	r := []rune(flat)
	rs := utf8.RuneCountInString(flat[:start])
	end := min(rs+n, len(r))
	out := string(r[rs:end])
	if rs > 0 {
		out = "…" + out
	}
	if end < len(r) {
		out += "…"
	}
	return out
}

// ---------- time ----------

func now() time.Time { return time.Now().UTC() }

func ts(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func pt(s string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, s)
	return t
}

func ptp(s string) *time.Time {
	if s == "" {
		return nil
	}
	t := pt(s)
	return &t
}

// sameTime compares a client-sent timestamp with a stored one.
func sameTime(client, stored string) bool {
	a, err := time.Parse(time.RFC3339Nano, client)
	if err != nil {
		return false
	}
	return a.Equal(pt(stored))
}
