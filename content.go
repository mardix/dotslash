package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

// ---------- permissions ----------
//
// The rules, in one place:
//   - a visitor (nil actor) sees public pages and nothing else;
//   - a signed-in person sees what they own and everything shared or public;
//   - private things are only ever visible to their owner, admins included;
//   - owners and admins manage (share, unshare, delete) shared things;
//   - shared task lists can be edited by everyone signed in; other shared
//     things by members only when the admin allows it;
//   - only admins make pages public.

func canSee(a *Actor, owner, vis string) bool {
	if a == nil {
		return vis == "public"
	}
	return owner == a.ID || vis != "private"
}

func canManage(a *Actor, owner, vis string) bool {
	if a == nil {
		return false
	}
	return owner == a.ID || (a.Admin() && vis != "private")
}

func canEdit(a *Actor, owner, vis, kind string, membersEdit bool) bool {
	if canManage(a, owner, vis) {
		return true
	}
	if a == nil || vis == "private" {
		return false
	}
	return kind == "list" || membersEdit
}

func checkVisibility(a *Actor, kind, vis string) error {
	switch vis {
	case "private", "shared":
		return nil
	case "public":
		if kind != "page" {
			return invalid("only link pages can be public")
		}
		if !a.Admin() {
			return forbidden("only admins can make a page public")
		}
		return nil
	}
	return invalid("visibility must be private, shared or public")
}

func access(a *Actor, owner, ownerName, vis, kind string, membersEdit bool) Access {
	return Access{Visibility: vis, Owner: owner, OwnerName: ownerName,
		CanEdit: canEdit(a, owner, vis, kind, membersEdit), CanManage: canManage(a, owner, vis)}
}

const ownerName = `COALESCE(NULLIF(u.name, ''), u.username, '')`

var kindTables = map[string]string{"page": "pages", "note": "notes", "list": "task_lists", "entry": "entries"}

// ownerVis loads who owns an item and how it is shared.
func ownerVis(q querier, kind, id string) (owner, vis string, err error) {
	t, ok := kindTables[kind]
	if !ok {
		return "", "", ErrNotFound
	}
	err = q.QueryRow(`SELECT owner_id, visibility FROM `+t+` WHERE id = ?`, id).Scan(&owner, &vis)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return
}

// mayEdit checks that a can see and edit the item.
func mayEdit(q querier, a *Actor, kind, id string, membersEdit bool) error {
	owner, vis, err := ownerVis(q, kind, id)
	if err != nil {
		return err
	}
	if !canSee(a, owner, vis) {
		return ErrNotFound
	}
	if !canEdit(a, owner, vis, kind, membersEdit) {
		return forbidden("this is shared read-only; ask its owner or an admin")
	}
	return nil
}

func mayManage(q querier, a *Actor, kind, id string) (vis string, err error) {
	owner, vis, err := ownerVis(q, kind, id)
	if err != nil {
		return "", err
	}
	if !canSee(a, owner, vis) {
		return "", ErrNotFound
	}
	if !canManage(a, owner, vis) {
		return "", forbidden("only its owner or an admin can do that")
	}
	return vis, nil
}

// setPositions stores a's own order for items of a kind.
func setPositions(q querier, uid, kind string, ids []string) error {
	for i, id := range ids {
		if _, err := q.Exec(`INSERT INTO user_state (user_id, kind, item_id, position) VALUES (?, ?, ?, ?)
			ON CONFLICT(user_id, kind, item_id) DO UPDATE SET position = excluded.position`, uid, kind, id, i); err != nil {
			return err
		}
	}
	return nil
}

func forgetItem(q querier, kind, id string) error {
	_, err := q.Exec(`DELETE FROM user_state WHERE kind = ? AND item_id = ?`, kind, id)
	return err
}

// SetVisibility shares, unshares or publishes an item.
func (s *Store) SetVisibility(a *Actor, kind, id, vis string) (Access, error) {
	var out Access
	err := s.tx(func(q querier) error {
		if _, err := mayManage(q, a, kind, id); err != nil {
			return err
		}
		if err := checkVisibility(a, kind, vis); err != nil {
			return err
		}
		t := kindTables[kind]
		extra := ""
		if kind == "page" {
			extra = ", rev = rev + 1"
		}
		if _, err := q.Exec(`UPDATE `+t+` SET visibility = ?`+extra+` WHERE id = ?`, vis, id); err != nil {
			return err
		}
		in, err := loadInstance(q)
		if err != nil {
			return err
		}
		owner, _, err := ownerVis(q, kind, id)
		if err != nil {
			return err
		}
		var name string
		_ = q.QueryRow(`SELECT `+ownerName+` FROM users u WHERE u.id = ?`, owner).Scan(&name)
		out = access(a, owner, name, vis, kind, in.MembersEditShared)
		return nil
	})
	return out, err
}

// ---------- instance & prefs ----------

func loadInstance(q querier) (Instance, error) {
	var raw string
	in := defaultInstance(appName)
	if err := q.QueryRow(`SELECT data FROM instance WHERE id = 1`).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return in, nil
		}
		return in, err
	}
	_ = json.Unmarshal([]byte(raw), &in)
	normalizeInstance(&in)
	return in, nil
}

func loadPrefs(q querier, uid string) (Prefs, error) {
	var raw string
	p := defaultPrefs()
	err := q.QueryRow(`SELECT data FROM prefs WHERE user_id = ?`, uid).Scan(&raw)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return p, err
	}
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &p)
	}
	normalizePrefs(&p)
	return p, nil
}

func (s *Store) Instance() Instance {
	in := defaultInstance(appName)
	_ = s.read(func(q querier) (err error) { in, err = loadInstance(q); return })
	return in
}

// publicStart picks where a visitor lands.
func publicStart(in Instance, pub []Page) string {
	if in.PublicStart == "blank" {
		return "blank"
	}
	if strings.HasPrefix(in.PublicStart, "links:") {
		id := strings.TrimPrefix(in.PublicStart, "links:")
		for _, p := range pub {
			if p.ID == id {
				return in.PublicStart
			}
		}
	}
	if len(pub) > 0 {
		return "links:" + pub[0].ID
	}
	return "blank"
}

// Settings returns the merged settings for a (a visitor gets the defaults).
func (s *Store) Settings(a *Actor) (Settings, error) {
	var out Settings
	err := s.read(func(q querier) error {
		in, err := loadInstance(q)
		if err != nil {
			return err
		}
		if a == nil {
			pub, err := pagesFor(q, nil, false)
			if err != nil {
				return err
			}
			p := defaultPrefs()
			p.StartOn = publicStart(in, pub)
			out = mergeSettings(in, p)
			out.MembersEditShared = false
			return nil
		}
		p, err := loadPrefs(q, a.ID)
		if err != nil {
			return err
		}
		out = mergeSettings(in, p)
		return nil
	})
	return out, err
}

// SaveSettings stores a's preferences, and the instance settings when a is an admin.
func (s *Store) SaveSettings(a *Actor, st Settings) (Settings, error) {
	in, p := st.split()
	normalizePrefs(&p)
	err := s.tx(func(q querier) error {
		b, _ := json.Marshal(p)
		if _, err := q.Exec(`INSERT INTO prefs (user_id, data) VALUES (?, ?) ON CONFLICT(user_id) DO UPDATE SET data = excluded.data`, a.ID, string(b)); err != nil {
			return err
		}
		if a.Admin() {
			normalizeInstance(&in)
			b, _ := json.Marshal(in)
			if _, err := q.Exec(`UPDATE instance SET data = ? WHERE id = 1`, string(b)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Settings{}, err
	}
	return s.Settings(a)
}

// ---------- pages ----------

func pagesFor(q querier, a *Actor, membersEdit bool) ([]Page, error) {
	var rows *sql.Rows
	var err error
	cols := `SELECT p.id, p.name, p.groups_json, p.rev, p.visibility, p.owner_id, ` + ownerName + ` FROM pages p LEFT JOIN users u ON u.id = p.owner_id `
	if a == nil {
		// Visitors see public pages in their owners' order.
		rows, err = q.Query(cols + `LEFT JOIN user_state us ON us.user_id = p.owner_id AND us.kind = 'page' AND us.item_id = p.id
			WHERE p.visibility = 'public' ORDER BY us.position IS NULL, us.position, p.created_at`)
	} else {
		rows, err = q.Query(cols+`LEFT JOIN user_state us ON us.user_id = ? AND us.kind = 'page' AND us.item_id = p.id
			WHERE p.owner_id = ? OR p.visibility != 'private' ORDER BY us.position IS NULL, us.position, p.created_at`, a.ID, a.ID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Page{}
	for rows.Next() {
		var p Page
		var groups, owner, vis, name string
		if err := rows.Scan(&p.ID, &p.Name, &groups, &p.Rev, &vis, &owner, &name); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(groups), &p.Groups)
		if p.Groups == nil {
			p.Groups = []Group{}
		}
		p.Access = access(a, owner, name, vis, "page", membersEdit)
		if a == nil {
			p.Owner, p.OwnerName = "", ""
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) Pages(a *Actor) ([]Page, error) {
	var out []Page
	err := s.read(func(q querier) error {
		in, err := loadInstance(q)
		if err != nil {
			return err
		}
		out, err = pagesFor(q, a, in.MembersEditShared)
		return err
	})
	return out, err
}

// PageIn is one page as the editor sends it back.
type PageIn struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Visibility string  `json:"visibility"`
	Groups     []Group `json:"groups"`
	Rev        int64   `json:"rev"`
}

// SaveLayout applies the editor's version of every page a can see, in order.
// New pages are created as a's own; changed pages are saved if a may edit
// them and nobody changed them in the meantime (otherwise they are reported
// as conflicts and left alone); pages left out are deleted when a manages
// them; the order is saved as a's own.
func (s *Store) SaveLayout(a *Actor, in []PageIn) (pages []Page, conflicts []string, err error) {
	conflicts = []string{}
	err = s.tx(func(q querier) error {
		inst, err := loadInstance(q)
		if err != nil {
			return err
		}
		cur, err := pagesFor(q, a, inst.MembersEditShared)
		if err != nil {
			return err
		}
		byID := map[string]Page{}
		for _, p := range cur {
			byID[p.ID] = p
		}
		kept := map[string]bool{}
		order := []string{}
		t := ts(now())
		for _, p := range in {
			name := clip(p.Name, 60)
			if name == "" {
				name = "Untitled"
			}
			groups := normalizeGroups(p.Groups)
			gj, _ := json.Marshal(groups)
			old, visible := byID[p.ID]
			if !visible {
				if idRe.MatchString(p.ID) {
					if _, _, err := ownerVis(q, "page", p.ID); err == nil {
						continue // someone else's page this user can't see
					}
				} else {
					p.ID = newID()
				}
				vis := p.Visibility
				if vis == "" {
					vis = "private"
				}
				if err := checkVisibility(a, "page", vis); err != nil {
					return err
				}
				if _, err := q.Exec(`INSERT INTO pages (id, owner_id, visibility, name, groups_json, rev, created_at, modified_at) VALUES (?, ?, ?, ?, ?, 1, ?, ?)`,
					p.ID, a.ID, vis, name, string(gj), t, t); err != nil {
					return err
				}
				kept[p.ID] = true
				order = append(order, p.ID)
				continue
			}
			kept[p.ID] = true
			order = append(order, p.ID)
			oldGroups, _ := json.Marshal(normalizeGroups(old.Groups))
			contentChanged := old.Name != name || string(oldGroups) != string(gj)
			vis := p.Visibility
			if vis == "" {
				vis = old.Visibility
			}
			visChanged := vis != old.Visibility
			if !contentChanged && !visChanged {
				continue
			}
			if contentChanged && !old.CanEdit {
				return forbidden("“%s” is shared read-only; ask its owner or an admin", old.Name)
			}
			if visChanged {
				if !old.CanManage {
					return forbidden("only the owner of “%s” or an admin can change who sees it", old.Name)
				}
				if err := checkVisibility(a, "page", vis); err != nil {
					return err
				}
			}
			if p.Rev != old.Rev {
				conflicts = append(conflicts, old.Name)
				continue
			}
			if _, err := q.Exec(`UPDATE pages SET name = ?, groups_json = ?, visibility = ?, rev = rev + 1, modified_at = ? WHERE id = ?`,
				name, string(gj), vis, t, p.ID); err != nil {
				return err
			}
		}
		for _, p := range cur {
			if kept[p.ID] || !p.CanManage {
				continue
			}
			if _, err := q.Exec(`DELETE FROM pages WHERE id = ?`, p.ID); err != nil {
				return err
			}
			if err := forgetItem(q, "page", p.ID); err != nil {
				return err
			}
		}
		if err := setPositions(q, a.ID, "page", order); err != nil {
			return err
		}
		pages, err = pagesFor(q, a, inst.MembersEditShared)
		return err
	})
	return
}

// ---------- notes ----------

type NotePatch struct {
	Type         *string `json:"type"`
	Title        *string `json:"title"`
	Content      *string `json:"content"`
	Pinned       *bool   `json:"pinned"`
	Visibility   *string `json:"visibility"` // on create only; use SetVisibility after
	BaseModified string  `json:"base_modified"`
}

func notesFor(q querier, a *Actor, membersEdit bool, id string) ([]Note, error) {
	where := `(n.owner_id = ? OR n.visibility != 'private')`
	args := []any{a.ID, a.ID}
	if id != "" {
		where += ` AND n.id = ?`
		args = append(args, id)
	}
	rows, err := q.Query(`SELECT n.id, n.type, n.title, n.content, n.created_at, n.modified_at, n.visibility, n.owner_id, `+ownerName+`, COALESCE(us.pinned, 0)
		FROM notes n LEFT JOIN users u ON u.id = n.owner_id
		LEFT JOIN user_state us ON us.user_id = ? AND us.kind = 'note' AND us.item_id = n.id
		WHERE `+where+` ORDER BY n.modified_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Note{}
	for rows.Next() {
		var n Note
		var created, modified, vis, owner, name string
		var pinned int
		if err := rows.Scan(&n.ID, &n.Type, &n.Title, &n.Content, &created, &modified, &vis, &owner, &name, &pinned); err != nil {
			return nil, err
		}
		n.CreatedAt, n.ModifiedAt, n.Pinned = pt(created), pt(modified), pinned != 0
		n.Access = access(a, owner, name, vis, "note", membersEdit)
		out = append(out, n)
	}
	return out, rows.Err()
}

func noteMeta(n Note) NoteMeta {
	m := NoteMeta{ID: n.ID, Type: n.Type, Title: n.Title, Pinned: n.Pinned, Snippet: snippetOf(n.Content, 140), CreatedAt: n.CreatedAt, ModifiedAt: n.ModifiedAt, Access: n.Access}
	if n.Type == "snippet" {
		m.Snippet = firstLine(n.Content, 140)
		m.Content = n.Content
	}
	return m
}

func (s *Store) NoteMetas(a *Actor) ([]NoteMeta, error) {
	out := []NoteMeta{}
	err := s.read(func(q querier) error {
		in, err := loadInstance(q)
		if err != nil {
			return err
		}
		notes, err := notesFor(q, a, in.MembersEditShared, "")
		for _, n := range notes {
			out = append(out, noteMeta(n))
		}
		return err
	})
	return out, err
}

func getNote(q querier, a *Actor, id string) (Note, error) {
	in, err := loadInstance(q)
	if err != nil {
		return Note{}, err
	}
	list, err := notesFor(q, a, in.MembersEditShared, id)
	if err != nil {
		return Note{}, err
	}
	if len(list) == 0 {
		return Note{}, ErrNotFound
	}
	return list[0], nil
}

func (s *Store) Note(a *Actor, id string) (Note, error) {
	var n Note
	err := s.read(func(q querier) (err error) { n, err = getNote(q, a, id); return })
	return n, err
}

func (s *Store) CreateNote(a *Actor, p NotePatch) (Note, error) {
	t := ts(now())
	id := newID()
	typ, title, content, vis := "note", "Untitled", "", "private"
	if p.Type != nil {
		typ = noteType(*p.Type)
	}
	if p.Title != nil && clip(*p.Title, 200) != "" {
		title = clip(*p.Title, 200)
	}
	if p.Content != nil {
		if len(*p.Content) > s.maxNoteLen {
			return Note{}, invalid("note is larger than %d KB", s.maxNoteLen/1024)
		}
		content = *p.Content
	}
	if p.Visibility != nil {
		if err := checkVisibility(a, "note", *p.Visibility); err != nil {
			return Note{}, err
		}
		vis = *p.Visibility
	}
	var out Note
	err := s.tx(func(q querier) error {
		if _, err := q.Exec(`INSERT INTO notes (id, owner_id, visibility, type, title, content, created_at, modified_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			id, a.ID, vis, typ, title, content, t, t); err != nil {
			return err
		}
		if p.Pinned != nil && *p.Pinned {
			if err := setPinned(q, a.ID, id, true); err != nil {
				return err
			}
		}
		var err error
		out, err = getNote(q, a, id)
		return err
	})
	return out, err
}

func setPinned(q querier, uid, id string, pinned bool) error {
	v := 0
	if pinned {
		v = 1
	}
	_, err := q.Exec(`INSERT INTO user_state (user_id, kind, item_id, pinned) VALUES (?, 'note', ?, ?)
		ON CONFLICT(user_id, kind, item_id) DO UPDATE SET pinned = excluded.pinned`, uid, id, v)
	return err
}

// UpdateNote edits a note. Pinning is personal, so anyone who can see a note
// can pin it. When BaseModified doesn't match, ErrConflict is returned with
// the current note.
func (s *Store) UpdateNote(a *Actor, id string, p NotePatch) (Note, error) {
	var out Note
	err := s.tx(func(q querier) error {
		cur, err := getNote(q, a, id)
		if err != nil {
			return err
		}
		if p.Pinned != nil {
			if err := setPinned(q, a.ID, id, *p.Pinned); err != nil {
				return err
			}
		}
		if p.Title != nil || p.Content != nil || p.Type != nil {
			if !cur.CanEdit {
				return forbidden("this note is shared read-only")
			}
			var modified string
			_ = q.QueryRow(`SELECT modified_at FROM notes WHERE id = ?`, id).Scan(&modified)
			if p.BaseModified != "" && !sameTime(p.BaseModified, modified) {
				out = cur
				return ErrConflict
			}
			title, content, typ := cur.Title, cur.Content, cur.Type
			if p.Title != nil {
				if title = clip(*p.Title, 200); title == "" {
					title = "Untitled"
				}
			}
			if p.Content != nil {
				if len(*p.Content) > s.maxNoteLen {
					return invalid("note is larger than %d KB", s.maxNoteLen/1024)
				}
				content = *p.Content
			}
			if p.Type != nil {
				typ = noteType(*p.Type)
			}
			if _, err := q.Exec(`UPDATE notes SET title = ?, content = ?, type = ?, modified_at = ? WHERE id = ?`, title, content, typ, ts(now()), id); err != nil {
				return err
			}
		}
		out, err = getNote(q, a, id)
		return err
	})
	if errors.Is(err, ErrConflict) {
		return out, ErrConflict
	}
	return out, err
}

func (s *Store) DeleteNote(a *Actor, id string) error {
	return s.tx(func(q querier) error {
		if _, err := mayManage(q, a, "note", id); err != nil {
			return err
		}
		if _, err := q.Exec(`DELETE FROM notes WHERE id = ?`, id); err != nil {
			return err
		}
		return forgetItem(q, "note", id)
	})
}

// ---------- tasks ----------

func listsFor(q querier, a *Actor, membersEdit bool) ([]TaskList, error) {
	rows, err := q.Query(`SELECT t.id, t.title, t.created_at, t.modified_at, t.visibility, t.owner_id, `+ownerName+`
		FROM task_lists t LEFT JOIN users u ON u.id = t.owner_id
		LEFT JOIN user_state us ON us.user_id = ? AND us.kind = 'list' AND us.item_id = t.id
		WHERE t.owner_id = ? OR t.visibility != 'private'
		ORDER BY us.position IS NULL, us.position, t.created_at`, a.ID, a.ID)
	if err != nil {
		return nil, err
	}
	out := []TaskList{}
	idx := map[string]int{}
	for rows.Next() {
		var l TaskList
		var created, modified, vis, owner, name string
		if err := rows.Scan(&l.ID, &l.Title, &created, &modified, &vis, &owner, &name); err != nil {
			rows.Close()
			return nil, err
		}
		l.CreatedAt, l.ModifiedAt = pt(created), pt(modified)
		l.Items = []TaskItem{}
		l.Access = access(a, owner, name, vis, "list", membersEdit)
		idx[l.ID] = len(out)
		out = append(out, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	items, err := q.Query(`SELECT i.id, i.list_id, i.text, i.done, i.due, i.created_at, i.done_at, COALESCE(NULLIF(u.name, ''), u.username, '')
		FROM task_items i LEFT JOIN users u ON u.id = i.done_by
		WHERE i.list_id IN (SELECT id FROM task_lists WHERE owner_id = ? OR visibility != 'private')
		ORDER BY i.position, i.created_at`, a.ID)
	if err != nil {
		return nil, err
	}
	defer items.Close()
	for items.Next() {
		var it TaskItem
		var list, created, doneAt string
		var done int
		if err := items.Scan(&it.ID, &list, &it.Text, &done, &it.Due, &created, &doneAt, &it.DoneByName); err != nil {
			return nil, err
		}
		it.Done, it.CreatedAt, it.DoneAt = done != 0, pt(created), ptp(doneAt)
		if !it.Done {
			it.DoneByName = ""
		}
		if i, ok := idx[list]; ok {
			out[i].Items = append(out[i].Items, it)
		}
	}
	return out, items.Err()
}

func (s *Store) TaskLists(a *Actor) ([]TaskList, error) {
	var out []TaskList
	err := s.read(func(q querier) error {
		in, err := loadInstance(q)
		if err != nil {
			return err
		}
		out, err = listsFor(q, a, in.MembersEditShared)
		return err
	})
	return out, err
}

func getList(q querier, a *Actor, id string) (TaskList, error) {
	in, err := loadInstance(q)
	if err != nil {
		return TaskList{}, err
	}
	lists, err := listsFor(q, a, in.MembersEditShared)
	if err != nil {
		return TaskList{}, err
	}
	for _, l := range lists {
		if l.ID == id {
			return l, nil
		}
	}
	return TaskList{}, ErrNotFound
}

func (s *Store) CreateList(a *Actor, title, vis string) (TaskList, error) {
	title = clip(title, 120)
	if title == "" {
		title = "New list"
	}
	if vis == "" {
		vis = "private"
	}
	if err := checkVisibility(a, "list", vis); err != nil {
		return TaskList{}, err
	}
	var out TaskList
	err := s.tx(func(q querier) error {
		id, t := newID(), ts(now())
		if _, err := q.Exec(`INSERT INTO task_lists (id, owner_id, visibility, title, created_at, modified_at) VALUES (?, ?, ?, ?, ?, ?)`, id, a.ID, vis, title, t, t); err != nil {
			return err
		}
		var err error
		out, err = getList(q, a, id)
		return err
	})
	return out, err
}

func membersEdit(q querier) bool {
	in, _ := loadInstance(q)
	return in.MembersEditShared
}

func (s *Store) RenameList(a *Actor, id, title string) (TaskList, error) {
	title = clip(title, 120)
	if title == "" {
		return TaskList{}, invalid("title is required")
	}
	var out TaskList
	err := s.tx(func(q querier) error {
		if err := mayEdit(q, a, "list", id, membersEdit(q)); err != nil {
			return err
		}
		if _, err := q.Exec(`UPDATE task_lists SET title = ?, modified_at = ? WHERE id = ?`, title, ts(now()), id); err != nil {
			return err
		}
		var err error
		out, err = getList(q, a, id)
		return err
	})
	return out, err
}

func (s *Store) DeleteList(a *Actor, id string) error {
	return s.tx(func(q querier) error {
		if _, err := mayManage(q, a, "list", id); err != nil {
			return err
		}
		if _, err := q.Exec(`DELETE FROM task_lists WHERE id = ?`, id); err != nil {
			return err
		}
		return forgetItem(q, "list", id)
	})
}

// OrderLists stores a's own order of task lists.
func (s *Store) OrderLists(a *Actor, ids []string) ([]TaskList, error) {
	var out []TaskList
	err := s.tx(func(q querier) error {
		if err := setPositions(q, a.ID, "list", ids); err != nil {
			return err
		}
		var err error
		out, err = listsFor(q, a, membersEdit(q))
		return err
	})
	return out, err
}

type ItemPatch struct {
	Text  *string `json:"text"`
	Done  *bool   `json:"done"`
	Due   *string `json:"due"`
	Index *int    `json:"index"` // move within the list
}

func getItem(q querier, a *Actor, id string) (TaskItem, string, error) {
	var list string
	if err := q.QueryRow(`SELECT list_id FROM task_items WHERE id = ?`, id).Scan(&list); err != nil {
		return TaskItem{}, "", ErrNotFound
	}
	l, err := getList(q, a, list)
	if err != nil {
		return TaskItem{}, "", err
	}
	for _, it := range l.Items {
		if it.ID == id {
			return it, list, nil
		}
	}
	return TaskItem{}, "", ErrNotFound
}

func (s *Store) AddItem(a *Actor, listID, text, due string) (TaskItem, error) {
	text = clip(text, 500)
	if text == "" {
		return TaskItem{}, invalid("text is required")
	}
	if due != "" && !dateRe.MatchString(due) {
		return TaskItem{}, invalid("due must be YYYY-MM-DD")
	}
	var out TaskItem
	err := s.tx(func(q querier) error {
		if err := mayEdit(q, a, "list", listID, membersEdit(q)); err != nil {
			return err
		}
		var pos int
		_ = q.QueryRow(`SELECT COALESCE(MAX(position), -1) + 1 FROM task_items WHERE list_id = ?`, listID).Scan(&pos)
		id, t := newID(), ts(now())
		if _, err := q.Exec(`INSERT INTO task_items (id, list_id, text, due, position, created_at) VALUES (?, ?, ?, ?, ?, ?)`, id, listID, text, due, pos, t); err != nil {
			return err
		}
		if _, err := q.Exec(`UPDATE task_lists SET modified_at = ? WHERE id = ?`, t, listID); err != nil {
			return err
		}
		var err error
		out, _, err = getItem(q, a, id)
		return err
	})
	return out, err
}

func (s *Store) UpdateItem(a *Actor, id string, p ItemPatch) (TaskItem, error) {
	var out TaskItem
	err := s.tx(func(q querier) error {
		_, list, err := getItem(q, a, id)
		if err != nil {
			return err
		}
		if err := mayEdit(q, a, "list", list, membersEdit(q)); err != nil {
			return err
		}
		t := ts(now())
		if p.Text != nil {
			txt := clip(*p.Text, 500)
			if txt == "" {
				return invalid("text is required")
			}
			if _, err := q.Exec(`UPDATE task_items SET text = ? WHERE id = ?`, txt, id); err != nil {
				return err
			}
		}
		if p.Due != nil {
			if *p.Due != "" && !dateRe.MatchString(*p.Due) {
				return invalid("due must be YYYY-MM-DD")
			}
			if _, err := q.Exec(`UPDATE task_items SET due = ? WHERE id = ?`, *p.Due, id); err != nil {
				return err
			}
		}
		if p.Done != nil {
			var err error
			if *p.Done {
				_, err = q.Exec(`UPDATE task_items SET done = 1, done_at = ?, done_by = ? WHERE id = ? AND done = 0`, t, a.ID, id)
			} else {
				_, err = q.Exec(`UPDATE task_items SET done = 0, done_at = '', done_by = '' WHERE id = ?`, id)
			}
			if err != nil {
				return err
			}
		}
		if p.Index != nil {
			if err := moveItem(q, list, id, *p.Index); err != nil {
				return err
			}
		}
		if _, err := q.Exec(`UPDATE task_lists SET modified_at = ? WHERE id = ?`, t, list); err != nil {
			return err
		}
		out, _, err = getItem(q, a, id)
		return err
	})
	return out, err
}

func moveItem(q querier, list, id string, to int) error {
	rows, err := q.Query(`SELECT id FROM task_items WHERE list_id = ? ORDER BY position, created_at`, list)
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var x string
		if err := rows.Scan(&x); err != nil {
			rows.Close()
			return err
		}
		if x != id {
			ids = append(ids, x)
		}
	}
	rows.Close()
	to = max(0, min(to, len(ids)))
	ids = append(ids[:to], append([]string{id}, ids[to:]...)...)
	for i, x := range ids {
		if _, err := q.Exec(`UPDATE task_items SET position = ? WHERE id = ?`, i, x); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) DeleteItem(a *Actor, id string) error {
	return s.tx(func(q querier) error {
		_, list, err := getItem(q, a, id)
		if err != nil {
			return err
		}
		if err := mayEdit(q, a, "list", list, membersEdit(q)); err != nil {
			return err
		}
		_, err = q.Exec(`DELETE FROM task_items WHERE id = ?`, id)
		return err
	})
}

// ---------- directory ----------

type EntryPatch struct {
	Name       *string `json:"name"`
	Category   *string `json:"category"`
	Phone      *string `json:"phone"`
	Email      *string `json:"email"`
	Address    *string `json:"address"`
	URL        *string `json:"url"`
	IP         *string `json:"ip"`
	MAC        *string `json:"mac"`
	Notes      *string `json:"notes"`
	Visibility *string `json:"visibility"` // on create only
}

func applyEntry(e *Entry, p EntryPatch) {
	set := func(dst *string, v *string) {
		if v != nil {
			*dst = *v
		}
	}
	set(&e.Name, p.Name)
	set(&e.Category, p.Category)
	set(&e.Phone, p.Phone)
	set(&e.Email, p.Email)
	set(&e.Address, p.Address)
	set(&e.URL, p.URL)
	set(&e.IP, p.IP)
	set(&e.MAC, p.MAC)
	set(&e.Notes, p.Notes)
	normalizeEntry(e)
}

func entriesFor(q querier, a *Actor, membersEdit bool, id string) ([]Entry, error) {
	where := `(e.owner_id = ? OR e.visibility != 'private')`
	args := []any{a.ID}
	if id != "" {
		where += ` AND e.id = ?`
		args = append(args, id)
	}
	rows, err := q.Query(`SELECT e.id, e.name, e.category, e.phone, e.email, e.address, e.url, e.ip, e.mac, e.notes, e.created_at, e.modified_at, e.visibility, e.owner_id, `+ownerName+`
		FROM entries e LEFT JOIN users u ON u.id = e.owner_id WHERE `+where+` ORDER BY e.created_at`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Entry{}
	for rows.Next() {
		var e Entry
		var created, modified, vis, owner, name string
		if err := rows.Scan(&e.ID, &e.Name, &e.Category, &e.Phone, &e.Email, &e.Address, &e.URL, &e.IP, &e.MAC, &e.Notes, &created, &modified, &vis, &owner, &name); err != nil {
			return nil, err
		}
		e.CreatedAt, e.ModifiedAt = pt(created), pt(modified)
		e.Access = access(a, owner, name, vis, "entry", membersEdit)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) Entries(a *Actor) ([]Entry, error) {
	var out []Entry
	err := s.read(func(q querier) (err error) { out, err = entriesFor(q, a, membersEdit(q), ""); return })
	return out, err
}

func getEntry(q querier, a *Actor, id string) (Entry, error) {
	list, err := entriesFor(q, a, membersEdit(q), id)
	if err != nil {
		return Entry{}, err
	}
	if len(list) == 0 {
		return Entry{}, ErrNotFound
	}
	return list[0], nil
}

func (s *Store) CreateEntry(a *Actor, p EntryPatch) (Entry, error) {
	vis := "private"
	if p.Visibility != nil {
		if err := checkVisibility(a, "entry", *p.Visibility); err != nil {
			return Entry{}, err
		}
		vis = *p.Visibility
	}
	var e Entry
	applyEntry(&e, p)
	var out Entry
	err := s.tx(func(q querier) error {
		id, t := newID(), ts(now())
		if _, err := q.Exec(`INSERT INTO entries (id, owner_id, visibility, name, category, phone, email, address, url, ip, mac, notes, created_at, modified_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, a.ID, vis, e.Name, e.Category, e.Phone, e.Email, e.Address, e.URL, e.IP, e.MAC, e.Notes, t, t); err != nil {
			return err
		}
		var err error
		out, err = getEntry(q, a, id)
		return err
	})
	return out, err
}

func (s *Store) UpdateEntry(a *Actor, id string, p EntryPatch) (Entry, error) {
	var out Entry
	err := s.tx(func(q querier) error {
		e, err := getEntry(q, a, id)
		if err != nil {
			return err
		}
		if !e.CanEdit {
			return forbidden("this entry is shared read-only")
		}
		applyEntry(&e, p)
		if _, err := q.Exec(`UPDATE entries SET name = ?, category = ?, phone = ?, email = ?, address = ?, url = ?, ip = ?, mac = ?, notes = ?, modified_at = ? WHERE id = ?`,
			e.Name, e.Category, e.Phone, e.Email, e.Address, e.URL, e.IP, e.MAC, e.Notes, ts(now()), id); err != nil {
			return err
		}
		out, err = getEntry(q, a, id)
		return err
	})
	return out, err
}

func (s *Store) DeleteEntry(a *Actor, id string) error {
	return s.tx(func(q querier) error {
		if _, err := mayManage(q, a, "entry", id); err != nil {
			return err
		}
		_, err := q.Exec(`DELETE FROM entries WHERE id = ?`, id)
		return err
	})
}

// ---------- scratchpad ----------

const maxScratchpadLen = 64 * 1024

func getPad(q querier, uid string) (Scratchpad, error) {
	var p Scratchpad
	var modified string
	err := q.QueryRow(`SELECT content, modified_at FROM scratchpad WHERE user_id = ?`, uid).Scan(&p.Content, &modified)
	if errors.Is(err, sql.ErrNoRows) {
		return Scratchpad{}, nil
	}
	p.ModifiedAt = ptp(modified)
	return p, err
}

func (s *Store) Scratchpad(a *Actor) (Scratchpad, error) {
	var p Scratchpad
	err := s.read(func(q querier) (err error) { p, err = getPad(q, a.ID); return })
	return p, err
}

// SaveScratchpad replaces a's scratchpad. If baseModified is set and does not
// match, ErrConflict is returned along with the current pad.
func (s *Store) SaveScratchpad(a *Actor, content, baseModified string) (Scratchpad, error) {
	if len(content) > maxScratchpadLen {
		return Scratchpad{}, invalid("the scratchpad is limited to %d KB", maxScratchpadLen/1024)
	}
	var out Scratchpad
	err := s.tx(func(q querier) error {
		cur, err := getPad(q, a.ID)
		if err != nil {
			return err
		}
		if baseModified != "" && cur.ModifiedAt != nil && !sameTime(baseModified, ts(*cur.ModifiedAt)) {
			out = cur
			return ErrConflict
		}
		t := now()
		if _, err := q.Exec(`INSERT INTO scratchpad (user_id, content, modified_at) VALUES (?, ?, ?)
			ON CONFLICT(user_id) DO UPDATE SET content = excluded.content, modified_at = excluded.modified_at`, a.ID, content, ts(t)); err != nil {
			return err
		}
		out = Scratchpad{Content: content, ModifiedAt: &t}
		return nil
	})
	return out, err
}

// ---------- search ----------

type SearchResult struct {
	Kind    string `json:"kind"` // link | note | snippet | task | entry
	ID      string `json:"id"`
	Title   string `json:"title"`
	Context string `json:"context"`
	URL     string `json:"url,omitempty"`
	Icon    string `json:"icon,omitempty"`
	NewTab  bool   `json:"new_tab,omitempty"`
	PageID  string `json:"page_id,omitempty"`
	ListID  string `json:"list_id,omitempty"`
	Done    bool   `json:"done,omitempty"`
	Content string `json:"content,omitempty"`
	Shared  bool   `json:"shared,omitempty"`
}

// Search looks through what a can see: links (public pages only for
// visitors) and, for signed-in people, notes, tasks and directory entries in
// the sections they have turned on.
func (s *Store) Search(a *Actor, qs string, limit int) ([]SearchResult, error) {
	qs = strings.ToLower(strings.TrimSpace(qs))
	out := []SearchResult{}
	if qs == "" {
		return out, nil
	}
	terms := strings.Fields(qs)
	match := func(fields ...string) bool {
		hay := strings.ToLower(strings.Join(fields, " \u0000 "))
		for _, t := range terms {
			if !strings.Contains(hay, t) {
				return false
			}
		}
		return true
	}
	err := s.read(func(q querier) error {
		in, err := loadInstance(q)
		if err != nil {
			return err
		}
		pages, err := pagesFor(q, a, in.MembersEditShared)
		if err != nil {
			return err
		}
		for _, p := range pages {
			for _, g := range p.Groups {
				for _, l := range g.Links {
					if match(l.Name, l.URL, l.Description, g.Name, p.Name) {
						out = append(out, SearchResult{Kind: "link", ID: l.ID, Title: l.Name, Context: p.Name + " · " + g.Name, URL: l.URL, Icon: l.Icon, NewTab: l.NewTab, PageID: p.ID})
					}
				}
			}
		}
		if a == nil {
			return nil
		}
		prefs, err := loadPrefs(q, a.ID)
		if err != nil {
			return err
		}
		if !prefs.Hidden("notes") {
			notes, err := notesFor(q, a, in.MembersEditShared, "")
			if err != nil {
				return err
			}
			for _, n := range notes {
				if match(n.Title, n.Content) {
					r := SearchResult{Kind: n.Type, ID: n.ID, Title: n.Title, Context: contextAround(n.Content, terms[0], 90), Shared: n.Visibility != "private"}
					if n.Type == "snippet" {
						r.Content = n.Content // Enter on a snippet result copies it
					}
					out = append(out, r)
				}
			}
		}
		if !prefs.Hidden("tasks") {
			lists, err := listsFor(q, a, in.MembersEditShared)
			if err != nil {
				return err
			}
			for _, l := range lists {
				for _, it := range l.Items {
					if match(it.Text, l.Title) {
						out = append(out, SearchResult{Kind: "task", ID: it.ID, Title: it.Text, Context: l.Title, ListID: l.ID, Done: it.Done, Shared: l.Visibility != "private"})
					}
				}
			}
		}
		if !prefs.Hidden("directory") {
			entries, err := entriesFor(q, a, in.MembersEditShared, "")
			if err != nil {
				return err
			}
			for _, e := range entries {
				if match(e.Name, e.Category, e.Phone, e.Email, e.Address, e.URL, e.IP, e.MAC, e.Notes) {
					out = append(out, SearchResult{Kind: "entry", ID: e.ID, Title: e.Name, Context: e.summary(), Shared: e.Visibility != "private"})
				}
			}
		}
		return nil
	})
	// Results whose title matches come first; otherwise keep the section order.
	inTitle := func(r SearchResult) bool {
		t := strings.ToLower(r.Title)
		for _, term := range terms {
			if !strings.Contains(t, term) {
				return false
			}
		}
		return true
	}
	sort.SliceStable(out, func(i, j int) bool { return inTitle(out[i]) && !inTitle(out[j]) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, err
}

// ---------- counts ----------

type Counts struct {
	Pages     int `json:"pages"`
	Links     int `json:"links"`
	Notes     int `json:"notes"`
	Lists     int `json:"task_lists"`
	Tasks     int `json:"tasks"`
	Directory int `json:"directory"`
	Shared    int `json:"shared"`
	Users     int `json:"users"`
}

// MyCounts counts what a owns.
func (s *Store) MyCounts(a *Actor) Counts {
	var c Counts
	_ = s.read(func(q querier) error {
		rows, err := q.Query(`SELECT groups_json FROM pages WHERE owner_id = ?`, a.ID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var gj string
			var groups []Group
			if rows.Scan(&gj) == nil && json.Unmarshal([]byte(gj), &groups) == nil {
				c.Pages++
				for _, g := range groups {
					c.Links += len(g.Links)
				}
			}
		}
		rows.Close()
		_ = q.QueryRow(`SELECT count(*) FROM notes WHERE owner_id = ?`, a.ID).Scan(&c.Notes)
		_ = q.QueryRow(`SELECT count(*) FROM task_lists WHERE owner_id = ?`, a.ID).Scan(&c.Lists)
		_ = q.QueryRow(`SELECT count(*) FROM task_items WHERE list_id IN (SELECT id FROM task_lists WHERE owner_id = ?)`, a.ID).Scan(&c.Tasks)
		_ = q.QueryRow(`SELECT count(*) FROM entries WHERE owner_id = ?`, a.ID).Scan(&c.Directory)
		_ = q.QueryRow(`SELECT (SELECT count(*) FROM pages WHERE owner_id = ?1 AND visibility != 'private') + (SELECT count(*) FROM notes WHERE owner_id = ?1 AND visibility != 'private')
			+ (SELECT count(*) FROM task_lists WHERE owner_id = ?1 AND visibility != 'private') + (SELECT count(*) FROM entries WHERE owner_id = ?1 AND visibility != 'private')`, a.ID).Scan(&c.Shared)
		_ = q.QueryRow(`SELECT count(*) FROM users`).Scan(&c.Users)
		return nil
	})
	return c
}
