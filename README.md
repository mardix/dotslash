# DotSlash

A small, fast, self-hosted start page for your **links**, **notes**, **tasks** and **directory**, for one person, small teams or a household.

DotSlash is in the same family as Homarr, Heimdall and Homepage, but deliberately simpler: one small binary, one SQLite file, no background workers, no widgets polling your services. It uses about **15 MB of RAM**, and the Docker image is about **17 MB**.

- **Links**: organized into pages (tabs) and groups. Drag and drop pages, groups and links. Pick icons from a searchable library of ~4,800 [Dashboard Icons](https://dashboardicons.com) with live previews.
- **People and sharing**: everyone gets their own pages, notes, tasks, directory and scratchpad. Share any of them with everyone who signs in, and make link pages **public** for visitors.
- **Notes**: Markdown notes that save automatically while you type, plus **snippets**: plain-text notes such as commands and config blocks, shown as cards with a one-click **Copy**.
- **Tasks**: checklists with due dates and progress. Shared lists can be checked off by everyone, and show who did what.
- **Directory**: one sheet for people, services, places and devices, with phone, email, address, URL, IP and MAC. Double-click any value to copy it.
- **Scratchpad**: a floating notepad that stays open on every page and syncs between your devices.
- **Search** across everything you can see with `/`.
- **Four link layouts**: Cards, Tiles, Compact and Rows.
- **Settings in the UI**: your name, password, theme, accent, layout, sections and start page; for admins also the dashboard's name, icon and header, users, backups, and a one-click **database download**.

---

## Screenshots

| | |
|---|---|
| ![Links, dark theme](docs/screenshots/links-dark.png) **Links**, Cards layout. Your name is top right; shared pages are marked with who shared them. | ![What visitors see](docs/screenshots/visitor.png) **Visitors** see only public pages: no sections, editing or settings. |
| ![Sharing a note](docs/screenshots/sharing.png) **Sharing**: Ana's own note, shared; notes others shared show their name. | ![Shared task lists](docs/screenshots/tasks-shared.png) **Shared task lists**: everyone can check things off, and sees who did. |
| ![Users](docs/screenshots/users.png) **Users**: add people, set roles, reset passwords. | ![Edit mode](docs/screenshots/edit-mode.png) **Edit mode**: drag pages, groups and links; choose who sees each page. |
| ![Icon picker](docs/screenshots/icon-picker.png) **Icon picker** with live previews as you type. | ![Notes](docs/screenshots/notes.png) **Notes** in Markdown, saved as you type. |
| ![Snippets](docs/screenshots/snippets.png) **Snippets**: notes of the Snippet type, as copy cards. | ![Tasks](docs/screenshots/tasks.png) **Tasks** with progress and due dates. |
| ![Directory](docs/screenshots/directory.png) **Directory**: everyone and everything you need to reach. | ![Directory, devices](docs/screenshots/directory-devices.png) **Directory** filtered to Devices shows only the columns devices use. |
| ![Floating scratchpad](docs/screenshots/scratchpad.png) **Scratchpad** floats over any page; drag it anywhere. | ![Light theme, Rows layout](docs/screenshots/links-light-rows.png) **Light theme**, Rows layout. |
| ![Settings](docs/screenshots/settings.png) **Settings**: profile, preferences, dashboard, users, data. | ![Directory on a phone](docs/screenshots/mobile-directory.png) **On a phone**: rows become cards; tap to copy. |
| ![Sign-in page](docs/screenshots/signin.png) **Sign-in page**, shown when nothing is public. | ![On a phone](docs/screenshots/mobile.png) **Links on a phone**. |

---

## Contents

- [Screenshots](#screenshots)
- [Quick start](#quick-start)
- [Configuration](#configuration)
- [People, roles and sharing](#people-roles-and-sharing)
- [Using DotSlash](#using-dotslash)
- [Security](#security)
- [Data, backups, import and export](#data-backups-import-and-export)
- [Reverse proxy](#reverse-proxy)
- [Building](#building)
- [API](#api)
- [Troubleshooting](#troubleshooting)
- [Project layout](#project-layout)
- [License](#license)

---

## Quick start

```bash
mkdir dotslash && cd dotslash
curl -LO https://raw.githubusercontent.com/mardix/dotslash/main/docker-compose.yml
curl -L -o .env https://raw.githubusercontent.com/mardix/dotslash/main/.env.example
# edit .env: set DS_PASSWORD
docker compose up -d
```

Open **http://localhost:8080**, click the lock in the bottom-right corner, and sign in with `DS_USER` / `DS_PASSWORD`. That's the first **admin**. Add everyone else under **Settings → Users**.

The compose file uses the published image `ghcr.io/mardix/dotslash:latest` and keeps your data in a Docker volume named `dotslash-data`. To build from source instead, clone the repository and replace `image:` with `build: .` in `docker-compose.yml`.

Want to try it with sample data? Sign in, open **Settings → Data & backups**, and import [`docs/demo.json`](docs/demo.json). It's the admin's data from the screenshots above. [`docs/demo-ana.json`](docs/demo-ana.json) is a second person's.

### Plain `docker run`

```bash
docker run -d --name dotslash --restart unless-stopped \
  -p 8080:8080 \
  -e DS_PASSWORD='change-me' \
  -v dotslash-data:/data \
  ghcr.io/mardix/dotslash:latest
```

### Using a host folder for data

The container runs as an unprivileged user (UID **65532**), so a host folder must be writable by it:

```bash
mkdir -p ./data && sudo chown 65532:65532 ./data
```

Then mount `./data:/data` instead of the named volume. Alternatively, run as your own user with `user: "1000:1000"`.

---

## Configuration

Everything else is set inside the app, under **Settings** (the gear in the bottom-right dock).

| Variable | Default | Description |
|---|---|---|
| `DS_USER` | `admin` | Username of the **first admin**, created on the first start. |
| `DS_PASSWORD` | *(empty)* | That admin's password, used on the first start. **If empty, sign-in is off: there is one account, and anyone who can reach the page can see and edit everything** (a warning is logged). |
| `DS_SECRET` | *(generated)* | Key that signs session cookies. Optional: if unset, one is generated and kept in the database, so sign-ins survive restarts. |
| `DS_TRUST_PROXY` | `false` | Set `true` behind a reverse proxy so the real client IP and HTTPS are detected. |
| `DS_PORT` | `8080` | Port to listen on. |
| `DS_DATA_DIR` | `/data` (Docker), `./data` | Where the database and backups live. |
| `TZ` | `UTC` | Time zone for logs and backup file names. The clock on the page uses each viewer's own time zone. |

`DS_USER` and `DS_PASSWORD` only matter while there are no users yet. After that, people and passwords are managed in the app. Changing them later does nothing, except giving a password to an account created while sign-in was off.

Fixed defaults: "Keep me signed in" lasts 30 days, 7 backups are kept, passwords need 8+ characters, a note or snippet can be up to 100 KB, and the scratchpad up to 64 KB.

**Forgot the password?** `docker exec dotslash /dotslash reset-password <username>` prints a new one.

---

## People, roles and sharing

Everyone who signs in has **their own space**: pages, notes, snippets, task lists, directory entries, scratchpad and preferences (theme, accent, layout, start page). Your name is shown in the top-right corner; click it for your profile.

### Roles

| | **Member** | **Admin** |
|---|---|---|
| Own pages, notes, tasks, directory, scratchpad | ✓ | ✓ |
| Share them with everyone signed in | ✓ | ✓ |
| Make a link page **public** (visible without signing in) | | ✓ |
| Manage things others shared (unshare, delete) | | ✓ |
| Dashboard name, icon, header, defaults for visitors | | ✓ |
| Add people, set roles, reset passwords, delete accounts | | ✓ |
| Download the database, backups and restore | | ✓ |

There is always at least one active admin: the last one can't be demoted, disabled or deleted.

### Who sees what

| Visibility | Who sees it | Applies to |
|---|---|---|
| **Private** | Only you. **Not even admins.** | everything (the default) |
| **Shared** | Everyone who signs in | pages, notes, snippets, task lists, directory entries |
| **Public** | Everyone, without signing in | link pages (admins choose) |

- **Share** with the **Share** button on a note, on a task list or directory row in edit mode, or from a page tab's **⋯** menu. Pages can also be set under **Settings → Security**.
- Shared things show a small 👥 mark. Things others shared also show their name ("Shared by Ana").
- **Who can edit shared things:** the owner and admins can always edit. **Shared task lists can always be edited by everyone signed in**: add, check off and edit items. The list shows who checked each item off. Shared pages, notes and directory entries are **read-only for members**, unless an admin turns on **Settings → Dashboard → Members can edit shared things**. Only the owner or an admin can unshare or delete.
- **Pins and order are personal.** Pinning a shared note, or dragging shared pages and lists into your own order, doesn't change anyone else's view.
- The Notes, Tasks and Directory filters have a **Shared** tab.
- When someone else changes a shared thing, your tab picks it up when you come back to it. If two people edit the same shared page at the same time, the second save keeps the first person's version of that page and says so. Your other changes are saved.
- **Deleting a person** deletes everything private to them. Their shared things are deleted too, or handed to you if you tick "Keep what they shared".

### Visitors

People who aren't signed in see **public link pages only**: no section tabs, no notes, tasks, directory or scratchpad, no editing, no settings. Search covers only those links. They land on the first public page, or the one an admin picks under **Settings → Dashboard → Visitors land on**. With no public pages, visitors see only the sign-in screen. The **server** enforces all of this: nothing else is ever sent to a visitor's browser. To sign in on a wall display, open `/#/login`.

---

## Using DotSlash

### Layout

```
[ ./  DotSlash   Links · Notes · Tasks · Directory     date time  (Ma) Mardix ]
[ Search links, notes, tasks and directory                                 / ]
[ submenu: pages · All/Notes/Snippets/Pinned/Shared · Open/All/Done/Shared lists · categories ]

                              … content …

                        With ❤️ DotSlash
                                       [ ☾ | ✎ Edit  ▤  ⚙ ]   ← dock
```

The **dock** holds the theme toggle, **Edit**, the **Scratchpad** toggle and **Settings**. While editing it becomes `[ ☾ | + Page  Close  Save ]`, where the add button matches the section: **+ Note** (or **+ Snippet** on the Snippets filter), **+ List** or **+ Entry**. Visitors only get `[ ☾ | 🔒 ]`; the lock opens the sign-in.

**While editing, you can't leave the page** until you press **Save** or **Close**: the header sections, logo and search are disabled, and the back button is ignored with a reminder. On Links you can still switch between page tabs, since that's part of editing them.

### Links

- Click **Edit** (or press `E`), then:
  - **drag** page tabs, groups and links to reorder them, including between groups;
  - use the **⋯** menu on a page tab to rename it, choose who sees it (**Private**, **Shared** or, for admins, **Public**), move it to the front, duplicate it as your own, or delete it;
  - use the **⋯** menu on a group to rename or delete it;
  - click **Add link**, or the pencil on a link, to edit it.
- Changes are a draft until **Save**. **Close** asks before discarding.
- **New pages start private**, so nothing becomes visible by accident.
- A shared page you can't edit stays read-only in edit mode. You can still put it where you like among your pages, or duplicate it to make your own copy.
- **Icons:** start typing in the **Icon** field and a grid of matching icons with previews appears. Use the mouse or the arrow keys and Enter to pick one. When you type a link's name, a matching icon is filled in automatically (e.g. "Pi-hole" → `pi-hole`). You can also paste any image URL, or leave it empty for colored initials.

**Layouts** (Settings → Appearance → Link layout, per person; the default comes from Settings → Dashboard):

| Layout | What it looks like |
|---|---|
| **Cards** | Groups side by side in columns; each link is a card with its name and address. |
| **Tiles** | Groups side by side; links are square icon tiles, like an app launcher. |
| **Compact** | Narrow columns of plain text links, many per screen. |
| **Rows** | Each group gets its own full-width row; its links flow in a grid next to each other. |

### Notes and snippets

Every note has a **type**:

| | **Note** | **Snippet** |
|---|---|---|
| For | Setup steps, how-tos, ideas | Commands, SSH strings, config blocks: text you copy |
| Text | Markdown, rendered | Plain text, shown exactly as written in monospace |
| Copy | A Copy button on every code block | A big **Copy** button, or **double-click** the text |

- The bar under the search filters **All · Notes · Snippets · Pinned · Shared**. **Snippets** shows them as a grid of copy cards; click a title to open one.
- Choose the type with the **Note | Snippet** switch in the editor, and change it any time. **+ Note** in the dock creates a snippet when you're on the Snippets filter, and a shared note on the Shared filter.
- Markdown: `# heading`, `- list`, `1. list`, `- [ ] task`, `**bold**`, `*italic*`, `` `code` ``, fenced code blocks, `> quote`, `---`, `[text](https://…)`. Raw HTML is never rendered.
- Click **Edit** (or double-click a note's text). It saves automatically about a second after you stop typing, and on **Done**. `Ctrl/⌘ + S` saves immediately. `Tab` indents.
- **Share** makes a note visible to everyone who signs in. **Pin** keeps it on top, just for you.
- If something was changed in another tab, on another device or by someone else, you choose **Load theirs** or **Keep mine**. Nothing is silently overwritten.
- In search, **Enter on a snippet copies it**; `Ctrl/⌘ + Enter` opens it instead.

### Tasks

- Type in **Add a task** and press Enter. The calendar button sets an optional due date.
- Tick to finish; double-click to edit the text; hover for **due date** and **delete**.
- Due dates turn amber within two days and red when overdue.
- In edit mode, rename, share, delete and drag lists into your own order.
- **Shared lists** are for things everyone does, like groceries or chores: anyone signed in can add, check off and edit items. Finished items show who did them.

### Directory

One sheet for everyone and everything you need to reach: people, services, places and devices.

| Column | In view mode |
|---|---|
| **Name** | double-click to copy |
| **Category** | free text (People, Services, Places, Devices…); the bar under the search filters by it |
| **Phone** | copy · 📞 call |
| **Email** | copy · ✉ write an email |
| **Address** | copy · 📍 open in maps (Apple Maps on Apple devices, Google Maps elsewhere) |
| **URL** | copy · ↗ open in a new tab |
| **IP**, **MAC** | copy (monospace) |
| **Notes** | copy |

- **Double-click any value to copy it**. A message confirms ("Copied IP"). On phones, tap instead.
- **Columns follow what you're looking at:** only columns that have data are shown. Inside one category the Category column is hidden, so *Devices* shows Name · URL · IP · MAC · Notes, and *People* shows Name · Phone · Email · Address. Click a header to sort.
- **Editing:** in edit mode every cell is editable in place and saves as you leave it. `Enter` or `↓` moves down, `Tab` moves across, and `Enter` on the last row adds a new entry. The 🔒/👥 button on each row shares or unshares it. Rows left completely empty are removed on **Save** or **Close**.
- Share the household's doctor, plumber and router so everyone has them; keep the rest private.
- On phones each entry becomes a card.

### Scratchpad

A notepad for quick thoughts, things to paste between devices, or a phone number someone is reading out. It's always yours alone.

- Toggle it with the **▤** button in the dock. It **floats over every page** and stays open as you move around, and after a reload.
- Drag it by its title bar, and resize it from the corner. Its position and size are remembered per browser.
- It **saves itself** while you type and follows you to your other devices. If it changed somewhere else you choose **Load theirs** or **Keep mine**.

### Sections

Links is always on. **Settings → Preferences → Sections** turns Notes, Tasks, Directory and the Scratchpad on or off, just for you. A section that's off disappears from your header, search and start page list; nothing is deleted.

### Search

Press `/`, type, use ↑ ↓ and Enter. `Ctrl/⌘ + Enter` opens a link in a new tab. Signed in, search covers your things and everything shared, in the sections you have on. Visitors only search links on public pages.

### Settings

| Section | What's there |
|---|---|
| **Profile** | Your name (shown top right and on what you share) · username and role · change password · **Sign out everywhere else** · Sign out. |
| **Preferences** | **Sections** to show · **Start on**: a links page, Notes, Tasks, Directory, **Blank page**, or Last visited · **Return to start page** after 1–60 idle minutes · default "open in new tab". |
| **Appearance** | Theme (Dark / Light / System) · accent color · link layout. "Default" follows the dashboard's choice (amber `#F0B45A` and Cards out of the box). |
| **Dashboard** *(admins)* | Name · icon: the **./** logo (in any color), a letter, a library icon, or an uploaded image; also the browser tab icon · default accent and layout (for visitors and anyone who hasn't picked one) · header color and text · where visitors land · **Members can edit shared things**. |
| **Users** *(admins)* | Add people (with a password you set or generate, ready to copy and send) · Admin/Member · reset password · disable · delete. |
| **Data & backups** | **Export my data** (JSON) · **Import my data**. Admins also get **Download database** (.db), **Restore the database** from a .db file, and the backup list with download, **Restore** and **Back up now**. |
| **Security** | Who sees each of your pages (Private / Shared / Public) and what visitors can see. |
| **About** | Version, database location (admins), memory in use, number of people. |

The dock's moon/sun toggle switches the theme for that browser only.

### Keyboard shortcuts

| Key | Action |
|---|---|
| `/` | Focus search |
| `↑` `↓` `←` `→` `Enter` | Move through and pick search results or icons |
| `Esc` | Close the icon list, search, dialogs and menus |
| `E` | Enter edit mode |
| `Ctrl/⌘ + S` | Save the note you're editing |

---

## Security

- **Accounts:** passwords are hashed with **PBKDF2-SHA256 (600,000 iterations)** and a random salt per password.
- **Sessions:** sign-in sets an **HttpOnly, SameSite=Lax** cookie, signed with HMAC-SHA256. It carries the account and a token version, so these end every session of that person at once:
  - changing a password (other devices are signed out; this one stays signed in);
  - **Sign out everywhere else**;
  - a password reset by an admin;
  - disabling the account.
- **Rate limiting:** five failed sign-ins per minute, per IP and per username. Unknown usernames take as long to reject as wrong passwords.
- **CSRF protection:** state-changing API calls require an `X-Requested-With: DotSlash` header.
- **Permissions are checked on the server for every request**, in one place:
  - private things are never sent to anyone but their owner, admins included;
  - visitors receive public pages and appearance settings only;
  - admin-only routes return 403 for members.
- `javascript:`/`data:` link URLs are rejected, notes never render raw HTML, and a strict Content-Security-Policy only allows the app's own scripts.
- The container runs as non-root on a read-only filesystem with all capabilities dropped. Only `/data` is writable. The database file is readable by its owner only.

**Use HTTPS if the dashboard is reachable from outside your home network**, so passwords aren't sent in the clear. See [Reverse proxy](#reverse-proxy).

---

## Data, backups, import and export

```
/data
├── dotslash.db                          ← everything: people, pages, notes, tasks, settings (SQLite)
├── backups/
│   ├── dotslash-20261108-023000-daily.db
│   └── dotslash-20261106-211400-before-restore.db
└── .cache/                              ← SQLite's compiled engine (safe to delete)
```

- **Storage:** SQLite in WAL mode, with schema migrations applied automatically on start.
- **Backups** (whole database, admins): daily, before every import or restore, and on **Back up now**. The newest 7 are kept. Each one is a consistent snapshot taken while DotSlash keeps running, and can be downloaded from Settings.
- **Download database** gives admins the whole dashboard as one `.db` file: every person, page, note, task and setting. Open it with any SQLite tool, or restore it into another DotSlash with **Restore the database**. That replaces everything, takes a backup first, and checks that the file really is a DotSlash database with an admin.
- **Export my data** (anyone) downloads what *you* own as JSON. **Import my data** replaces what you own with such a file, after showing a summary and taking a backup. Things others share are never touched. Use it to move your things between dashboards, or between accounts.
- **From the host:** `docker run --rm -v dotslash-data:/data -v "$PWD":/out alpine cp /data/dotslash.db /out/` (stop DotSlash first, or use **Download database**, which is always consistent).

<details>
<summary>"My data" JSON format</summary>

```json
{
  "format": "dotslash", "version": 2, "exported_at": "2026-11-08T21:00:00Z", "user": "ana",
  "prefs": { "theme": "light", "accent": "", "layout": "", "start_on": "links",
             "idle_minutes": 0, "new_tab": true, "hidden_sections": [] },
  "pages": [
    { "id": "school", "name": "School", "visibility": "shared", "groups": [
      { "id": "g1", "name": "Kids", "links": [
        { "id": "l1", "name": "Classroom", "url": "https://classroom.google.com",
          "icon": "google-classroom", "description": "", "new_tab": true } ] } ] }
  ],
  "notes": [
    { "type": "note", "title": "Wi-Fi for guests", "content": "Markdown…", "pinned": true,
      "visibility": "shared", "created_at": "…", "modified_at": "…" },
    { "type": "snippet", "title": "Print", "content": "lp -d Brother_HL file.pdf", "visibility": "private",
      "created_at": "…", "modified_at": "…" }
  ],
  "tasks": [
    { "title": "Groceries", "visibility": "shared",
      "items": [ { "text": "Oat milk", "done": false, "due": "2026-11-12", "created_at": "…" } ] }
  ],
  "directory": [
    { "name": "Dr. Patel", "category": "People", "phone": "+1 555 0177", "visibility": "shared" }
  ],
  "scratchpad": "Pick up dry cleaning Thu"
}
```

`visibility` is `private` (default), `shared`, or for pages `public` (imported as `shared` unless you're an admin). `layout` is `cards`, `tiles`, `compact`, `rows` or `""` for the dashboard default. `start_on` is `links`, `links:<pageId>`, `notes`, `tasks`, `directory`, `blank` or `last`. IDs in the file are only used to keep `start_on` pointing at the right page; everything gets new IDs on import.
</details>

---

## Reverse proxy

Set `DS_TRUST_PROXY=true`, and only expose DotSlash through the proxy.

**Caddy**

```caddy
home.example.com {
    reverse_proxy dotslash:8080
}
```

**nginx**

```nginx
location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    client_max_body_size 512m;   # database restores
}
```

**Traefik**

```yaml
    labels:
      - traefik.enable=true
      - traefik.http.routers.dotslash.rule=Host(`home.example.com`)
      - traefik.http.routers.dotslash.entrypoints=websecure
      - traefik.http.routers.dotslash.tls.certresolver=letsencrypt
      - traefik.http.services.dotslash.loadbalancer.server.port=8080
```

---

## Building

### Automatic image builds

`.github/workflows/docker.yml` builds the image with GitHub Actions and publishes it to **GitHub Container Registry** for `linux/amd64` and `linux/arm64` (Raspberry Pi 4/5 included):

| Event | Image tags |
|---|---|
| Push to `main` | `latest`, `main`, `sha-<commit>` |
| Tag `v1.2.3` | `1.2.3`, `1.2`, `1`, `latest` |
| Pull request | built and tested, not pushed |

Each run first vets, unit-tests and smoke-tests the code, then refreshes the icon list from Dashboard Icons, so every image carries current icon names. The Dockerfile cross-compiles (DotSlash is pure Go, SQLite included), so arm64 builds as fast as amd64. No secrets are needed; it uses the built-in `GITHUB_TOKEN`. To release a version:

```bash
git tag v1.0.0 && git push origin v1.0.0
```

The first time a package is published, GHCR makes it private. To let others pull it without signing in, open the package on GitHub → **Package settings** → **Change visibility** → Public.

### Locally

Requires Go 1.24+. The only dependency is the pure-Go SQLite driver [`ncruces/go-sqlite3`](https://github.com/ncruces/go-sqlite3), so no C compiler is needed.

```bash
make run      # start on :8080 with password "admin"
make test     # Go unit tests
make build    # ./dotslash binary
make image    # Docker image
make icons    # refresh icons.json from Dashboard Icons (needs git)
```

Or without make: `CGO_ENABLED=0 go build -o dotslash . && DS_PASSWORD=change-me ./dotslash`. Cross-compile with `GOOS=linux GOARCH=arm64 go build …`.

Other commands: `dotslash version`, `dotslash reset-password <username>`, and `dotslash healthcheck` (used by Docker's HEALTHCHECK).

**Memory:** about 15 MB in use. SQLite's engine is compiled once into `/data/.cache` on the very first start. A short-lived helper process does that and needs ~70 MB for about a second, which is why the compose file allows 128 MB.

### Icon library

The icon picker searches `icons.json`, a list of Dashboard Icons names built into the binary (~70 KB, ~22 KB compressed). The images themselves load from the jsDelivr CDN in the viewer's browser. `make icons` (or `scripts/update-icons.sh`) regenerates the list. CI does this on every build.

### Updating a running install

```bash
docker compose pull && docker compose up -d
```

---

## API

State-changing requests need the header `X-Requested-With: DotSlash` and the session cookie from `POST /api/login`. Everything acts as the signed-in person and only sees what they can see.

| Method & path | Purpose |
|---|---|
| `GET /healthz` | Liveness check |
| `GET /api/session` | Sign-in status and who you are (`me`) |
| `POST /api/login` · `POST /api/logout` | `{ "username", "password", "remember" }` |
| `GET /api/state` | Visitors: public pages and appearance. Signed in: your things and everything shared (note bodies load separately) |
| `GET /api/search?q=` | Search what you can see |
| `GET /api/icons` | Icon library names |
| `PUT /api/settings` | Your preferences, plus the dashboard's when you're an admin |
| `PUT /api/layout` | Save your view of pages/groups/links: `{ "pages": [{ id, name, visibility, groups, rev }] }` in your order. Returns `{ pages, conflicts }` |
| `PUT /api/visibility` | `{ "kind": "page\|note\|list\|entry", "id", "visibility": "private\|shared\|public" }` |
| `GET/POST /api/notes` · `GET/PUT/DELETE /api/notes/{id}` | Notes and snippets (`type`, `title`, `content`, `pinned` (yours), `visibility` on create; `PUT` accepts `base_modified` for conflict checks) |
| `GET/POST /api/tasks` · `PUT/DELETE /api/tasks/{id}` · `PUT /api/tasks/order` | Task lists (order is yours) |
| `POST /api/tasks/{id}/items` · `PUT/DELETE /api/items/{id}` | Tasks (`text`, `done`, `due`, `index`) |
| `POST /api/directory` · `PUT/DELETE /api/directory/{id}` | Directory entries (`name`, `category`, `phone`, `email`, `address`, `url`, `ip`, `mac`, `notes`, `visibility` on create) |
| `PUT /api/scratchpad` | `{ "content", "base_modified" }` (409 if changed elsewhere) |
| `GET/PUT /api/me` · `PUT /api/me/password` · `POST /api/me/signout-all` | Your profile, password and sessions |
| `GET /api/export` · `POST /api/import[?dry_run=1]` | Your data as JSON · replace your data |
| `GET /api/about` | Version, memory, your counts |
| `GET/POST /api/users` · `PUT/DELETE /api/users/{id}[?handover=1]` | *(admins)* People |
| `GET /api/db` · `POST /api/db/restore` | *(admins)* Download / restore the whole database |
| `GET/POST /api/backups` · `GET /api/backups/{name}` · `POST /api/backups/{name}/restore` | *(admins)* Backups |

```bash
B=http://localhost:8080; H='X-Requested-With: DotSlash'
curl -s -c jar -H "$H" -H 'Content-Type: application/json' -d '{"username":"admin","password":"change-me"}' $B/api/login
LIST=$(curl -s -b jar $B/api/tasks | jq -r '.[0].id')
curl -s -b jar -H "$H" -H 'Content-Type: application/json' -d '{"text":"Rotate backups"}' $B/api/tasks/$LIST/items
curl -s -b jar -o dotslash.db $B/api/db     # admins: download the database
```

---

## Troubleshooting

**`DS_PASSWORD` error on `docker compose up`.** Copy `.env.example` to `.env` and set a password.

**Permission denied writing `/data`.** A host folder must be owned by UID 65532 (`sudo chown -R 65532:65532 ./data`), or set `user:` to your own UID:GID.

**Forgot a password.** An admin can reset anyone's under **Settings → Users**. Locked out entirely? `docker exec dotslash /dotslash reset-password <username>`.

**Changed `DS_PASSWORD` and it didn't work.** It's only used to create the first admin. Change passwords in the app, or with `reset-password`.

**"Too many attempts."** Five failed sign-ins within a minute; wait a minute. Behind a proxy, set `DS_TRUST_PROXY=true`, or every visitor shares the proxy's IP.

**Icons show initials, or the picker shows blank tiles.** The browser can't reach `cdn.jsdelivr.net`, for example because the network is offline or a blocker is active. Image URLs for icons still work.

**"Save or close your changes first."** You're in edit mode. Use **Save** or **Close** in the dock.

**I can't edit something someone shared.** Shared pages, notes and directory entries are read-only for members, unless an admin turns on **Members can edit shared things**. Shared task lists are always editable. Duplicate a page to make your own copy.

**Visitors only see a sign-in screen.** No page is public. An admin can make one public in **Settings → Security**.

**A page I made disappeared after signing out.** New pages start private. Share it, or (as an admin) make it public.

**The container was killed on its first start.** The one-time SQLite preparation needs about 70 MB for a second; give it at least 96 MB.

**Logs:** `docker compose logs -f dotslash`.

---

## Project layout

```
dotslash/
├── main.go               entry point, configuration, subcommands, embedded assets
├── db.go                 SQLite: open, schema migrations, backups, download, restore
├── model.go              data types, validation and limits
├── content.go            permissions (in one place), pages, notes, tasks, directory, scratchpad, settings, search
├── users.go              accounts, password hashing, first start
├── io.go                 "my data" JSON export and import
├── auth.go               sign-in, signed sessions, rate limiting
├── api.go                HTTP routes and handlers
├── store_test.go         unit tests: permissions, sessions, users, import/export, backups
├── icons.json            Dashboard Icons name index (generated)
├── web/                  index.html, boot.js, app.css, app.js: no framework, no build step
├── docs/                 demo.json, demo-ana.json, screenshots/
├── scripts/update-icons.sh
├── scripts/dev/          seed-demo.py (demo data) · ui-check.py (browser checks + screenshots)
├── .github/workflows/docker.yml
├── Dockerfile · docker-compose.yml · .env.example · Makefile
├── LICENSE
├── HANDOFF.md            architecture, decisions and design notes for contributors
└── README.md
```

---

## License

[MIT](LICENSE). Icons are from [Dashboard Icons](https://github.com/homarr-labs/dashboard-icons) and remain the property of their respective owners.

With ❤️ [DotSlash](https://github.com/mardix/dotslash)
