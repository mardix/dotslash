package main

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ---------- passwords ----------

// pbkdfIter is a variable so tests can make hashing fast.
var pbkdfIter = 600_000

const minPasswordLen = 8

func hashPassword(pw string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	k, err := pbkdf2.Key(sha256.New, pw, salt, pbkdfIter, 32)
	if err != nil {
		return "", err
	}
	enc := base64.RawStdEncoding
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", pbkdfIter, enc.EncodeToString(salt), enc.EncodeToString(k)), nil
}

func checkPassword(stored, pw string) bool {
	f := strings.Split(stored, "$")
	if len(f) != 4 || f[0] != "pbkdf2-sha256" {
		return false
	}
	iter, err := strconv.Atoi(f[1])
	if err != nil || iter < 1 {
		return false
	}
	enc := base64.RawStdEncoding
	salt, err1 := enc.DecodeString(f[2])
	want, err2 := enc.DecodeString(f[3])
	if err1 != nil || err2 != nil {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, pw, salt, iter, len(want))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}

func randomPassword() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// ---------- users ----------

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Name      string    `json:"name"`
	Role      string    `json:"role"` // admin | member
	Disabled  bool      `json:"disabled"`
	CreatedAt time.Time `json:"created_at"`
	pwHash    string
	tokenVer  int64
}

// Display is the name shown on the page: the user's name, else the username.
func (u User) Display() string {
	if strings.TrimSpace(u.Name) != "" {
		return u.Name
	}
	return u.Username
}

// Actor is the signed-in person a request acts for. A nil *Actor is a visitor.
type Actor struct {
	ID       string
	Username string
	Name     string
	Role     string
	TokenVer int64
}

func (a *Actor) Admin() bool { return a != nil && a.Role == "admin" }

func actorOf(u User) *Actor {
	return &Actor{ID: u.ID, Username: u.Username, Name: u.Display(), Role: u.Role, TokenVer: u.tokenVer}
}

const userCols = `id, username, name, role, pw_hash, token_ver, disabled, created_at`

func scanUser(sc interface{ Scan(...any) error }) (User, error) {
	var u User
	var created string
	var disabled int
	err := sc.Scan(&u.ID, &u.Username, &u.Name, &u.Role, &u.pwHash, &u.tokenVer, &disabled, &created)
	u.Disabled = disabled != 0
	u.CreatedAt = pt(created)
	return u, err
}

func getUser(q querier, where string, arg any) (User, error) {
	u, err := scanUser(q.QueryRow(`SELECT `+userCols+` FROM users WHERE `+where, arg))
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) UserByID(id string) (User, error) {
	var u User
	err := s.read(func(q querier) (err error) { u, err = getUser(q, `id = ?`, id); return })
	return u, err
}

func (s *Store) UserByName(username string) (User, error) {
	var u User
	err := s.read(func(q querier) (err error) { u, err = getUser(q, `username = ?`, strings.TrimSpace(username)); return })
	return u, err
}

func (s *Store) Users() ([]User, error) {
	out := []User{}
	err := s.read(func(q querier) error {
		rows, err := q.Query(`SELECT ` + userCols + ` FROM users ORDER BY created_at, username`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			u, err := scanUser(rows)
			if err != nil {
				return err
			}
			out = append(out, u)
		}
		return rows.Err()
	})
	return out, err
}

func (s *Store) UserCount() (n int) {
	_ = s.read(func(q querier) error { return q.QueryRow(`SELECT count(*) FROM users`).Scan(&n) })
	return
}

func activeAdmins(q querier) (n int, err error) {
	err = q.QueryRow(`SELECT count(*) FROM users WHERE role = 'admin' AND disabled = 0`).Scan(&n)
	return
}

func validateUsername(u string) (string, error) {
	u = strings.TrimSpace(u)
	if !usernameRe.MatchString(u) {
		return "", invalid("usernames are 1–40 letters, numbers, dots, dashes, underscores or @")
	}
	return u, nil
}

// insertUser creates a user and their starting preferences and scratchpad.
func insertUser(q querier, username, name, role, pwHash string) (User, error) {
	u := User{ID: newID(), Username: username, Name: clip(name, 60), Role: role, CreatedAt: now(), pwHash: pwHash, tokenVer: 1}
	_, err := q.Exec(`INSERT INTO users (id, username, name, role, pw_hash, token_ver, created_at) VALUES (?, ?, ?, ?, ?, 1, ?)`,
		u.ID, u.Username, u.Name, u.Role, u.pwHash, ts(u.CreatedAt))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return User{}, invalid("the username “%s” is taken", username)
		}
		return User{}, err
	}
	p, _ := json.Marshal(defaultPrefs())
	if _, err := q.Exec(`INSERT INTO prefs (user_id, data) VALUES (?, ?)`, u.ID, string(p)); err != nil {
		return User{}, err
	}
	_, err = q.Exec(`INSERT INTO scratchpad (user_id, content, modified_at) VALUES (?, '', ?)`, u.ID, ts(u.CreatedAt))
	return u, err
}

func (s *Store) CreateUser(username, name, password, role string) (User, error) {
	username, err := validateUsername(username)
	if err != nil {
		return User{}, err
	}
	if role != "admin" && role != "member" {
		return User{}, invalid("role must be admin or member")
	}
	if len(password) < minPasswordLen {
		return User{}, invalid("passwords need at least %d characters", minPasswordLen)
	}
	h, err := hashPassword(password)
	if err != nil {
		return User{}, err
	}
	var u User
	err = s.tx(func(q querier) (err error) { u, err = insertUser(q, username, name, role, h); return })
	return u, err
}

type UserPatch struct {
	Name     *string `json:"name"`
	Role     *string `json:"role"`
	Disabled *bool   `json:"disabled"`
	Password *string `json:"password"`
}

// UpdateUser changes a user. Disabling, demoting or resetting the password
// signs them out everywhere. The last active admin can't be demoted or disabled.
func (s *Store) UpdateUser(id string, p UserPatch) (User, error) {
	var hash string
	if p.Password != nil {
		if len(*p.Password) < minPasswordLen {
			return User{}, invalid("passwords need at least %d characters", minPasswordLen)
		}
		var err error
		if hash, err = hashPassword(*p.Password); err != nil {
			return User{}, err
		}
	}
	var out User
	err := s.tx(func(q querier) error {
		u, err := getUser(q, `id = ?`, id)
		if err != nil {
			return err
		}
		bump := false
		if p.Name != nil {
			u.Name = clip(*p.Name, 60)
		}
		if p.Role != nil && *p.Role != u.Role {
			if *p.Role != "admin" && *p.Role != "member" {
				return invalid("role must be admin or member")
			}
			u.Role = *p.Role
			bump = true
		}
		if p.Disabled != nil && *p.Disabled != u.Disabled {
			u.Disabled = *p.Disabled
			bump = bump || u.Disabled
		}
		if hash != "" {
			u.pwHash = hash
			bump = true
		}
		if bump {
			u.tokenVer++
		}
		disabled := 0
		if u.Disabled {
			disabled = 1
		}
		if _, err := q.Exec(`UPDATE users SET name = ?, role = ?, disabled = ?, pw_hash = ?, token_ver = ? WHERE id = ?`,
			u.Name, u.Role, disabled, u.pwHash, u.tokenVer, u.ID); err != nil {
			return err
		}
		if n, err := activeAdmins(q); err != nil {
			return err
		} else if n == 0 {
			return invalid("there must always be at least one active admin")
		}
		out = u
		return nil
	})
	return out, err
}

// ChangeOwnPassword checks the current password, sets a new one and signs
// out every other session (the caller re-issues this session's cookie).
func (s *Store) ChangeOwnPassword(a *Actor, current, next string) (User, error) {
	u, err := s.UserByID(a.ID)
	if err != nil {
		return User{}, err
	}
	if u.pwHash != "" && !checkPassword(u.pwHash, current) {
		return User{}, invalid("your current password is not right")
	}
	return s.UpdateUser(a.ID, UserPatch{Password: &next})
}

// SignOutEverywhere invalidates every session of the user.
func (s *Store) SignOutEverywhere(id string) (User, error) {
	var out User
	err := s.tx(func(q querier) error {
		if _, err := q.Exec(`UPDATE users SET token_ver = token_ver + 1 WHERE id = ?`, id); err != nil {
			return err
		}
		u, err := getUser(q, `id = ?`, id)
		out = u
		return err
	})
	return out, err
}

// DeleteUser removes a user and their private things. Their shared things
// go to the admin doing the delete when handover is true; otherwise they are
// deleted too.
func (s *Store) DeleteUser(id string, by *Actor, handover bool) error {
	if by != nil && by.ID == id {
		return invalid("you can't delete your own account")
	}
	return s.tx(func(q querier) error {
		if _, err := getUser(q, `id = ?`, id); err != nil {
			return err
		}
		if handover && by != nil {
			for _, t := range []string{"pages", "notes", "task_lists", "entries"} {
				if _, err := q.Exec(`UPDATE `+t+` SET owner_id = ? WHERE owner_id = ? AND visibility != 'private'`, by.ID, id); err != nil {
					return err
				}
			}
		}
		if err := deleteOrphanState(q, id); err != nil {
			return err
		}
		if _, err := q.Exec(`DELETE FROM users WHERE id = ?`, id); err != nil {
			return err
		}
		if n, err := activeAdmins(q); err != nil {
			return err
		} else if n == 0 {
			return invalid("there must always be at least one active admin")
		}
		return nil
	})
}

// deleteOrphanState removes other people's order and pins for items that are
// about to be deleted along with their owner.
func deleteOrphanState(q querier, owner string) error {
	for kind, table := range map[string]string{"page": "pages", "note": "notes", "list": "task_lists"} {
		if _, err := q.Exec(`DELETE FROM user_state WHERE kind = ? AND item_id IN (SELECT id FROM `+table+` WHERE owner_id = ?)`, kind, owner); err != nil {
			return err
		}
	}
	return nil
}

// ---------- first start ----------

// Bootstrap makes sure there is someone to act as. With sign-in on and no
// users yet, DS_USER/DS_PASSWORD become the first admin. With sign-in off,
// a single admin is created (without a password) and everyone acts as them.
func (s *Store) Bootstrap(username, password string, authOn bool) error {
	if username == "" {
		username = "admin"
	}
	if _, err := validateUsername(username); err != nil {
		return fmt.Errorf("DS_USER: %w", err)
	}
	return s.tx(func(q querier) error {
		var n int
		if err := q.QueryRow(`SELECT count(*) FROM users`).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			if authOn {
				// A dashboard started without a password gets one now.
				u, err := getUser(q, `username = ?`, username)
				if err == nil && u.pwHash == "" {
					h, err := hashPassword(password)
					if err != nil {
						return err
					}
					_, err = q.Exec(`UPDATE users SET pw_hash = ?, token_ver = token_ver + 1 WHERE id = ?`, h, u.ID)
					return err
				}
			}
			return nil
		}
		h := ""
		if authOn {
			var err error
			if h, err = hashPassword(password); err != nil {
				return err
			}
		}
		u, err := insertUser(q, username, "", "admin", h)
		if err != nil {
			return err
		}
		logf("created admin account “%s”", username)
		return seedStarter(q, u.ID)
	})
}

// OpenModeActor is who everyone acts as when sign-in is off: the first admin.
func (s *Store) OpenModeActor() (*Actor, error) {
	var u User
	err := s.read(func(q querier) (err error) {
		u, err = scanUser(q.QueryRow(`SELECT ` + userCols + ` FROM users WHERE role = 'admin' AND disabled = 0 ORDER BY created_at LIMIT 1`))
		return
	})
	if err != nil {
		return nil, err
	}
	return actorOf(u), nil
}

// ResetPassword sets a new random password (for the reset-password command).
func (s *Store) ResetPassword(username string) (string, error) {
	u, err := s.UserByName(username)
	if err != nil {
		return "", fmt.Errorf("no user named %q", username)
	}
	pw := randomPassword()
	f := false
	if _, err := s.UpdateUser(u.ID, UserPatch{Password: &pw, Disabled: &f}); err != nil {
		return "", err
	}
	return pw, nil
}

// seedStarter gives the first admin a small example of everything.
func seedStarter(q querier, owner string) error {
	t := ts(now())
	groups, _ := json.Marshal([]Group{{
		ID: newID(), Name: "Getting started",
		Links: []Link{
			{ID: newID(), Name: "DotSlash", URL: repoURL, Icon: "github", Description: "Source, issues and releases", NewTab: true},
			{ID: newID(), Name: "Singlebase", URL: "https://singlebase.io", NewTab: true},
			{ID: newID(), Name: "Dashboard Icons", URL: "https://dashboardicons.com", Icon: "dashboard-icons", Description: "Icon names you can use for links", NewTab: true},
		},
	}})
	stmts := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO pages (id, owner_id, visibility, name, groups_json, rev, created_at, modified_at) VALUES (?, ?, 'public', 'Home', ?, 1, ?, ?)`, []any{newID(), owner, string(groups), t, t}},
		{`INSERT INTO notes (id, owner_id, type, title, content, created_at, modified_at) VALUES (?, ?, 'note', 'Welcome', ?, ?, ?)`, []any{newID(), owner,
			"This is a note. Notes support **Markdown**:\n\n## Headings\n- lists\n- `inline code`\n- [links](https://example.com)\n\n```\ncode blocks\n```\n\nClick **Edit** on a note to change it. Changes save automatically.\n\nUse **Share** to let everyone who signs in see a note, a task list, a page or a directory entry.", t, t}},
		{`INSERT INTO notes (id, owner_id, type, title, content, created_at, modified_at) VALUES (?, ?, 'snippet', 'Update all containers', ?, ?, ?)`, []any{newID(), owner,
			"docker compose pull\ndocker compose up -d\ndocker image prune -f", t, t}},
	}
	for _, st := range stmts {
		if _, err := q.Exec(st.sql, st.args...); err != nil {
			return err
		}
	}
	listID := newID()
	if _, err := q.Exec(`INSERT INTO task_lists (id, owner_id, title, created_at, modified_at) VALUES (?, ?, 'Getting started', ?, ?)`, listID, owner, t, t); err != nil {
		return err
	}
	for i, txt := range []string{"Sign in and add your first link", "Pick a theme and accent in Settings", "Invite someone in Settings → Users", "Download a backup"} {
		if _, err := q.Exec(`INSERT INTO task_items (id, list_id, text, position, created_at) VALUES (?, ?, ?, ?, ?)`, newID(), listID, txt, i, t); err != nil {
			return err
		}
	}
	return nil
}
