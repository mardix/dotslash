package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	pbkdfIter = 1000 // fast hashing in tests
	cache, _ := os.MkdirTemp("", "ds-test-cache")
	setupRuntime(cache) // share SQLite's compiled engine across tests
	code := m.Run()
	os.RemoveAll(cache)
	os.Exit(code)
}

type fixture struct {
	s          *Store
	admin, ana *Actor
	bob, other *Actor
	dir        string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dir := t.TempDir()
	s, err := NewStore(dir, 7, 100)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.Bootstrap("admin", "secret123", true); err != nil {
		t.Fatal(err)
	}
	f := &fixture{s: s, dir: dir}
	u, _ := s.UserByName("admin")
	f.admin = actorOf(u)
	for _, name := range []string{"ana", "bob"} {
		u, err := s.CreateUser(name, strings.ToUpper(name[:1])+name[1:], "password1", "member")
		if err != nil {
			t.Fatal(err)
		}
		if name == "ana" {
			f.ana = actorOf(u)
		} else {
			f.bob = actorOf(u)
		}
	}
	return f
}

func must[T any](t *testing.T) func(T, error) T {
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

func isErr(t *testing.T, err, want error, what string) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("%s: got %v, want %v", what, err, want)
	}
}

func setMembersEdit(t *testing.T, f *fixture, on bool) {
	st := must[Settings](t)(f.s.Settings(f.admin))
	st.MembersEditShared = on
	must[Settings](t)(f.s.SaveSettings(f.admin, st))
}

func TestBootstrapAndPasswords(t *testing.T) {
	f := newFixture(t)
	u, _ := f.s.UserByName("ADMIN") // case-insensitive
	if u.Role != "admin" || !checkPassword(u.pwHash, "secret123") || checkPassword(u.pwHash, "nope") {
		t.Fatal("first admin not set up from DS_USER/DS_PASSWORD")
	}
	// Bootstrap again does nothing.
	if err := f.s.Bootstrap("admin", "other-password", true); err != nil {
		t.Fatal(err)
	}
	u, _ = f.s.UserByName("admin")
	if !checkPassword(u.pwHash, "secret123") {
		t.Fatal("bootstrap changed an existing password")
	}
	if _, err := f.s.CreateUser("ana", "", "password1", "member"); err == nil {
		t.Fatal("duplicate username accepted")
	}
	if _, err := f.s.CreateUser("x y", "", "password1", "member"); err == nil {
		t.Fatal("bad username accepted")
	}
	if _, err := f.s.CreateUser("short", "", "123", "member"); err == nil {
		t.Fatal("short password accepted")
	}
}

func TestPermissionMatrix(t *testing.T) {
	f := newFixture(t)
	s := f.s
	// Ana's private and shared things.
	priv := must[Note](t)(s.CreateNote(f.ana, NotePatch{}))
	sh := "shared"
	shared := must[Note](t)(s.CreateNote(f.ana, NotePatch{Visibility: &sh}))
	list := must[TaskList](t)(s.CreateList(f.ana, "Chores", "shared"))
	entry := must[Entry](t)(s.CreateEntry(f.ana, EntryPatch{Visibility: &sh}))

	// Private: invisible to everyone else, admins included.
	for _, who := range []*Actor{f.bob, f.admin} {
		_, err := s.Note(who, priv.ID)
		isErr(t, err, ErrNotFound, who.Username+" reads ana's private note")
		_, err = s.UpdateNote(who, priv.ID, NotePatch{Title: &sh})
		isErr(t, err, ErrNotFound, who.Username+" edits ana's private note")
		isErr(t, s.DeleteNote(who, priv.ID), ErrNotFound, who.Username+" deletes ana's private note")
		_, err = s.SetVisibility(who, "note", priv.ID, "shared")
		isErr(t, err, ErrNotFound, who.Username+" shares ana's private note")
	}

	// Shared: everyone sees it.
	n := must[Note](t)(s.Note(f.bob, shared.ID))
	if n.CanEdit || n.CanManage || n.OwnerName != "Ana" {
		t.Fatalf("bob on ana's shared note: %+v", n.Access)
	}
	title := "Bob was here"
	_, err := s.UpdateNote(f.bob, shared.ID, NotePatch{Title: &title})
	isErr(t, err, ErrForbidden, "bob edits shared note while members can't edit")
	isErr(t, s.DeleteNote(f.bob, shared.ID), ErrForbidden, "bob deletes shared note")
	_, err = s.SetVisibility(f.bob, "note", shared.ID, "private")
	isErr(t, err, ErrForbidden, "bob unshares ana's note")

	// Bob can still pin it, just for himself.
	pin := true
	n = must[Note](t)(s.UpdateNote(f.bob, shared.ID, NotePatch{Pinned: &pin}))
	if !n.Pinned || must[Note](t)(s.Note(f.ana, shared.ID)).Pinned {
		t.Fatal("pins should be per person")
	}

	// Shared task lists are always editable by everyone signed in.
	it := must[TaskItem](t)(s.AddItem(f.bob, list.ID, "Bins out", ""))
	done := true
	it = must[TaskItem](t)(s.UpdateItem(f.bob, it.ID, ItemPatch{Done: &done}))
	if !it.Done || it.DoneByName != "Bob" {
		t.Fatalf("done by: %+v", it)
	}
	isErr(t, s.DeleteList(f.bob, list.ID), ErrForbidden, "bob deletes ana's shared list")

	// Shared entries follow the members-edit switch.
	nm := "Plumber"
	_, err = s.UpdateEntry(f.bob, entry.ID, EntryPatch{Name: &nm})
	isErr(t, err, ErrForbidden, "bob edits shared entry")
	setMembersEdit(t, f, true)
	must[Entry](t)(s.UpdateEntry(f.bob, entry.ID, EntryPatch{Name: &nm}))
	must[Note](t)(s.UpdateNote(f.bob, shared.ID, NotePatch{Title: &title}))
	isErr(t, s.DeleteEntry(f.bob, entry.ID), ErrForbidden, "members still can't delete shared things")

	// Admins manage shared things.
	must[Access](t)(s.SetVisibility(f.admin, "note", shared.ID, "private"))
	_, err = s.Note(f.bob, shared.ID)
	isErr(t, err, ErrNotFound, "unshared note still visible to bob")
	if must[Note](t)(s.Note(f.ana, shared.ID)).Title != title {
		t.Fatal("owner lost the note")
	}

	// Only pages can be public, and only admins publish.
	_, err = s.SetVisibility(f.ana, "note", priv.ID, "public")
	isErr(t, err, ErrInvalid, "public note")
}

func TestPagesVisibilityAndLayout(t *testing.T) {
	f := newFixture(t)
	s := f.s
	// Ana makes a private and a shared page; publishing needs an admin.
	pages, _, err := s.SaveLayout(f.ana, []PageIn{
		{ID: "anapriv01", Name: "Ana private", Visibility: "private"},
		{ID: "anashare1", Name: "Ana shared", Visibility: "shared", Groups: []Group{{Name: "G", Links: []Link{{Name: "x", URL: "https://x.test"}}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Ana also sees the admin's public Home page; it was left out of her list, but she can't delete it.
	if len(pages) != 3 {
		t.Fatalf("ana sees %d pages", len(pages))
	}
	_, _, err = s.SaveLayout(f.ana, []PageIn{{ID: "anapub001", Name: "Nope", Visibility: "public"}})
	isErr(t, err, ErrForbidden, "member publishes a page")

	// Visitors see public pages only.
	vis := must[[]Page](t)(s.Pages(nil))
	if len(vis) != 1 || vis[0].Name != "Home" || vis[0].Owner != "" {
		t.Fatalf("visitor pages: %+v", vis)
	}
	// Bob sees Home and Ana's shared page, not her private one.
	bp := must[[]Page](t)(s.Pages(f.bob))
	names := []string{}
	for _, p := range bp {
		names = append(names, p.Name)
	}
	if strings.Join(names, ",") != "Home,Ana shared" {
		t.Fatalf("bob pages: %v", names)
	}
	// Bob can't edit Ana's shared page, and can't delete it by leaving it out.
	edited := bp[1]
	edited.Name = "Hijacked"
	_, _, err = s.SaveLayout(f.bob, []PageIn{{ID: edited.ID, Name: edited.Name, Visibility: edited.Visibility, Groups: edited.Groups, Rev: edited.Rev}})
	isErr(t, err, ErrForbidden, "bob renames ana's page")
	must[[]Page](t)(func() ([]Page, error) { p, _, err := s.SaveLayout(f.bob, []PageIn{}); return p, err }())
	if len(must[[]Page](t)(s.Pages(f.ana))) != 3 {
		t.Fatal("bob deleted pages he doesn't manage")
	}
	// Per-person order: Bob reverses his view, Ana's stays.
	setMembersEdit(t, f, true)
	bp = must[[]Page](t)(s.Pages(f.bob))
	in := []PageIn{}
	for i := len(bp) - 1; i >= 0; i-- {
		in = append(in, PageIn{ID: bp[i].ID, Name: bp[i].Name, Visibility: bp[i].Visibility, Groups: bp[i].Groups, Rev: bp[i].Rev})
	}
	if _, _, err := s.SaveLayout(f.bob, in); err != nil {
		t.Fatal(err)
	}
	if got := must[[]Page](t)(s.Pages(f.bob)); got[0].Name != "Ana shared" {
		t.Fatalf("bob's order not kept: %s", got[0].Name)
	}
	// Conflicts: two people edit the same shared page from the same revision.
	base := must[[]Page](t)(s.Pages(f.ana))
	var page Page
	for _, p := range base {
		if p.Name == "Ana shared" {
			page = p
		}
	}
	a1 := PageIn{ID: page.ID, Name: "From Ana", Visibility: "shared", Groups: page.Groups, Rev: page.Rev}
	b1 := PageIn{ID: page.ID, Name: "From Bob", Visibility: "shared", Groups: page.Groups, Rev: page.Rev}
	if _, c, err := s.SaveLayout(f.ana, []PageIn{a1}); err != nil || len(c) != 0 {
		t.Fatal(err, c)
	}
	_, c, err := s.SaveLayout(f.bob, []PageIn{b1})
	if err != nil || len(c) != 1 {
		t.Fatalf("expected a conflict, got %v %v", c, err)
	}
}

func TestSessionsAndUsers(t *testing.T) {
	f := newFixture(t)
	s := f.s
	auth, err := NewAuth(Config{Password: "secret123", SessionDays: 30, CookieSecure: "false"}, s)
	if err != nil {
		t.Fatal(err)
	}
	login := func(user, pw string) *http.Cookie {
		body, _ := json.Marshal(map[string]any{"username": user, "password": pw})
		w := httptest.NewRecorder()
		auth.handleLogin(w, httptest.NewRequest("POST", "/api/login", bytes.NewReader(body)))
		for _, c := range w.Result().Cookies() {
			return c
		}
		return nil
	}
	who := func(c *http.Cookie) *Actor {
		r := httptest.NewRequest("GET", "/", nil)
		if c != nil {
			r.AddCookie(c)
		}
		return auth.Actor(r)
	}
	if login("ana", "wrong") != nil {
		t.Fatal("wrong password signed in")
	}
	c := login("ana", "password1")
	if a := who(c); a == nil || a.Username != "ana" {
		t.Fatal("ana not signed in")
	}
	// Signing out everywhere ends the session.
	must[User](t)(s.SignOutEverywhere(f.ana.ID))
	if who(c) != nil {
		t.Fatal("session survived sign out everywhere")
	}
	// Disabling ends sessions and blocks sign-in.
	c = login("ana", "password1")
	yes := true
	must[User](t)(s.UpdateUser(f.ana.ID, UserPatch{Disabled: &yes}))
	if who(c) != nil || login("ana", "password1") != nil {
		t.Fatal("disabled user still signed in")
	}
	// Changing a password ends other sessions.
	c = login("bob", "password1")
	must[User](t)(s.ChangeOwnPassword(f.bob, "password1", "password2"))
	if who(c) != nil || login("bob", "password2") == nil {
		t.Fatal("password change")
	}
	_, err = s.ChangeOwnPassword(f.bob, "wrong-current", "password3")
	isErr(t, err, ErrInvalid, "wrong current password")

	// The last admin can't be demoted, disabled or deleted.
	member := "member"
	_, err = s.UpdateUser(f.admin.ID, UserPatch{Role: &member})
	isErr(t, err, ErrInvalid, "demote last admin")
	_, err = s.UpdateUser(f.admin.ID, UserPatch{Disabled: &yes})
	isErr(t, err, ErrInvalid, "disable last admin")
	isErr(t, s.DeleteUser(f.admin.ID, f.admin, false), ErrInvalid, "delete yourself")

	// Deleting with handover: shared things go to the admin, private ones go away.
	sh := "shared"
	sharedNote := must[Note](t)(s.CreateNote(f.bob, NotePatch{Visibility: &sh}))
	privNote := must[Note](t)(s.CreateNote(f.bob, NotePatch{}))
	if err := s.DeleteUser(f.bob.ID, f.admin, true); err != nil {
		t.Fatal(err)
	}
	if n := must[Note](t)(s.Note(f.admin, sharedNote.ID)); n.Owner != f.admin.ID {
		t.Fatal("shared note not handed over")
	}
	var count int
	_ = s.read(func(q querier) error {
		return q.QueryRow(`SELECT count(*) FROM notes WHERE id = ?`, privNote.ID).Scan(&count)
	})
	if count != 0 {
		t.Fatal("private note of deleted user kept")
	}
	if _, err := s.UserByID(f.bob.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("user not deleted")
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	f := newFixture(t)
	s := f.s
	sh := "shared"
	must[Note](t)(s.CreateNote(f.ana, NotePatch{Title: &sh, Visibility: &sh}))
	l := must[TaskList](t)(s.CreateList(f.ana, "Groceries", "private"))
	must[TaskItem](t)(s.AddItem(f.ana, l.ID, "Milk", "2026-12-01"))
	must[Entry](t)(s.CreateEntry(f.ana, EntryPatch{Name: &sh}))
	if _, _, err := s.SaveLayout(f.ana, []PageIn{{ID: "anapage01", Name: "Mine", Visibility: "private", Groups: []Group{{Name: "G", Links: []Link{{Name: "L", URL: "https://l.test"}}}}}}); err != nil {
		t.Fatal(err)
	}
	st := must[Settings](t)(s.Settings(f.ana))
	st.StartOn = "links:anapage01"
	st.Theme = "light"
	must[Settings](t)(s.SaveSettings(f.ana, st))

	ex := must[Export](t)(s.ExportMine(f.ana))
	if len(ex.Pages) != 1 || len(ex.Notes) != 1 || len(ex.Tasks) != 1 || len(ex.Directory) != 1 {
		t.Fatalf("export: %d pages %d notes %d lists %d entries", len(ex.Pages), len(ex.Notes), len(ex.Tasks), len(ex.Directory))
	}
	raw, _ := json.Marshal(ex)
	parsed, sum, err := ParseExport(raw)
	if err != nil || sum.Links != 1 || sum.Tasks != 1 {
		t.Fatal(err, sum)
	}
	// Bob imports Ana's file into his own space; the public Home page of the admin is untouched.
	if err := s.ImportMine(f.bob, parsed); err != nil {
		t.Fatal(err)
	}
	ex2 := must[Export](t)(s.ExportMine(f.bob))
	if len(ex2.Pages) != 1 || ex2.Pages[0].Name != "Mine" || ex2.Notes[0].Visibility != "shared" || ex2.Tasks[0].Items[0].Due != "2026-12-01" {
		t.Fatalf("round trip lost data: %+v", ex2)
	}
	if ex2.Prefs.Theme != "light" || ex2.Prefs.StartOn != "links:"+ex2.Pages[0].ID {
		t.Fatalf("prefs not imported / start page not remapped: %+v", ex2.Prefs)
	}
	if len(must[[]Page](t)(s.Pages(nil))) != 1 {
		t.Fatal("import touched someone else's pages")
	}
	if _, _, err := ParseExport([]byte(`{"pages":[]}`)); err == nil {
		t.Fatal("non-v2 file accepted")
	}
}

func TestBackupRestoreAndMigrations(t *testing.T) {
	f := newFixture(t)
	s := f.s
	b := must[BackupInfo](t)(s.Backup("manual"))
	sh := "after"
	n := must[Note](t)(s.CreateNote(f.ana, NotePatch{Title: &sh}))
	if err := s.Restore(b.Name); err != nil {
		t.Fatal(err)
	}
	_, err := s.Note(f.ana, n.ID)
	isErr(t, err, ErrNotFound, "note made after the backup")
	if len(s.Backups()) < 2 {
		t.Fatal("no before-restore backup")
	}
	// A downloaded snapshot is a valid database for restore.
	var buf bytes.Buffer
	if err := s.WriteSnapshot(&buf); err != nil {
		t.Fatal(err)
	}
	if err := s.RestoreFile(bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatal(err)
	}
	if err := s.RestoreFile(strings.NewReader("not a database")); err == nil {
		t.Fatal("garbage restored")
	}
	// Reopening an up-to-date database runs no migrations and keeps data.
	s.Close()
	s2, err := NewStore(f.dir, 7, 100)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	var v int
	_ = s2.read(func(q querier) error { return q.QueryRow(`PRAGMA user_version`).Scan(&v) })
	if v != schemaVersion() || s2.UserCount() != 3 {
		t.Fatalf("schema %d users %d", v, s2.UserCount())
	}
	if _, err := os.Stat(filepath.Join(f.dir, "backups")); err != nil {
		t.Fatal(err)
	}
}

func TestVisitorAPI(t *testing.T) {
	f := newFixture(t)
	auth, _ := NewAuth(Config{Password: "secret123", SessionDays: 30, CookieSecure: "false"}, f.s)
	srv := &Server{cfg: Config{MaxNoteKB: 100, BackupKeep: 7}, store: f.s, auth: auth, assets: loadAssets()}
	h := srv.Handler()
	sh := "shared"
	must[Note](t)(f.s.CreateNote(f.ana, NotePatch{Title: &sh, Visibility: &sh}))
	get := func(path string) (int, string) {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w.Code, w.Body.String()
	}
	code, body := get("/api/state")
	if code != 200 || strings.Contains(body, `"notes"`) || strings.Contains(body, "Ana") || !strings.Contains(body, "Home") {
		t.Fatalf("visitor state leaks or is missing data: %d %s", code, body)
	}
	if _, body := get("/api/search?q=shared"); body != "[]\n" {
		t.Fatalf("visitor search: %s", body)
	}
	for _, p := range []string{"/api/notes", "/api/tasks", "/api/export", "/api/users", "/api/db", "/api/backups", "/api/about", "/api/me"} {
		if code, _ := get(p); code != http.StatusUnauthorized {
			t.Fatalf("%s for a visitor: %d", p, code)
		}
	}
}
