package main

import (
	"encoding/json"
	"strings"
	"time"
)

// Export is the "my data" file: everything one person owns, in one JSON
// document. It is also the import format.
type Export struct {
	Format     string        `json:"format"`  // "dotslash"
	Version    int           `json:"version"` // 2
	ExportedAt time.Time     `json:"exported_at"`
	User       string        `json:"user,omitempty"`
	Prefs      *Prefs        `json:"prefs,omitempty"`
	Pages      []ExportPage  `json:"pages"`
	Notes      []ExportNote  `json:"notes"`
	Tasks      []ExportList  `json:"tasks"`
	Directory  []ExportEntry `json:"directory"`
	Scratchpad string        `json:"scratchpad"`
}

type ExportPage struct {
	ID         string  `json:"id,omitempty"`
	Name       string  `json:"name"`
	Visibility string  `json:"visibility,omitempty"`
	Groups     []Group `json:"groups"`
}

type ExportNote struct {
	Type       string    `json:"type,omitempty"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Pinned     bool      `json:"pinned,omitempty"`
	Visibility string    `json:"visibility,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
}

type ExportList struct {
	Title      string       `json:"title"`
	Visibility string       `json:"visibility,omitempty"`
	Items      []ExportItem `json:"items"`
}

type ExportItem struct {
	Text      string     `json:"text"`
	Done      bool       `json:"done,omitempty"`
	Due       string     `json:"due,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	DoneAt    *time.Time `json:"done_at,omitempty"`
}

type ExportEntry struct {
	Name       string `json:"name"`
	Category   string `json:"category,omitempty"`
	Phone      string `json:"phone,omitempty"`
	Email      string `json:"email,omitempty"`
	Address    string `json:"address,omitempty"`
	URL        string `json:"url,omitempty"`
	IP         string `json:"ip,omitempty"`
	MAC        string `json:"mac,omitempty"`
	Notes      string `json:"notes,omitempty"`
	Visibility string `json:"visibility,omitempty"`
}

type ImportSummary struct {
	Pages     int `json:"pages"`
	Links     int `json:"links"`
	Notes     int `json:"notes"`
	TaskLists int `json:"task_lists"`
	Tasks     int `json:"tasks"`
	Directory int `json:"directory"`
}

// ExportMine collects what a owns.
func (s *Store) ExportMine(a *Actor) (Export, error) {
	ex := Export{Format: "dotslash", Version: 2, ExportedAt: now(), User: a.Username,
		Pages: []ExportPage{}, Notes: []ExportNote{}, Tasks: []ExportList{}, Directory: []ExportEntry{}}
	err := s.read(func(q querier) error {
		p, err := loadPrefs(q, a.ID)
		if err != nil {
			return err
		}
		ex.Prefs = &p
		pages, err := pagesFor(q, a, false)
		if err != nil {
			return err
		}
		for _, pg := range pages {
			if pg.Owner == a.ID {
				ex.Pages = append(ex.Pages, ExportPage{ID: pg.ID, Name: pg.Name, Visibility: pg.Visibility, Groups: pg.Groups})
			}
		}
		notes, err := notesFor(q, a, false, "")
		if err != nil {
			return err
		}
		for _, n := range notes {
			if n.Owner == a.ID {
				ex.Notes = append(ex.Notes, ExportNote{Type: n.Type, Title: n.Title, Content: n.Content, Pinned: n.Pinned, Visibility: n.Visibility, CreatedAt: n.CreatedAt, ModifiedAt: n.ModifiedAt})
			}
		}
		lists, err := listsFor(q, a, false)
		if err != nil {
			return err
		}
		for _, l := range lists {
			if l.Owner != a.ID {
				continue
			}
			el := ExportList{Title: l.Title, Visibility: l.Visibility, Items: []ExportItem{}}
			for _, it := range l.Items {
				el.Items = append(el.Items, ExportItem{Text: it.Text, Done: it.Done, Due: it.Due, CreatedAt: it.CreatedAt, DoneAt: it.DoneAt})
			}
			ex.Tasks = append(ex.Tasks, el)
		}
		entries, err := entriesFor(q, a, false, "")
		if err != nil {
			return err
		}
		for _, e := range entries {
			if e.Owner == a.ID {
				ex.Directory = append(ex.Directory, ExportEntry{Name: e.Name, Category: e.Category, Phone: e.Phone, Email: e.Email, Address: e.Address,
					URL: e.URL, IP: e.IP, MAC: e.MAC, Notes: e.Notes, Visibility: e.Visibility})
			}
		}
		pad, err := getPad(q, a.ID)
		ex.Scratchpad = pad.Content
		return err
	})
	return ex, err
}

func ParseExport(raw []byte) (*Export, ImportSummary, error) {
	var ex Export
	var sum ImportSummary
	if err := json.Unmarshal(raw, &ex); err != nil {
		return nil, sum, invalid("not a valid DotSlash file: %v", err)
	}
	if ex.Format != "dotslash" || ex.Version != 2 {
		return nil, sum, invalid("this is not a DotSlash export (format “dotslash”, version 2)")
	}
	sum.Pages = len(ex.Pages)
	for _, p := range ex.Pages {
		for _, g := range p.Groups {
			sum.Links += len(g.Links)
		}
	}
	sum.Notes = len(ex.Notes)
	sum.TaskLists = len(ex.Tasks)
	for _, l := range ex.Tasks {
		sum.Tasks += len(l.Items)
	}
	sum.Directory = len(ex.Directory)
	return &ex, sum, nil
}

// ImportMine replaces everything a owns with the contents of ex. A backup is
// taken first. Things shared by other people are not touched.
func (s *Store) ImportMine(a *Actor, ex *Export) error {
	if _, err := s.Backup("before-import"); err != nil {
		return err
	}
	vis := func(kind, v string) string {
		if v == "" {
			return "private"
		}
		if checkVisibility(a, kind, v) != nil {
			if v == "public" {
				return "shared"
			}
			return "private"
		}
		return v
	}
	return s.tx(func(q querier) error {
		if err := deleteOrphanState(q, a.ID); err != nil {
			return err
		}
		for _, t := range []string{"pages", "notes", "task_lists", "entries"} {
			if _, err := q.Exec(`DELETE FROM `+t+` WHERE owner_id = ?`, a.ID); err != nil {
				return err
			}
		}
		t := now()
		pageIDs := map[string]string{}
		order := []string{}
		for _, p := range ex.Pages {
			id := newID()
			if p.ID != "" {
				pageIDs[p.ID] = id
			}
			name := clip(p.Name, 60)
			if name == "" {
				name = "Untitled"
			}
			gj, _ := json.Marshal(normalizeGroups(p.Groups))
			if _, err := q.Exec(`INSERT INTO pages (id, owner_id, visibility, name, groups_json, rev, created_at, modified_at) VALUES (?, ?, ?, ?, ?, 1, ?, ?)`,
				id, a.ID, vis("page", p.Visibility), name, string(gj), ts(t), ts(t)); err != nil {
				return err
			}
			order = append(order, id)
		}
		if err := setPositions(q, a.ID, "page", order); err != nil {
			return err
		}
		for _, n := range ex.Notes {
			id := newID()
			title := clip(n.Title, 200)
			if title == "" {
				title = "Untitled"
			}
			if len(n.Content) > s.maxNoteLen {
				return invalid("the note “%s” is larger than %d KB", title, s.maxNoteLen/1024)
			}
			created, modified := n.CreatedAt, n.ModifiedAt
			if created.IsZero() {
				created = t
			}
			if modified.IsZero() {
				modified = created
			}
			if _, err := q.Exec(`INSERT INTO notes (id, owner_id, visibility, type, title, content, created_at, modified_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				id, a.ID, vis("note", n.Visibility), noteType(n.Type), title, n.Content, ts(created), ts(modified)); err != nil {
				return err
			}
			if n.Pinned {
				if err := setPinned(q, a.ID, id, true); err != nil {
					return err
				}
			}
		}
		order = order[:0]
		for _, l := range ex.Tasks {
			id := newID()
			title := clip(l.Title, 120)
			if title == "" {
				title = "Untitled"
			}
			if _, err := q.Exec(`INSERT INTO task_lists (id, owner_id, visibility, title, created_at, modified_at) VALUES (?, ?, ?, ?, ?, ?)`,
				id, a.ID, vis("list", l.Visibility), title, ts(t), ts(t)); err != nil {
				return err
			}
			order = append(order, id)
			for i, it := range l.Items {
				text := clip(it.Text, 500)
				if text == "" {
					continue
				}
				due := it.Due
				if !dateRe.MatchString(due) {
					due = ""
				}
				created := it.CreatedAt
				if created.IsZero() {
					created = t
				}
				done, doneAt, doneBy := 0, "", ""
				if it.Done {
					done, doneBy = 1, a.ID
					if it.DoneAt != nil {
						doneAt = ts(*it.DoneAt)
					} else {
						doneAt = ts(t)
					}
				}
				if _, err := q.Exec(`INSERT INTO task_items (id, list_id, text, done, due, position, created_at, done_at, done_by) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					newID(), id, text, done, due, i, ts(created), doneAt, doneBy); err != nil {
					return err
				}
			}
		}
		if err := setPositions(q, a.ID, "list", order); err != nil {
			return err
		}
		for _, x := range ex.Directory {
			e := Entry{Name: x.Name, Category: x.Category, Phone: x.Phone, Email: x.Email, Address: x.Address, URL: x.URL, IP: x.IP, MAC: x.MAC, Notes: x.Notes}
			normalizeEntry(&e)
			if _, err := q.Exec(`INSERT INTO entries (id, owner_id, visibility, name, category, phone, email, address, url, ip, mac, notes, created_at, modified_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, newID(), a.ID, vis("entry", x.Visibility), e.Name, e.Category, e.Phone, e.Email, e.Address, e.URL, e.IP, e.MAC, e.Notes, ts(t), ts(t)); err != nil {
				return err
			}
		}
		if ex.Prefs != nil {
			p := *ex.Prefs
			if strings.HasPrefix(p.StartOn, "links:") {
				if id, ok := pageIDs[strings.TrimPrefix(p.StartOn, "links:")]; ok {
					p.StartOn = "links:" + id
				} else {
					p.StartOn = "links"
				}
			}
			normalizePrefs(&p)
			b, _ := json.Marshal(p)
			if _, err := q.Exec(`INSERT INTO prefs (user_id, data) VALUES (?, ?) ON CONFLICT(user_id) DO UPDATE SET data = excluded.data`, a.ID, string(b)); err != nil {
				return err
			}
		}
		pad := ex.Scratchpad
		if len(pad) > maxScratchpadLen {
			pad = pad[:maxScratchpadLen]
		}
		_, err := q.Exec(`INSERT INTO scratchpad (user_id, content, modified_at) VALUES (?, ?, ?)
			ON CONFLICT(user_id) DO UPDATE SET content = excluded.content, modified_at = excluded.modified_at`, a.ID, pad, ts(t))
		return err
	})
}
