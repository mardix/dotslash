# DotSlash — Engineering Handoff

> For agents and developers picking up DotSlash. This covers what was built, why, how it is put together, the visual rules, and how to make changes safely. README.md is the user-facing doc. This file is the builder-facing one.

- **Repo:** https://github.com/mardix/dotslash · **Image:** `ghcr.io/mardix/dotslash` · **License:** MIT (© 2026 Mardix)
- **State at handoff:** first release (v1), multi-user on SQLite. Builds clean with `go vet`. Two test suites pass:
  - Go unit tests (`go test ./...`, 7 groups, ~2 s);
  - browser suite (`scripts/dev/ui-check.py`, 41 checks, 0 failures, 0 page errors).

---

## 1. What DotSlash is (and is not)

A **small, fast, elegant, self-hosted start page** for one person, small teams or a household. Each person has their own links (organized into pages and groups), plus personal reference material: notes, snippets, tasks, a directory, and a scratchpad. Any of it except the scratchpad can be **shared** with everyone who signs in. Link pages can also be made **public** for visitors. Everything is edited in a friendly UI and stored in **one SQLite file**.

**It is deliberately NOT** a monitoring dashboard. There are **no** service status checks, Docker/host status, API widgets, integrations, weather or RSS. The owner said so explicitly: *"we would not need status checks. no need for the hosts docker status. simple links, well arrange."* Do not add these unless asked.

**Positioning vs others:**
- Homarr: UI-edited and multi-user, but heavy, with many integrations.
- Homepage: YAML-only, with widgets.
- Heimdall: PHP.
- Dashy: Vue, heavy.
- Glance and Homer: YAML-only.

DotSlash = UI editing like Homarr + accounts and sharing + about 15 MB of RAM and a ~17 MB image + notes, tasks and directory built in.

### Owner's taste (follow it)
- *"Simple, nice looking, low memory."* *"Professional, nothing cheesy, balanced colors."* *"Make sure everything is elegant."*
- Reduced corner radius. Active navigation is **not** button-shaped (it uses an underline). The header blends with the body. It is quiet by default.
- The owner likes being shown a mock or screenshot before big UI changes ("show me", "don't build yet", "let's think first"). For a large new feature, describe or mock it first and build after confirmation.
- Keep the env vars minimal and prefixed `DS_`.

---

## 2. Key design decisions

| Decision | Why |
|---|---|
| **SQLite** (`dotslash.db`), one file | Row-level writes (a note autosave doesn't rewrite everything), atomic sharing (a column flip, nothing moves), permission rules as SQL `WHERE` clauses, consistent online backups (`VACUUM INTO`). The whole dashboard is still one downloadable file. |
| **Pure-Go driver** `github.com/ncruces/go-sqlite3` **v0.32.0** (SQLite compiled to WebAssembly, run by `wazero` v1.11.0) | Keeps `CGO_ENABLED=0`, the static scratch image and fast cross-compiles. Pinned to v0.32.0 because later versions need Go ≥ 1.25 (the project targets Go 1.24). FTS5 is included, but search uses Go matching (see §6). |
| **Compiled-engine cache** in `/data/.cache/sqlite`, warmed by a child process on first start | wazero's compile spikes to ~70 MB. Doing it in `dotslash warmup` (a short-lived child) keeps the server at ~15 MB from the first start. |
| **Accounts:** admin + member, PBKDF2 passwords, first admin from `DS_USER`/`DS_PASSWORD` | Household-sized multi-user. Two roles plus one switch ("members can edit shared") cover both "everyone edits" and "I curate, they use". |
| **Visibility:** `private` / `shared` (all signed-in users) / `public` (pages only, admins only) | Sharing with *specific people* is deliberately not built. The schema allows adding it (a new table) later. |
| **Private means private, admins included** | Admins manage the dashboard and shared things, not people's private items. |
| **Shared task lists are always editable by everyone signed in** | Checking off chores is the point. Other shared kinds follow the switch. |
| **Pins and order are per person** (`user_state`) | My pin or my page order never changes your view. |
| **Per-page revision** (`pages.rev`) for conflicts | Two people editing *different* shared pages never collide. A clash on one page doesn't block saving the rest. |
| **Settings split:** instance (admins) vs prefs (each person) | Title, icon, header, default accent/layout, visitor landing and the members-edit switch are instance settings. Theme, accent, layout, start page, idle, sections and new-tab are personal. |
| **No migration from any earlier format** | v1 is the first release. Schema migrations exist from here on (`PRAGMA user_version`). |
| **Go + one dependency, single static binary**, assets embedded | Low memory, small image. Don't add more dependencies without a strong reason. |
| **Vanilla JS**, no build step | One `app.js`, no React/Vue/bundler. |
| **Server-enforced visibility** | Visitors only ever receive public pages. Every query is scoped to the actor. Never rely on the client to hide data. |
| **New items start private** | Nothing becomes visible by accident. |
| **Edit lock**: can't navigate while editing until Save or Close | Prevents losing a draft. Switching between link page tabs *is* allowed while editing links. |
| **Directory** is one flexible table; **snippets** are a note type | See §8. |
| **Icons**: Dashboard Icons via jsDelivr plus an embedded name index | No icon files to bundle. |
| **Search is local only** | It searches DotSlash's own content. |

---

## 3. Repository map

```
main.go          config (env), constants, embed, subcommands (healthcheck, version, reset-password, warmup),
                 prepareSQLite (first-start warmup child), index templating
db.go            SQLite: openDB (pragmas, 1 connection), migrations[], Store (read/tx helpers),
                 meta table, backups (VACUUM INTO), WriteSnapshot (download), Restore/RestoreFile (validated swap)
model.go         types (Page, Note, TaskList, Entry, Access, Instance, Prefs, Settings…), validation
                 (normalize*, clip, safeURL, regexes), errors (ErrNotFound/Conflict/Invalid/Forbidden), time helpers
content.go       PERMISSIONS (canSee/canManage/canEdit/checkVisibility) + all content queries:
                 pages & SaveLayout, notes, task lists & items, directory, scratchpad, settings, search, counts
users.go         User/Actor, PBKDF2 hashing, user CRUD with last-admin guard, sign-out-everywhere,
                 DeleteUser (handover), Bootstrap (first admin + starter content), ResetPassword
io.go            "my data" Export (v2 JSON), ParseExport, ImportMine
auth.go          signed session cookie (uid|token_ver|exp|remember), Actor(r), login/logout, rate limits,
                 Reissue (keep this session after a token bump), generated session secret
api.go           routes + gates (public / signed in / admin), middleware (CSP, CSRF header), handlers, favicon
store_test.go    unit tests (see §11)
web/index.html   shell template ({{TITLE}} {{THEME}} {{ACCENT}} {{V}} {{ICONV}})
web/boot.js      sets data-theme before first paint (no flash)
web/app.js       the entire UI (~2.9k lines, one IIFE)
web/app.css      all styles (~590 lines), tokens at top
icons.json       ~4,860 Dashboard Icons names, embedded and served gzipped
docs/            demo.json (admin "Mardix"), demo-ana.json (member "Ana"), screenshots/
scripts/update-icons.sh   regenerates icons.json
scripts/dev/seed-demo.py  loads the demo into a running server (3 people, shared items)
scripts/dev/ui-check.py   Playwright checks + regenerates README screenshots
Dockerfile       golang:1.24-alpine on $BUILDPLATFORM, cross-compiles → scratch, UID 65532
docker-compose.yml  ghcr image, hardened (read_only, cap_drop ALL, no-new-privileges, 128m)
.github/workflows/docker.yml  gofmt/vet/test/build/smoke → buildx amd64+arm64 (no QEMU) → GHCR
Makefile         run | build | test | image | icons | fmt | clean
```

---

## 4. Configuration

Only these env vars exist. Keep it that way unless there is a strong reason:

| Var | Default | Notes |
|---|---|---|
| `DS_PASSWORD` | *(empty)* | **Empty = sign-in off ("open mode").** Everyone acts as the first admin (`Store.OpenModeActor`), and there's no Users screen. With a value, it's the first admin's password, **used only while there are no users** (or to give a password to an admin created in open mode). |
| `DS_USER` | `admin` | First admin's username (same rules). |
| `DS_SECRET` | *(generated)* | Signs session cookies. If unset, a random secret is generated once and stored in the `meta` table (`session_secret`), so sessions survive restarts. |
| `DS_TRUST_PROXY` | `false` | Use X-Forwarded-For/Proto for the client IP (rate limiting) and the Secure cookie flag. Only true/false; anything else is fatal. |
| `DS_PORT` | `8080` | |
| `DS_DATA_DIR` | `./data` (`/data` in Docker) | Holds `dotslash.db` (+ `-wal`/`-shm`), `backups/` and `.cache/sqlite`. |
| `TZ` | — | Image is built with `-tags timetzdata`. |

Fixed constants in `main.go` (deliberately not env): `sessionDays=30`, `backupKeep=7`, `maxNoteKB=100`, `iconCDN`, `appName="DotSlash"`, `repoURL`. Also `minPasswordLen=8` and `pbkdfIter=600_000` in users.go (tests lower the iteration count). `version` is set via `-ldflags -X main.version=…`.

**Subcommands:**
- `dotslash healthcheck` (Docker HEALTHCHECK);
- `dotslash version`;
- `dotslash reset-password <user>` (prints a random password, bumps token_ver, re-enables the account; works in the scratch image via `docker exec`);
- `dotslash warmup` (internal: compiles SQLite into the cache).

---

## 5. Data model (SQLite, schema version 1)

```
meta        key, value                         -- session_secret
users       id, username UNIQUE NOCASE, name, role(admin|member), pw_hash, token_ver, disabled, created_at
instance    id=1, data (JSON Instance)         -- dashboard-wide settings
prefs       user_id → data (JSON Prefs)        -- each person's settings
pages       id, owner_id, visibility(private|shared|public), name, groups_json, rev, created_at, modified_at
notes       id, owner_id, visibility(private|shared), type(note|snippet), title, content, created_at, modified_at
task_lists  id, owner_id, visibility(private|shared), title, created_at, modified_at
task_items  id, list_id → task_lists ON DELETE CASCADE, text, done, due, position, created_at, done_at, done_by
entries     id, owner_id, visibility(private|shared), name, category, phone, email, address, url, ip, mac, notes, …
scratchpad  user_id → content, modified_at     -- always personal
user_state  (user_id, kind, item_id) PK, position, pinned   -- kind: page | note | list
```

- All owner FKs are `ON DELETE CASCADE`, so deleting a user deletes what they still own.
- Timestamps are RFC3339Nano UTC strings (`ts()`/`pt()` in model.go). Client conflict tokens are compared as times (`sameTime`).
- **Pages store groups and links as one JSON column** (`groups_json`), because the editor saves a page as a unit. Group and link IDs are client-generated and validated by `normalizeGroups`.
- **Instance JSON:** `title, icon{type,letter,color,image,name}, accent, layout, header_bg, header_text, public_start ("" | links:<id> | blank), members_edit_shared`.
- **Prefs JSON:** `theme, accent ("" = instance), layout ("" = instance), start_on, idle_minutes, new_tab, hidden_sections`.
- **Merged `Settings`** (what the browser gets and sends back) holds both, plus resolved `accent`/`layout` and the raw `pref_accent`/`pref_layout`/`default_accent`/`default_layout`. `SaveSettings` always stores prefs, and stores the instance part **only for admins** (members' instance fields are ignored).

**Migrations** (`db.go`): `migrations[i]` takes the DB from `user_version` i to i+1 inside a transaction. **Append only; never edit a shipped migration.** Opening a DB newer than the build refuses to start.

**Validation is the security boundary** (`normalize*`, `clip`, `safeURL`, `checkVisibility` in model.go/content.go). It applies on every write and import:
- **Lengths:** page name 60, group 80, link name 120, URL 2048, description 300, note title 200, note 100 KB, list 120, item 500, entry fields 40–1000, scratchpad 64 KB, user name 60, username 1–40 `[A-Za-z0-9._@-]`.
- **Links:** `javascript:`/`data:`/`vbscript:`/`file:` URLs are dropped.
- **Icons:** images are raster `data:image/*` only (≤400 KB; **no SVG uploads**).

**Seed** (`seedStarter`, first admin only): a public **Home** page with *Getting started* links (DotSlash → repo with the `github` icon, Singlebase → https://singlebase.io, Dashboard Icons), a Welcome note, a snippet, and a Getting started list.

**Store concurrency:**
- **One connection:** the pool has `MaxOpenConns(1)`, so all queries are serialized.
- **The querier rule:** helpers receive a `querier` (the DB or the current Tx) and **must never use `s.db` directly**. They must also fully read and close `rows` before issuing another query. Breaking either rule deadlocks the single connection.
- **Locking:** `s.read` / `s.tx` take an RLock. Only a database swap (restore) takes the write lock.
- **Pragmas:** WAL, `foreign_keys=1`, `busy_timeout=10000`, `synchronous=NORMAL`, `_txlock=immediate`.

**Backups** (`data/backups/dotslash-YYYYMMDD-HHMMSS-<reason>.db`): `daily` (checked at start and hourly), `before-import`, `before-restore`, `manual`. Each is made with `VACUUM INTO`, and the newest 7 are kept.

**Restore** (backup or upload):
1. `checkDBFile` opens the candidate read-only and checks `quick_check`, the schema version and that there is at least one active admin.
2. The candidate is staged as a copy, and a `before-restore` backup is taken.
3. The live DB is closed, its `-wal`/`-shm` files are removed, the staged copy is renamed into place, and the DB is reopened and migrated.

**Conflicts:**
- **Pages:** `SaveLayout` compares each page's `rev`. A mismatch skips that page and reports its name in `conflicts`.
- **Notes:** `base_modified` returns 409 with the current note ("Load theirs / Keep mine").
- **Scratchpad:** same as notes.

---

## 6. Permissions and HTTP API

**Permissions live in one place** (top of `content.go`); don't re-implement them in handlers:

```
canSee(a, owner, vis):    visitor → vis == public;   signed in → owner == me || vis != private
canManage(a, owner, vis): owner == me || (admin && vis != private)        -- share/unshare/delete
canEdit(a, owner, vis, kind, membersEdit):
                          canManage || (vis != private && (kind == "list" || membersEdit))
checkVisibility(a, kind, vis): private|shared ok; public only for pages and only for admins
```

- **Every item carries an `Access` block** in JSON: `visibility, owner, owner_name, can_edit, can_manage`. The UI only reads these flags; the server re-checks everything.
- **Item lookups use `mayEdit` / `mayManage`.** An item the actor can't see returns **404, not 403**, so private items stay undiscoverable.

**Gates** (`api.go`): `guard(gatePublic|gateSignedIn|gateAdmin, h)` resolves the actor once and puts it in the request context (`actor(r)`). Signed-in gates return 401 without a session; admin gates return 403 for members.

All JSON. Every non-GET under `/api/` **must** send `X-Requested-With: DotSlash` (CSRF guard); otherwise the response is 403.

```
public      GET /healthz · / · /favicon(.ico) · /assets/* · /api/icons
            GET /api/session (me) · POST /api/login · POST /api/logout
            GET /api/state   visitor: {settings, pages(public), signed_in:false} only
                             signed in: + me, notes(meta), tasks, directory, scratchpad
            GET /api/search?q=
signed in   PUT /api/settings · PUT /api/layout {pages:[{id,name,visibility,groups,rev}]} → {pages, conflicts}
            PUT /api/visibility {kind: page|note|list|entry, id, visibility}
            notes: GET/POST /api/notes · GET/PUT/DELETE /api/notes/{id}   (pinned = mine; visibility on create)
            tasks: GET/POST /api/tasks · PUT /api/tasks/order (mine) · PUT/DELETE /api/tasks/{id}
                   POST /api/tasks/{id}/items · PUT/DELETE /api/items/{id}   (done sets done_by)
            directory: POST /api/directory · PUT/DELETE /api/directory/{id}   (visibility on create)
            PUT /api/scratchpad · GET/PUT /api/me · PUT /api/me/password · POST /api/me/signout-all
            GET /api/export · POST /api/import[?dry_run=1] (my data) · GET /api/about
admin       GET/POST /api/users · PUT /api/users/{id} {name,role,disabled,password} · DELETE /api/users/{id}[?handover=1]
            GET /api/db (download) · POST /api/db/restore (raw .db body, ≤512 MB)
            GET/POST /api/backups · GET /api/backups/{name} · POST /api/backups/{name}/restore
```

**SaveLayout semantics:** the editor sends *its whole view* of pages, in order.
- **New IDs** are created as the actor's own (visibility validated). An ID belonging to an invisible page is skipped.
- **Changed pages** need `can_edit` for content and `can_manage` for visibility; otherwise the save fails with 403.
- **Stale `rev`:** the page is skipped and reported as a conflict.
- **Pages missing from the list** are deleted, but only if the actor manages them; others are ignored.
- **Order** is saved as the actor's own `user_state` positions.
- **Visitors** see public pages in their owners' order.

**Search** (`Store.Search`): Go substring matching over what the actor can see, with terms ANDed. Hidden sections (prefs) are skipped, and title matches sort first. FTS5 isn't used, to keep substring behaviour (IPs, partial words). Revisit if data grows large.

**Favicon:** generated from **instance** settings (`logo` / `letter` / `image` / `library` → 302 to the CDN); `{{ICONV}}` busts the cache. The index page is rendered per actor: their theme and resolved accent avoid a flash.

---

## 7. Security model

- **Passwords:** stored as `pbkdf2-sha256$iter$salt$hash` (Go 1.24 stdlib `crypto/pbkdf2`, 600k iterations, 16-byte salt). Comparisons are constant-time. Unknown usernames are checked against a dummy hash for equal timing.
- **Session cookie** `ds_session`:
  - value: `base64(uid|token_ver|exp|remember).HMAC-SHA256`;
  - flags: HttpOnly and SameSite=Lax, with Secure set automatically (https or a trusted proxy);
  - lifetime: 30 days with "keep me signed in", else 12 h.
- **Each request re-reads the user:** a session is valid only if the account exists, isn't disabled, still has a password, and its `token_ver` matches the cookie's.
  - These bump `token_ver`: password change/reset, sign-out-everywhere, role change and disabling.
  - `Auth.Reissue` gives the *current* session a fresh cookie, so the person stays signed in here.
- **Rate limit:** 5 failures per minute, per IP *and* per username.
- **Headers:**
  - CSP: `default-src 'self'; img-src * data: blob:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; frame-ancestors 'self'; base-uri 'none'; form-action 'self'`
  - plus `nosniff`, `no-referrer`, `X-Frame-Options: SAMEORIGIN`.
- **No inline or third-party JS.** The only external fetches are icon images from jsDelivr (`referrerpolicy=no-referrer`). Markdown is rendered client-side **after escaping HTML**, and hrefs pass `safeHref`.
- **Container:** a scratch image running as non-root 65532. Compose runs it read-only with all capabilities dropped. The DB file is chmod 600.
- **Verified by tests** (Go `TestVisitorAPI` / `TestPermissionMatrix`, and browser checks):
  - visitors get only `{settings,pages,signed_in}` with public pages;
  - visitor search returns links only;
  - all signed-in routes return 401 to visitors;
  - members get 403 on `/api/users` and `/api/db`;
  - private items 404 for everyone else, admins included.

---

## 8. Frontend architecture (`web/app.js`)

One IIFE, no framework.

- **`h(tag, attrs, ...kids)`** is the DOM builder. Attrs: `class`, `dataset`, `on*` handlers, booleans. `null` kids are skipped in `h()`, **but not in `replaceChildren`**: filter with `.filter(Boolean)` there.
- **`ic(name, size)`** returns inline stroke SVG icons (`ICONS` map; added for accounts and sharing: `users`, `logout`, `key`).
- **`api(method, url, body, {keepalive, raw, quiet401})`** adds the CSRF header. An `ArrayBuffer` body is sent as `application/octet-stream` (database restore).
- **Global state `S`:**
  - `session` (with `me`), `settings` (merged), `pages`, `notes` (meta), `noteCache`, `tasks`, `directory`, `scratchpad`;
  - `route`, `editing`, `draft`, `dirty`, filters/sorts, `about`, `backups`, `users`.
  - **`NS`** is the open note editor; **`PS`** is the scratchpad.
- **Who am I:** `me()`, `isAdmin()`, `mine(item)`, `isShared(item)`, `authOn()`, `signedIn()`.
  - `sharedMark(item)` draws the quiet 👥/🌐 marker plus the owner's name when it isn't mine.
  - `shareButton(kind, item)` + `setVisibility(kind, item, vis)` → `PUT /api/visibility`.
- **Rendering is imperative re-render:** `renderAll()` = header + sub + main + dock + pad. The header shows the person's name and avatar in `.who` (a link to `#/settings/profile`).
- **Routing (hash):** `#/links[/<pageId>]`, `#/notes[/<id>]`, `#/tasks`, `#/directory`, `#/settings[/<section>]`, `#/blank`, `#/login`. `onRoute()` handles the start page, disabled sections, the **edit lock**, note flush and `ds-last`.
- **Edit mode:**
  - **Links:** `startEdit` clones pages into `S.draft`. A page with `can_edit === false` renders read-only (with a notice) but can still be reordered. The ⋯ tab menu offers:
    - Rename (if can_edit);
    - Private / Shared / Public (if can_manage; Public for admins only);
    - Move to the front;
    - **Duplicate as my page**;
    - Delete (if can_manage).
  - **Saving links:** Save → `PUT /api/layout` with `{id,name,visibility,groups,rev}` per page. `conflicts` are reported in a toast.
  - **Tasks:** lists show Share/Delete only when the person manages them; anyone can drag lists into their own order.
  - **Directory:** rows they can't edit are `readonly`; the action cell shows a 🔒/👥 share toggle (can_manage) and delete.
  - **The dock "+":** Page / Note / Snippet / List / Entry. On the **Shared** filters, new notes and entries are created shared.
- **Filters:** Notes `All / Notes / Snippets / Pinned / Shared`; Tasks `Open / All / Done / Shared lists`; Directory categories + `Shared` (`S.dirCat === '~shared'`).
- **Fresh data:** `refreshShared()` reloads state when the tab regains focus or visibility, at most every 15 s. It's skipped while editing, typing, writing a note or with a modal open, so you see other people's changes.
- **Autosave:** notes debounce 1.2 s and flush on route change, `visibilitychange` and `beforeunload` (keepalive). The scratchpad does the same.
- **Settings** (`renderSettings`, sections from `settingsSections()`):
  - **Profile:** name → `PUT /api/me`, password, sign out everywhere, sign out.
  - **Preferences** (`#s-general`).
  - **Appearance:** accent and layout have a "Default" choice that writes `pref_accent`/`pref_layout = ''`.
  - **Dashboard** (admins): instance fields, visitors land on, members-edit switch. The switch saves immediately and then reloads state, because a debounced save would race the reload.
  - **Users** (admins): add (with a generated password and a copyable credentials box), role seg, reset password, disable, delete with handover checkbox.
  - **Data:** my data JSON for everyone; admins also get the DB download, DB restore, backups (download/restore).
  - **Security:** who sees each page I manage.
  - **About.**
  - `setSetting(patch)` merges into `S.settings`, `resolveSettings()` recomputes accent/layout, then a debounced `PUT /api/settings`.
- **Keyboard:** `/` search, `e` edit, `Esc` closes menus. In search, ↑/↓ and Enter work, and Enter on a snippet result copies it.
- **Idle return** (`armIdle`) goes back to the start page after `idle_minutes`.
- **localStorage keys** (`ds-` prefix, per browser): `ds-theme`, `ds-last`, `ds-last-page`, `ds-note-sort`, `ds-pad-open`, `ds-pad-pos`, `ds-pad-size`.

### Feature behaviors worth preserving
- **Link icons:** `iconSrcs` tries SVG then PNG; on failure the link shows initials on a tinted chip (`t0..t5`). ⚠️ Link `<img>` must **not** be `loading="lazy"` (detached lazy images never load).
- **Icon picker:** typeahead grid (exact, then prefix, then contains), used for links and the dashboard Library icon.
- **Notes:** Markdown with copy buttons on code blocks; snippets as a card grid with Copy and double-click to copy; Note|Snippet toggle in the editor.
  - **Header controls:** every control in `.note-actions` is **36px** (buttons, icon buttons, share button, type toggle). `.note-head.is-edit` aligns the controls with the title input's line. Keep both.
  - **Shared notes:** shared by others show "Shared by X · … · read-only" and no Edit unless `can_edit`.
- **Tasks:** due labels (amber within 2 days, red overdue). Shared lists show `.done-by` with the name of whoever checked an item.
- **Directory:** visible columns follow the filter; double-click/tap copies with a "Copied IP" toast; ↗/tel:/mailto:/maps actions; cards on phones.
- **Scratchpad:** floating, draggable, resizable, autosaves, conflict-checks, personal.
- **Visitors:** public pages only, no sections nav, dock `[theme | 🔒]`.
- **Footer:** `With ❤️ DotSlash` (DotSlash links to the GitHub repo), from `footerEl()`. The same footer is on the sign-in page. It's a centered block, not flex, so it reads as one line.
- **Sign-in page** (`renderLoginPage`, shown when nothing is public): the dashboard's icon and name above a **flat** box (`.modal.flat`: no shadow, normal border). The sign-in modal is flat too.

---

## 9. Visual design system (`web/app.css`)

**Character:** quiet, professional, low-contrast chrome, accent used sparingly. Nothing glossy and no gradients, apart from the "default" swatches.

**Tokens** (top of app.css; keep all colors as tokens, and define every new token in BOTH themes):
- **Radii (reduced on purpose):** `--r-sm: 4px`, `--r: 6px`, `--r-lg: 8px`. The only round things are avatars and the `.who` pill.
- **Fonts:** `--font: Inter, ui-sans-serif, system-ui…` (no web fonts); `--mono: "JetBrains Mono", ui-monospace…` for URLs, hosts, IP/MAC, code, usernames.
- **Base:** 14px/1.5.
- **Dark:** bg `#0F1115`, header `#13161B`, card `#15181E`, border `#232830`, text `#E6E8EB`, muted `#8B93A1`.
- **Light:** bg `#F5F6F8`, header `#FFFFFF`, card `#FFFFFF`, border `#E3E6EA`, text `#16191F`, muted `#5B6472`.
- **Default accent: Amber `#F0B45A`**, and the default logo color is the same amber (`defaultInstance` in model.go, `--accent` in app.css, fallbacks in app.js). **Accent presets:** Blue `#7AA2F7`, Teal `#5ED3BE`, Violet `#B08CF7`, Amber `#F0B45A`, Rose `#F0788C`, Gray `#A0AAB9`. Icon colors add Amber after "Accent". `--row-active` derives from the accent via `color-mix`. `--accent-soft` (~14%) and computed `--on-accent`.
- **Header presets:** Navy `#1E2A3B`, Deep teal `#0F3D3E`, Moss `#2E3A2F`, Plum `#33293D`, Walnut `#3A2E26`.
- **Tint chips `--t0..t5`** for initials.

**Patterns:**
- **Header:** brand · section tabs · `.hdr-end` (clock + `.who`: a 24px round avatar with initials in `--accent-soft` + name; name hidden ≤760px) · search · submenu.
  - **Tabs are text, not buttons:** the active one gets weight 600 and a 2px accent underline.
- **Sharing marks** (`.smark`): a muted 11–12px icon (👥 shared, 🌐 public) plus the owner's name (`.smark-who`) when it isn't yours. **Never colorful, never a badge.** Private items show nothing.
- **Share button** (`.share-btn`): a normal `.btn` with lock "Share"; when shared, accent text and border with 👥 "Shared".
- **Read-only shared page in edit mode:** a muted `.notice.readonly` line, no dashed outlines.
- **Users list** (`.ulist/.urow`): 30px avatar, name + @username, a 170px role `.seg`, and a fixed-width `.urow-act` (106px) so rows align even when the self row has fewer buttons.
- **Group titles:** 11px/600 uppercase .1em, `--label`.
- **Link card:** 44px min height, 1px border, `--r`. On hover the border is accent-mixed.
- **Layouts:** cards (min 330px), tiles (112px, 46px icon), compact (20px icon), rows (full-width rows, 210px grid).
- **Buttons:** 38px (36px in note headers); `.icon-btn` 34px; `.seg` for enums; `.add-btn` dashed.
- **Dock:** fixed bottom-right, blurred `--dock`, `--r-lg`, shadow. Normal `[theme | ✎ Edit | ▤ | ⚙]`; editing `[theme | + X | Close | Save]`.
- **Edit mode visuals:** dashed group outlines, accent drop targets and insertion lines, header dimmed (`.hdr.locked`, includes `.who`).
- **Toasts:** bottom-left.
- **Breakpoint:** one `@media (max-width: 760px)`.
- **Accessibility:** `:focus-visible` uses a 2px accent outline; icon-only buttons have `aria-label`s; segments and toggles use `aria-pressed`.

**When you add UI:** reuse `.btn/.seg/.field/.input/.icon-btn/.chip/.card/.smark`. Use tokens, not raw colors. Check both themes and 390px width. Keep new accent usage minimal.

---

## 10. Build, run, release

```sh
make run      # DS_PASSWORD=admin go run .  → http://localhost:8080
make test     # go test ./...
make build    # static binary ./dotslash
make image    # docker build (VERSION from git describe)
make icons    # refresh icons.json (needs git + network to GitHub)
gofmt -l . && go vet ./...     # CI fails on unformatted code
```

**Go modules:** `go.mod` requires `github.com/ncruces/go-sqlite3 v0.32.0` and `github.com/tetratelabs/wazero v1.11.0` (indirect: `ncruces/julianday`, `golang.org/x/sys v0.41.0`). `go.sum` is committed.

**CI** (`.github/workflows/docker.yml`):
1. **test** job: gofmt / vet / **`go test`** / build, then a smoke test (healthz, index, `/api/icons`).
2. **image** job: refresh icons.json, then buildx for amd64+arm64. **No QEMU**: the Dockerfile's build stage runs on `$BUILDPLATFORM` and cross-compiles with `GOOS/GOARCH`.
3. Publish to GHCR: `latest`/`main`/`sha-*` on main, semver on `v*` tags. PRs build only.

**Docker and memory:**
- **Image:** ~17 MB.
- **Measured** with the compose hardening (read-only, no capabilities, 128 MB limit):
  - ~15 MB from the first start (thanks to the warmup child);
  - ~16 MB after 50 requests.
- **First start:** the warmup child needs ~70 MB for ~1 s, which is why `mem_limit: 128m`.
- **Health:** HEALTHCHECK `/dotslash healthcheck`. The volume is `/data`.

---

## 11. Testing

**Go unit tests** (`store_test.go`, `go test ./...`, ~2 s; `TestMain` lowers PBKDF2 iterations and shares one SQLite compile cache):

| Test | Covers |
|---|---|
| `TestBootstrapAndPasswords` | first admin from env, idempotent bootstrap, duplicate/invalid usernames, short passwords |
| `TestPermissionMatrix` | private invisible to others incl. admins (404); shared read-only for members; per-person pins; shared lists editable by all + done_by; members-edit switch; members can't delete/unshare; admin unshare; public only for pages |
| `TestPagesVisibilityAndLayout` | member can't publish; visitors see public only (no owner info); others' pages can't be edited or deleted by omission; per-person order; per-page rev conflicts |
| `TestSessionsAndUsers` | login, sign-out-everywhere, disable, password change kills sessions, last-admin guards, no self-delete, delete with handover |
| `TestExportImportRoundTrip` | export → import into another account, start page remapped, others untouched, non-v2 rejected |
| `TestBackupRestoreAndMigrations` | backup → restore, before-restore backup, snapshot → RestoreFile, garbage rejected, reopen keeps schema/data |
| `TestVisitorAPI` | visitor `/api/state` has no private data; search empty; signed-in routes 401 |

**Browser suite** (`scripts/dev/ui-check.py`, 41 checks; regenerates the README screenshots):

```sh
go build -o /tmp/dotslash .
D=$(mktemp -d); DS_PASSWORD=secret123 DS_PORT=8098 DS_DATA_DIR="$D" /tmp/dotslash &
python3 scripts/dev/seed-demo.py          # admin "Mardix" (admin/secret123), ana/password1, sam/password1
pip install playwright && python -m playwright install chromium   # if needed
python3 scripts/dev/ui-check.py           # PASS/FAIL lines, exit code ≠ 0 on failure
# offline icons: git clone --filter=blob:none https://github.com/homarr-labs/dashboard-icons /tmp/di
#                DS_ICON_REPO=/tmp/di python3 scripts/dev/ui-check.py
```

- **What it covers:**
  - visitor view, footer, sign-in, header name;
  - notes filters (incl. Shared), code copy, snippet grid/copy, type switch, search-Enter copy;
  - directory columns, copy toast, actions, adding an entry;
  - scratchpad, edit menu, icon picker, settings, the Users list, DB download;
  - Ana's view: sees shared/public, not private; read-only shared note; can't unshare;
  - "done by Ana" seen by Mardix; members-edit switch;
  - 403 for members on db/users; visitor state/search isolation;
  - the footer, the amber default accent, the sign-in page (logo, flat box, footer; nothing-public is simulated by intercepting `/api/state`), light Rows layout and mobile cards.
- **Screenshots:** everything is dark except `links-light-rows.png` (the one light example). Both demo people use the default accent.
- **Re-seed before every run:** the suite adds an entry and checks a task. Use a fresh data dir + `seed-demo.py`.
- **Demo data:**
  - `docs/demo.json` (Mardix): Home public, Lab shared, Admin private; 10 notes (3 shared); 3 lists ("This week" shared); 12 entries (6 shared).
  - `docs/demo-ana.json` (Ana): School shared, Reading private; 3 notes; Groceries shared; 3 entries.
- **Pitfalls:**
  - Use specific selectors.
  - `.who` contains the avatar initials too, so read `.who-name`.
  - Rate limits (5/min) apply after failed logins.

**Building in a sandbox without the Go module proxy:** if `proxy.golang.org` is blocked but GitHub works, serve modules from a local file proxy built with `golang.org/x/mod/zip.CreateFromVCS` from the GitHub mirrors (`github.com/golang/sys` for `golang.org/x/sys`). It yields the same `h1:` hashes as the official proxy. Then use `GOPROXY=file:///path,direct GOSUMDB=off`. This is how v1 was built; normal environments just use the default proxy.

**Not verified locally:** the Dockerfile's `golang:1.24-alpine` build stage (Docker Hub unreachable here). The runtime stage was tested with locally cross-compiled binaries (amd64 run, arm64 built). The first CI run confirms the rest.

---

## 12. How to make common changes

- **New setting:**
  - Personal: add it to `Prefs` + `defaultPrefs` + `normalizePrefs`.
  - Dashboard-wide: add it to `Instance` + `defaultInstance` + `normalizeInstance`.
  - Then add it to `Settings` + `mergeSettings` + `split()`, a row in the right card in app.js (`srow` + `setSetting`), and the README.
- **New Directory column:** `Entry`, `EntryPatch`, `applyEntry`, `normalizeEntry`, the `entries` table (**new migration**: `ALTER TABLE entries ADD COLUMN …`), the `entriesFor` SELECT/scan, insert/update SQL, the export/import structs, `DIR_COLS`, `entryRow` actions, `.col-<key>` CSS.
- **New shareable kind:**
  - Server: a table with `owner_id` + `visibility`, an entry in `kindTables`, a `*For(q, a, …)` loader scoped with `owner_id = ? OR visibility != 'private'`, `Access` via `access()`, `mayEdit`/`mayManage` in mutations, `deleteOrphanState`, export/import, and search.
  - Then add tests to the permission matrix.
- **Any schema change:** append to `migrations` and bump nothing else. `schemaVersion()` is derived from the list. Add a test that opens an old-version DB if the migration transforms data.
- **New section** (like Notes/Tasks):
  - Server: as above, plus the `HiddenSections` allowlist in `normalizePrefs` and inclusion in signed-in `/api/state`.
  - Client: `SECTIONS`, the `parseHash` allowlist, `renderMain`, `renderSub`, dock "+", `startOnRe` and the start-on options.
  - **Never** expose it to visitors.
- **Change the logo:** update `LOGO` in app.js and `logoSVG()` in api.go (same paths, 64×64).
- **Rename the app:** `appName`/`repoURL`, the `index.html` noscript, footer, README, LICENSE, compose/Dockerfile labels, workflow image, DB/backup file names and regex, and the `X-Requested-With` value (both sides).
- **Keep it lean:** no new dependencies without a strong reason, no JS frameworks or bundlers, no web fonts, no external scripts.

---

## 13. Known limitations / possible next steps

- **Sharing is all-or-nothing** (everyone signed in). Per-person sharing would be a `shares(kind, item_id, user_id)` table plus one more clause in the scope SQL and `canSee`.
- **No 2FA, no SSO.** A proxy SSO header was considered and deliberately left out.
- **No live updates:** other people's changes appear when a tab regains focus (or on reload), not instantly.
- **Not built, by design:** live status checks and widgets.
- **Icons need jsDelivr** at view time.
- **Single server process** (SQLite with one connection). That's fine for households and small teams. The store layer is the place to change if that ever matters.
- **Directory "All"** with 9 columns scrolls horizontally on narrow desktops (accepted).
- **Not used yet:** `directory-edit.png` is generated but not referenced in the README.
