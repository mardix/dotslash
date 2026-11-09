// DotSlash — a small, fast, self-hosted start page for links, notes, tasks
// and a directory, for one person or a whole household.
package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// version is set at build time with -ldflags "-X main.version=…".
var version = "dev"

//go:embed web
var webFS embed.FS

//go:embed icons.json
var iconIndex []byte

// Fixed defaults. Only the few things people actually change are env vars.
const (
	sessionDays = 30
	backupKeep  = 7
	maxNoteKB   = 100
	iconCDN     = "https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons"
	appName     = "DotSlash"
	repoURL     = "https://github.com/mardix/dotslash"
)

type Config struct {
	Port       string
	DataDir    string
	User       string
	Password   string
	Secret     string
	TrustProxy bool

	// fixed, kept on the struct so handlers have one place to look
	SessionDays  int
	CookieSecure string
	BackupKeep   int
	MaxNoteKB    int
	IconCDN      string
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return def
}

func loadConfig() Config {
	c := Config{
		Port:         env("DS_PORT", "8080"),
		DataDir:      env("DS_DATA_DIR", "./data"),
		User:         env("DS_USER", "admin"),
		Password:     os.Getenv("DS_PASSWORD"),
		Secret:       os.Getenv("DS_SECRET"),
		SessionDays:  sessionDays,
		CookieSecure: "auto",
		BackupKeep:   backupKeep,
		MaxNoteKB:    maxNoteKB,
		IconCDN:      iconCDN,
	}
	switch strings.ToLower(env("DS_TRUST_PROXY", "false")) {
	case "1", "true", "yes", "on":
		c.TrustProxy = true
	case "0", "false", "no", "off":
	default:
		log.Fatal("DS_TRUST_PROXY must be true or false")
	}
	if n, err := strconv.Atoi(c.Port); err != nil || n < 1 || n > 65535 {
		log.Fatal("DS_PORT must be a port number")
	}
	return c
}

func logf(format string, a ...any) { log.Printf(format, a...) }

func main() {
	log.SetFlags(log.LstdFlags)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "healthcheck":
			os.Exit(healthcheck())
		case "version", "--version", "-v":
			fmt.Println("dotslash", version)
			return
		case "reset-password":
			os.Exit(resetPassword(os.Args[2:]))
		case "warmup":
			os.Exit(warmup())
		default:
			fmt.Println("usage: dotslash [healthcheck | version | reset-password <username>]\nConfigure with DS_* environment variables; see README.md.")
			return
		}
	}

	cfg := loadConfig()
	prepareSQLite(cfg.DataDir)
	store, err := NewStore(cfg.DataDir, cfg.BackupKeep, cfg.MaxNoteKB)
	if err != nil {
		log.Fatalf("open data: %v", err)
	}
	if err := store.Bootstrap(cfg.User, cfg.Password, cfg.Password != ""); err != nil {
		log.Fatalf("set up accounts: %v", err)
	}
	auth, err := NewAuth(cfg, store)
	if err != nil {
		log.Fatalf("sessions: %v", err)
	}
	srv := NewServer(cfg, store, auth)

	if !auth.enabled {
		logf("WARNING: DS_PASSWORD is not set — anyone who can reach this server can see and edit everything.")
	}

	store.EnsureDaily()
	go func() {
		t := time.NewTicker(time.Hour)
		for range t.C {
			store.EnsureDaily()
		}
	}()

	httpSrv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       5 * time.Minute, // database uploads
		WriteTimeout:      5 * time.Minute, // database downloads
		IdleTimeout:       120 * time.Second,
	}
	go func() {
		logf("%s %s listening on :%s (data: %s, sign-in: %v, users: %d)", appName, version, cfg.Port, store.Path(), auth.enabled, store.UserCount())
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
	_ = store.Close()
	logf("stopped")
}

// prepareSQLite compiles SQLite's engine into the data folder's cache in a
// short-lived child process on the very first start. Compiling briefly needs
// ~70 MB; doing it in a child keeps the server itself at its usual ~15 MB.
func prepareSQLite(dir string) {
	cache := filepath.Join(dir, ".cache", "sqlite")
	if entries, err := os.ReadDir(cache); err == nil && len(entries) > 0 {
		return
	}
	self, err := os.Executable()
	if err != nil {
		return
	}
	logf("first start: preparing SQLite (one time)…")
	cmd := exec.Command(self, "warmup")
	cmd.Env = append(os.Environ(), "DS_DATA_DIR="+dir)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		logf("note: could not prepare SQLite ahead of time (%v); continuing", err)
	}
}

func warmup() int {
	dir := env("DS_DATA_DIR", "./data")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	setupRuntime(dir)
	db, err := openDB(":memory:")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	db.Close()
	return 0
}

// resetPassword gives a user a new random password and prints it:
//
//	docker exec dotslash /dotslash reset-password <username>
func resetPassword(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: dotslash reset-password <username>")
		return 2
	}
	store, err := NewStore(env("DS_DATA_DIR", "./data"), backupKeep, maxNoteKB)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open data:", err)
		return 1
	}
	defer store.Close()
	pw, err := store.ResetPassword(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("New password for %s: %s\nThey are signed out everywhere; sign in with it and change it under Settings → Profile.\n", args[0], pw)
	return 0
}

func healthcheck() int {
	c := http.Client{Timeout: 3 * time.Second}
	resp, err := c.Get("http://127.0.0.1:" + env("DS_PORT", "8080") + "/healthz")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 1
	}
	return 0
}

// ---------- static assets ----------

type assets struct {
	index   []byte
	files   fs.FS
	hash    string
	iconsGz []byte
}

func loadAssets() assets {
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}
	idx, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		log.Fatal(err)
	}
	h := sha256.New()
	_ = fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			b, _ := fs.ReadFile(sub, p)
			h.Write(b)
		}
		return nil
	})
	h.Write(iconIndex)
	var gz bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&gz, gzip.BestCompression)
	zw.Write(iconIndex)
	zw.Close()
	return assets{index: idx, files: sub, hash: hex.EncodeToString(h.Sum(nil))[:10], iconsGz: gz.Bytes()}
}

func renderIndex(a assets, st Settings, in Instance) []byte {
	r := strings.NewReplacer(
		"{{TITLE}}", html.EscapeString(st.Title),
		"{{THEME}}", st.Theme,
		"{{ACCENT}}", st.Accent,
		"{{V}}", a.hash,
		"{{ICONV}}", iconVersion(in),
	)
	return []byte(r.Replace(string(a.index)))
}

func iconVersion(in Instance) string {
	h := sha256.Sum256([]byte(in.Icon.Type + in.Icon.Letter + in.Icon.Color + in.Icon.Name + in.Accent + strconv.Itoa(len(in.Icon.Image)) + in.Icon.Image[:min(64, len(in.Icon.Image))]))
	return hex.EncodeToString(h[:])[:8]
}
