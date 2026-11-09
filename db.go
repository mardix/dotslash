package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/tetratelabs/wazero"
)

// ---------- schema ----------

// migrations[i] moves the database from user_version i to i+1. Append only:
// never edit a migration that has shipped.
var migrations = []string{
	`CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE users (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL UNIQUE COLLATE NOCASE,
  name TEXT NOT NULL DEFAULT '',
  role TEXT NOT NULL CHECK (role IN ('admin','member')),
  pw_hash TEXT NOT NULL DEFAULT '',
  token_ver INTEGER NOT NULL DEFAULT 1,
  disabled INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);
CREATE TABLE instance (id INTEGER PRIMARY KEY CHECK (id = 1), data TEXT NOT NULL);
CREATE TABLE prefs (user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE, data TEXT NOT NULL);
CREATE TABLE pages (
  id TEXT PRIMARY KEY,
  owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  visibility TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('private','shared','public')),
  name TEXT NOT NULL,
  groups_json TEXT NOT NULL DEFAULT '[]',
  rev INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  modified_at TEXT NOT NULL
);
CREATE TABLE notes (
  id TEXT PRIMARY KEY,
  owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  visibility TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('private','shared')),
  type TEXT NOT NULL DEFAULT 'note',
  title TEXT NOT NULL,
  content TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  modified_at TEXT NOT NULL
);
CREATE TABLE task_lists (
  id TEXT PRIMARY KEY,
  owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  visibility TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('private','shared')),
  title TEXT NOT NULL,
  created_at TEXT NOT NULL,
  modified_at TEXT NOT NULL
);
CREATE TABLE task_items (
  id TEXT PRIMARY KEY,
  list_id TEXT NOT NULL REFERENCES task_lists(id) ON DELETE CASCADE,
  text TEXT NOT NULL,
  done INTEGER NOT NULL DEFAULT 0,
  due TEXT NOT NULL DEFAULT '',
  position INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  done_at TEXT NOT NULL DEFAULT '',
  done_by TEXT NOT NULL DEFAULT ''
);
CREATE TABLE entries (
  id TEXT PRIMARY KEY,
  owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  visibility TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('private','shared')),
  name TEXT NOT NULL DEFAULT '', category TEXT NOT NULL DEFAULT '', phone TEXT NOT NULL DEFAULT '',
  email TEXT NOT NULL DEFAULT '', address TEXT NOT NULL DEFAULT '', url TEXT NOT NULL DEFAULT '',
  ip TEXT NOT NULL DEFAULT '', mac TEXT NOT NULL DEFAULT '', notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  modified_at TEXT NOT NULL
);
CREATE TABLE scratchpad (
  user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  content TEXT NOT NULL DEFAULT '',
  modified_at TEXT NOT NULL
);
-- Each person's own order and pins, for their items and shared ones alike.
CREATE TABLE user_state (
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  item_id TEXT NOT NULL,
  position INTEGER,
  pinned INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (user_id, kind, item_id)
);
CREATE INDEX pages_owner ON pages(owner_id);
CREATE INDEX notes_owner ON notes(owner_id);
CREATE INDEX lists_owner ON task_lists(owner_id);
CREATE INDEX items_list ON task_items(list_id, position);
CREATE INDEX entries_owner ON entries(owner_id);
CREATE INDEX state_item ON user_state(kind, item_id);`,
}

func schemaVersion() int { return len(migrations) }

// ---------- store ----------

// Store wraps the SQLite database. The pool has a single connection, so
// every query is serialized; helpers take a querier (the *sql.DB or the
// current *sql.Tx) and must never reach for s.db on their own.
type Store struct {
	mu         sync.RWMutex // write-locked only while the database file is swapped
	db         *sql.DB
	dir        string
	path       string
	backupDir  string
	keep       int
	maxNoteLen int
}

type querier interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

var runtimeOnce sync.Once

// setupRuntime caches SQLite's compiled engine in the data folder, so only
// the very first start pays for compiling it (time and a memory spike).
func setupRuntime(dir string) {
	runtimeOnce.Do(func() {
		cfg := wazero.NewRuntimeConfig()
		cacheDir := filepath.Join(dir, ".cache", "sqlite")
		if err := os.MkdirAll(cacheDir, 0o700); err == nil {
			if cc, err := wazero.NewCompilationCacheWithDir(cacheDir); err == nil {
				cfg = cfg.WithCompilationCache(cc)
			}
		}
		sqlite3.RuntimeConfig = cfg
	})
}

func openDB(path string) (*sql.DB, error) {
	dsn := "file:" + filepath.ToSlash(path) + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(10000)&_pragma=synchronous(NORMAL)&_txlock=immediate"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	var v int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		return err
	}
	if v > schemaVersion() {
		return fmt.Errorf("the database is from a newer DotSlash (schema %d, this build knows %d)", v, schemaVersion())
	}
	for ; v < schemaVersion(); v++ {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(migrations[v]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", v+1, err)
		}
		if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, v+1)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func NewStore(dir string, keep, maxNoteKB int) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(dir, "backups"), 0o700); err != nil {
		return nil, err
	}
	setupRuntime(dir)
	s := &Store{
		dir:        dir,
		path:       filepath.Join(dir, "dotslash.db"),
		backupDir:  filepath.Join(dir, "backups"),
		keep:       keep,
		maxNoteLen: maxNoteKB * 1024,
	}
	db, err := openDB(s.path)
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	s.db = db
	_ = os.Chmod(s.path, 0o600)
	if err := s.ensureInstance(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Close()
}

func (s *Store) Path() string { return s.path }

// read runs fn against the database.
func (s *Store) read(fn func(q querier) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return fn(s.db)
}

// tx runs fn in a write transaction.
func (s *Store) tx(fn func(q querier) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Store) ensureInstance() error {
	return s.tx(func(q querier) error {
		var n int
		if err := q.QueryRow(`SELECT count(*) FROM instance`).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			return nil
		}
		b, _ := json.Marshal(defaultInstance(appName))
		_, err := q.Exec(`INSERT INTO instance (id, data) VALUES (1, ?)`, string(b))
		return err
	})
}

// meta values (the generated session secret lives here).
func (s *Store) Meta(key string) (string, error) {
	var v string
	err := s.read(func(q querier) error {
		err := q.QueryRow(`SELECT value FROM meta WHERE key = ?`, key).Scan(&v)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	})
	return v, err
}

func (s *Store) SetMeta(key, value string) error {
	return s.tx(func(q querier) error {
		_, err := q.Exec(`INSERT INTO meta (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
		return err
	})
}

// ---------- backups ----------

type BackupInfo struct {
	Name   string    `json:"name"`
	Time   time.Time `json:"time"`
	Reason string    `json:"reason"`
	Size   int64     `json:"size"`
}

var backupRe = regexp.MustCompile(`^dotslash-(\d{8}-\d{6})-([a-z-]+)\.db$`)

// snapshot writes a consistent copy of the live database to dst.
func (s *Store) snapshotLocked(dst string) error {
	_ = os.Remove(dst)
	_, err := s.db.Exec(`VACUUM INTO ?`, dst)
	return err
}

func (s *Store) Backup(reason string) (BackupInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.backupLocked(reason)
}

func (s *Store) backupLocked(reason string) (BackupInfo, error) {
	t := time.Now()
	var name, p string
	for i := 0; ; i++ {
		name = fmt.Sprintf("dotslash-%s-%s.db", t.Add(time.Duration(i)*time.Second).Format("20060102-150405"), reason)
		p = filepath.Join(s.backupDir, name)
		if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
			break
		}
	}
	if err := s.snapshotLocked(p); err != nil {
		return BackupInfo{}, err
	}
	_ = os.Chmod(p, 0o600)
	fi, _ := os.Stat(p)
	s.prune()
	var size int64
	if fi != nil {
		size = fi.Size()
	}
	return BackupInfo{Name: name, Time: t, Reason: reason, Size: size}, nil
}

func (s *Store) prune() {
	list := s.Backups()
	for i := s.keep; i < len(list); i++ {
		_ = os.Remove(filepath.Join(s.backupDir, list[i].Name))
	}
}

// Backups lists backups, newest first.
func (s *Store) Backups() []BackupInfo {
	entries, err := os.ReadDir(s.backupDir)
	if err != nil {
		return []BackupInfo{}
	}
	out := []BackupInfo{}
	for _, e := range entries {
		m := backupRe.FindStringSubmatch(e.Name())
		if m == nil || e.IsDir() {
			continue
		}
		t, err := time.ParseInLocation("20060102-150405", m[1], time.Local)
		if err != nil {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, BackupInfo{Name: e.Name(), Time: t, Reason: m[2], Size: info.Size()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name > out[j].Name })
	return out
}

func (s *Store) BackupPath(name string) (string, error) {
	if !backupRe.MatchString(name) {
		return "", ErrNotFound
	}
	p := filepath.Join(s.backupDir, name)
	if _, err := os.Stat(p); err != nil {
		return "", ErrNotFound
	}
	return p, nil
}

// EnsureDaily makes a "daily" backup if the newest one is older than 24h.
func (s *Store) EnsureDaily() {
	for _, b := range s.Backups() {
		if b.Reason == "daily" {
			if time.Since(b.Time) < 24*time.Hour {
				return
			}
			break
		}
	}
	if _, err := s.Backup("daily"); err != nil {
		logf("daily backup failed: %v", err)
	}
}

// WriteSnapshot streams a consistent copy of the whole database to w.
func (s *Store) WriteSnapshot(w io.Writer) error {
	tmp, err := os.CreateTemp(s.dir, ".download-*.db")
	if err != nil {
		return err
	}
	name := tmp.Name()
	tmp.Close()
	defer os.Remove(name)
	s.mu.RLock()
	err = s.snapshotLocked(name)
	s.mu.RUnlock()
	if err != nil {
		return err
	}
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

// checkDBFile opens a candidate database read-only and makes sure it is a
// DotSlash database this build can use, with at least one active admin.
func checkDBFile(p string) error {
	db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(p)+"?mode=ro")
	if err != nil {
		return invalid("not a database file")
	}
	defer db.Close()
	var ok string
	if err := db.QueryRow(`PRAGMA quick_check`).Scan(&ok); err != nil || ok != "ok" {
		return invalid("this file is not a valid SQLite database")
	}
	var v, admins int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil || v < 1 {
		return invalid("this is not a DotSlash database")
	}
	if v > schemaVersion() {
		return invalid("this database is from a newer version of DotSlash")
	}
	if err := db.QueryRow(`SELECT count(*) FROM users WHERE role = 'admin' AND disabled = 0`).Scan(&admins); err != nil {
		return invalid("this is not a DotSlash database")
	}
	if admins == 0 {
		return invalid("this database has no active admin")
	}
	return nil
}

// Restore replaces the live database with a backup.
func (s *Store) Restore(name string) error {
	p, err := s.BackupPath(name)
	if err != nil {
		return err
	}
	return s.replaceWith(p, "before-restore")
}

// RestoreFile replaces the live database with an uploaded file.
func (s *Store) RestoreFile(r io.Reader) error {
	tmp, err := os.CreateTemp(s.dir, ".upload-*.db")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()
	return s.replaceWith(name, "before-restore")
}

func (s *Store) replaceWith(src, reason string) error {
	if err := checkDBFile(src); err != nil {
		return err
	}
	// Copy first, so a backup that is about to be pruned stays usable.
	staged := s.path + ".incoming"
	if err := copyFile(src, staged); err != nil {
		return err
	}
	defer os.Remove(staged)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.backupLocked(reason); err != nil {
		return fmt.Errorf("backup before restore failed: %w", err)
	}
	if err := s.db.Close(); err != nil {
		return err
	}
	_ = os.Remove(s.path + "-wal")
	_ = os.Remove(s.path + "-shm")
	if err := os.Rename(staged, s.path); err != nil {
		return err
	}
	db, err := openDB(s.path)
	if err == nil {
		err = migrate(db)
	}
	if err != nil {
		return fmt.Errorf("reopen after restore: %w", err)
	}
	s.db = db
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
