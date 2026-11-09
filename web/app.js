/* DotSlash — front end. Plain JavaScript, no framework, no build step. */
'use strict';

(() => {
  // ------------------------------------------------------------------
  // Small helpers
  // ------------------------------------------------------------------

  const $ = (sel, root = document) => root.querySelector(sel);

  function h(tag, props, ...kids) {
    const el = document.createElement(tag);
    if (props) {
      for (const [k, v] of Object.entries(props)) {
        if (v == null || v === false) continue;
        if (k === 'class') el.className = v;
        else if (k === 'style' && typeof v === 'object') Object.assign(el.style, v);
        else if (k.startsWith('on') && typeof v === 'function') el.addEventListener(k.slice(2).toLowerCase(), v);
        else if (k === 'html') el.innerHTML = v;
        else if (k === 'value') el.value = v;
        else if (k === 'checked' || k === 'selected' || k === 'indeterminate') el[k] = !!v;
        else if (k === 'dataset') Object.assign(el.dataset, v);
        else el.setAttribute(k, v === true ? (k === 'draggable' ? 'true' : '') : String(v));
      }
    }
    for (const c of kids.flat(Infinity)) {
      if (c == null || c === false) continue;
      el.append(c instanceof Node ? c : document.createTextNode(String(c)));
    }
    return el;
  }

  const ICONS = {
    search: '<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/>',
    pencil: '<path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z"/>',
    gear: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1Z"/>',
    plus: '<path d="M12 5v14"/><path d="M5 12h14"/>',
    sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.9 4.9 1.4 1.4"/><path d="m17.7 17.7 1.4 1.4"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m4.9 19.1 1.4-1.4"/><path d="m17.7 6.3 1.4-1.4"/>',
    moon: '<path d="M20 14.5A8 8 0 0 1 9.5 4a8 8 0 1 0 10.5 10.5Z"/>',
    grip: '<circle cx="9" cy="6" r="1.6" fill="currentColor" stroke="none"/><circle cx="15" cy="6" r="1.6" fill="currentColor" stroke="none"/><circle cx="9" cy="12" r="1.6" fill="currentColor" stroke="none"/><circle cx="15" cy="12" r="1.6" fill="currentColor" stroke="none"/><circle cx="9" cy="18" r="1.6" fill="currentColor" stroke="none"/><circle cx="15" cy="18" r="1.6" fill="currentColor" stroke="none"/>',
    dots: '<circle cx="5" cy="12" r="1.7" fill="currentColor" stroke="none"/><circle cx="12" cy="12" r="1.7" fill="currentColor" stroke="none"/><circle cx="19" cy="12" r="1.7" fill="currentColor" stroke="none"/>',
    trash: '<path d="M4 7h16"/><path d="M6 7l1 13h10l1-13"/><path d="M9 7V4h6v3"/>',
    x: '<path d="M6 6l12 12"/><path d="M18 6 6 18"/>',
    check: '<path d="M5 12l5 5 9-10"/>',
    pin: '<path d="M12 17v5"/><path d="M9 3h6l-1 6 3 3v2H7v-2l3-3Z"/>',
    copy: '<rect x="8" y="8" width="12" height="12" rx="2"/><path d="M16 8V5a1 1 0 0 0-1-1H5a1 1 0 0 0-1 1v10a1 1 0 0 0 1 1h3"/>',
    back: '<path d="M15 18l-6-6 6-6"/>',
    download: '<path d="M12 4v11"/><path d="m7 10 5 5 5-5"/><path d="M5 20h14"/>',
    upload: '<path d="M12 20V9"/><path d="m7 14 5-5 5 5"/><path d="M5 4h14"/>',
    lock: '<rect x="5" y="11" width="14" height="10" rx="2"/><path d="M8 11V7a4 4 0 0 1 8 0v4"/>',
    note: '<path d="M6 3h9l4 4v14H6Z"/><path d="M14 3v5h5"/><path d="M9 13h6"/><path d="M9 17h4"/>',
    task: '<rect x="4" y="4" width="16" height="16" rx="3"/><path d="m8.5 12 2.5 2.5 4.5-5"/>',
    calendar: '<rect x="4" y="5" width="16" height="15" rx="2"/><path d="M4 10h16"/><path d="M9 3v4"/><path d="M15 3v4"/>',
    link: '<path d="M10 14a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1 1"/><path d="M14 10a4 4 0 0 0-5.7 0l-3 3a4 4 0 0 0 5.7 5.7l1-1"/>',
    home: '<path d="M3 11 12 4l9 7"/><path d="M5 10v10h14V10"/>',
    code: '<path d="m8 8-4 4 4 4"/><path d="m16 8 4 4-4 4"/><path d="m14 4-4 16"/>',
    user: '<circle cx="12" cy="8" r="4"/><path d="M4 21a8 8 0 0 1 16 0"/>',
    pad: '<path d="M5 4h14v10l-6 6H5Z"/><path d="M13 20v-6h6"/>',
    globe: '<circle cx="12" cy="12" r="9"/><path d="M3 12h18"/><path d="M12 3a14 14 0 0 1 0 18a14 14 0 0 1 0-18"/>',
    phone: '<path d="M5 4h3l2 5-2.5 1.5a11 11 0 0 0 6 6L15 14l5 2v3a2 2 0 0 1-2 2A16 16 0 0 1 3 6a2 2 0 0 1 2-2"/>',
    mail: '<rect x="3" y="5" width="18" height="14" rx="2"/><path d="m3 7 9 6 9-6"/>',
    mappin: '<path d="M12 21s-7-6.2-7-11.5a7 7 0 0 1 14 0C19 14.8 12 21 12 21Z"/><circle cx="12" cy="9.5" r="2.5"/>',
    external: '<path d="M7 17 17 7"/><path d="M8 7h9v9"/>',
    users: '<circle cx="9" cy="8" r="3.5"/><path d="M2.5 20a6.5 6.5 0 0 1 13 0"/><path d="M16 4.6a3.5 3.5 0 0 1 0 6.8"/><path d="M18 14.2a6.5 6.5 0 0 1 3.5 5.8"/>',
    logout: '<path d="M15 4h4v16h-4"/><path d="M10 8l-4 4 4 4"/><path d="M6 12h10"/>',
    key: '<circle cx="8" cy="15" r="4"/><path d="m11 12 9-9"/><path d="m17 6 3 3"/>',
  };

  function ic(name, size = 16, label) {
    const t = document.createElement('template');
    t.innerHTML = `<svg width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" ${label ? `role="img" aria-label="${esc(label)}"` : 'aria-hidden="true"'}>${ICONS[name] || ''}</svg>`;
    return t.content.firstChild;
  }

  function esc(s) {
    return String(s).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
  }

  const clone = (o) => JSON.parse(JSON.stringify(o));
  const uid = () => Math.random().toString(16).slice(2, 10) + Date.now().toString(16).slice(-4);

  function debounce(fn, ms) {
    let t;
    const d = (...a) => { clearTimeout(t); t = setTimeout(() => fn(...a), ms); };
    d.cancel = () => clearTimeout(t);
    return d;
  }

  function hashStr(s) {
    let x = 0;
    for (let i = 0; i < s.length; i++) x = (x * 31 + s.charCodeAt(i)) | 0;
    return Math.abs(x);
  }

  function initials(name) {
    const words = String(name || '?').trim().split(/[\s._-]+/).filter(Boolean);
    if (!words.length) return '?';
    if (words.length === 1) return words[0].slice(0, 2).replace(/^./, (c) => c.toUpperCase());
    return (words[0][0] + words[1][0]).toUpperCase();
  }

  function luminance(hex) {
    const m = /^#?([0-9a-f]{6})$/i.exec(hex || '');
    if (!m) return 0.5;
    const v = parseInt(m[1], 16);
    return (0.2126 * ((v >> 16) & 255) + 0.7152 * ((v >> 8) & 255) + 0.0722 * (v & 255)) / 255;
  }

  function hexToRgba(hex, a) {
    const v = parseInt(hex.slice(1), 16);
    return `rgba(${(v >> 16) & 255}, ${(v >> 8) & 255}, ${v & 255}, ${a})`;
  }

  function safeHref(u) {
    const s = String(u || '').trim();
    if (/^(javascript|data|vbscript|file):/i.test(s.replace(/\s+/g, ''))) return '#';
    return s || '#';
  }

  function hostLabel(u) {
    return String(u || '').replace(/^[a-z]+:\/\//i, '').replace(/\/$/, '');
  }

  function timeAgo(iso) {
    const t = new Date(iso).getTime();
    if (!t) return '';
    const s = Math.round((Date.now() - t) / 1000);
    if (s < 45) return 'just now';
    const m = Math.round(s / 60);
    if (m < 60) return m + ' min ago';
    const hr = Math.round(m / 60);
    if (hr < 24) return hr + (hr === 1 ? ' hour ago' : ' hours ago');
    const d = Math.round(hr / 24);
    if (d === 1) return 'yesterday';
    if (d < 7) return d + ' days ago';
    return new Date(t).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: d > 300 ? 'numeric' : undefined });
  }

  function fmtDate(iso) {
    const t = new Date(iso);
    if (isNaN(t)) return '';
    return t.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
  }

  function fmtBytes(n) {
    if (n < 1024) return n + ' B';
    if (n < 1024 * 1024) return Math.round(n / 1024) + ' KB';
    return (n / 1048576).toFixed(1) + ' MB';
  }

  function todayStr() {
    const d = new Date();
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
  }

  function dueLabel(due) {
    if (!due) return '';
    const [y, m, d] = due.split('-').map(Number);
    const dt = new Date(y, m - 1, d);
    const diff = Math.round((dt - new Date(new Date().toDateString())) / 86400000);
    if (diff === 0) return 'Today';
    if (diff === 1) return 'Tomorrow';
    if (diff === -1) return 'Yesterday';
    return dt.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
  }

  function dueClass(due) {
    if (!due) return '';
    const t = todayStr();
    if (due < t) return 'over';
    const [y, m, d] = due.split('-').map(Number);
    const diff = (new Date(y, m - 1, d) - new Date(new Date().toDateString())) / 86400000;
    return diff <= 2 ? 'soon' : '';
  }

  // ------------------------------------------------------------------
  // Markdown (small and safe: everything is escaped first)
  // ------------------------------------------------------------------

  function mdInline(s) {
    const codes = [];
    s = s.replace(/`([^`]+)`/g, (_, c) => { codes.push(c); return `\u0000${codes.length - 1}\u0000`; });
    s = s.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, (m, text, url) => {
      const u = url.replace(/&amp;/g, '&');
      if (!/^(https?:\/\/|mailto:|\/|#)/i.test(u)) return m;
      return `<a href="${esc(u)}" target="_blank" rel="noopener noreferrer">${text}</a>`;
    });
    s = s.replace(/(^|[\s(])(https?:\/\/[^\s<)]+)/g, (m, pre, url) => `${pre}<a href="${url}" target="_blank" rel="noopener noreferrer">${url}</a>`);
    s = s.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>').replace(/__([^_]+)__/g, '<strong>$1</strong>');
    s = s.replace(/(^|[^*])\*([^*\s][^*]*)\*/g, '$1<em>$2</em>').replace(/(^|\W)_([^_\s][^_]*)_(?=\W|$)/g, '$1<em>$2</em>');
    s = s.replace(/~~([^~]+)~~/g, '<del>$1</del>');
    return s.replace(/\u0000(\d+)\u0000/g, (_, i) => `<code>${codes[+i]}</code>`);
  }

  function markdown(src) {
    const lines = esc(src || '').split('\n');
    const out = [];
    let para = [];
    let list = null; // {type, items}
    const flushPara = () => { if (para.length) { out.push(`<p>${para.map(mdInline).join('<br>')}</p>`); para = []; } };
    const flushList = () => {
      if (!list) return;
      out.push(`<${list.type}>${list.items.join('')}</${list.type}>`);
      list = null;
    };
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      if (/^\s*```/.test(line)) {
        flushPara(); flushList();
        const buf = [];
        i++;
        while (i < lines.length && !/^\s*```/.test(lines[i])) buf.push(lines[i++]);
        out.push(`<pre><code>${buf.join('\n')}</code></pre>`);
        continue;
      }
      let m;
      if ((m = /^(#{1,3})\s+(.*)$/.exec(line))) { flushPara(); flushList(); out.push(`<h${m[1].length}>${mdInline(m[2])}</h${m[1].length}>`); continue; }
      if (/^\s*(---|\*\*\*)\s*$/.test(line)) { flushPara(); flushList(); out.push('<hr>'); continue; }
      if ((m = /^&gt;\s?(.*)$/.exec(line))) { flushPara(); flushList(); out.push(`<blockquote>${mdInline(m[1])}</blockquote>`); continue; }
      if ((m = /^\s*[-*+]\s+(.*)$/.exec(line)) || (m = /^\s*\d+[.)]\s+(.*)$/.exec(line))) {
        flushPara();
        const type = /^\s*\d/.test(line) ? 'ol' : 'ul';
        if (!list || list.type !== type) { flushList(); list = { type, items: [] }; }
        const t = /^\[([ xX])\]\s+(.*)$/.exec(m[1]);
        if (t) list.items.push(`<li class="task"><input type="checkbox" disabled${t[1] !== ' ' ? ' checked' : ''}><span>${mdInline(t[2])}</span></li>`);
        else list.items.push(`<li>${mdInline(m[1])}</li>`);
        continue;
      }
      if (!line.trim()) { flushPara(); flushList(); continue; }
      flushList();
      para.push(line);
    }
    flushPara(); flushList();
    return out.join('\n');
  }

  // ------------------------------------------------------------------
  // State
  // ------------------------------------------------------------------

  const S = {
    session: null,
    settings: null,
    pages: [],
    users: null,
    notes: [],          // metadata
    noteCache: {},      // id -> full note
    tasks: [],
    directory: [],
    scratchpad: { content: '', modified_at: null },
    dirCat: 'all',
    dirSort: { key: 'name', dir: 1 },
    route: { section: 'links', id: '' },
    editing: false,
    draft: null,
    dirty: false,
    noteFilter: 'all',
    noteSort: localGet('ds-note-sort') || 'modified',
    taskFilter: 'open',
    noteEdit: null,     // id of note being edited
    about: null,
    backups: null,
    flashItem: null,
    focusAdd: null,
  };

  function localGet(k) { try { return localStorage.getItem(k); } catch (e) { return null; } }
  function localSet(k, v) { try { if (v == null) localStorage.removeItem(k); else localStorage.setItem(k, v); } catch (e) { /* ignore */ } }

  const signedIn = () => !!(S.session && S.session.signed_in);
  const canEdit = () => signedIn();
  const me = () => (S.session && S.session.me) || null;
  const isAdmin = () => !!(me() && me().role === 'admin');
  // Is this item mine? (Items without an owner are new drafts, which are mine.)
  const mine = (x) => !x.owner || (me() && x.owner === me().id);
  const isShared = (x) => x && (x.visibility === 'shared' || x.visibility === 'public');
  // A quiet marker for shared things: who shared it, or that I shared it.
  function sharedMark(x, size = 12) {
    if (!authOn() || !isShared(x)) return null;
    const pub = x.visibility === 'public';
    const who = mine(x) ? (pub ? 'Public — anyone can see it' : 'Shared with everyone signed in') : `${pub ? 'Public' : 'Shared'} by ${x.owner_name}`;
    return h('span', { class: 'smark', title: who }, ic(pub ? 'globe' : 'users', size, who), mine(x) ? null : h('span', { class: 'smark-who' }, x.owner_name));
  }
  const iconCDN = () => (S.session && S.session.icon_cdn) || 'https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons';

  // ------------------------------------------------------------------
  // API
  // ------------------------------------------------------------------

  async function api(method, path, body, opts = {}) {
    const init = { method, credentials: 'same-origin', headers: { 'X-Requested-With': 'DotSlash' } };
    if (opts.keepalive) init.keepalive = true;
    if (body !== undefined) {
      init.headers['Content-Type'] = body instanceof ArrayBuffer ? 'application/octet-stream' : 'application/json';
      init.body = opts.raw ? body : JSON.stringify(body);
    }
    let res;
    try {
      res = await fetch(path, init);
    } catch (e) {
      const err = new Error('Cannot reach the server');
      err.status = 0;
      throw err;
    }
    let data = null;
    if ((res.headers.get('content-type') || '').includes('json')) data = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((data && data.error) || `Request failed (${res.status})`);
      err.status = res.status;
      err.data = data;
      if (res.status === 401 && !opts.quiet401) onUnauthorized();
      throw err;
    }
    return data;
  }

  function onUnauthorized() {
    if (S.session) S.session.signed_in = false;
    renderDock();
    if (!$('.overlay')) openLogin();
  }

  async function loadSession() {
    S.session = await api('GET', '/api/session');
  }

  async function loadState() {
    const st = await api('GET', '/api/state');
    S.settings = st.settings;
    S.pages = st.pages;
    if (st.me && S.session) S.session.me = st.me;
    S.notes = st.notes || [];
    S.tasks = st.tasks || [];
    S.directory = st.directory || [];
    S.scratchpad = st.scratchpad || { content: '', modified_at: null };
    if (!st.signed_in) S.noteCache = {};
  }

  // Sections: Links is always on; the rest are for signed-in users and can be turned off in Settings.
  const SECTIONS = [['links', 'Links'], ['notes', 'Notes'], ['tasks', 'Tasks'], ['directory', 'Directory']];
  const authOn = () => !!(S.session && S.session.auth_enabled);
  function hiddenSet() { return new Set((S.settings && S.settings.hidden_sections) || []); }
  function sectionOn(k) {
    if (k === 'links') return true;
    if (!signedIn()) return false;
    return !hiddenSet().has(k);
  }
  const scratchOn = () => signedIn() && !hiddenSet().has('scratchpad');
  function searchPlaceholder() {
    const names = SECTIONS.filter(([k]) => sectionOn(k)).map(([, l]) => l.toLowerCase());
    if (names.length === 1) return 'Search links';
    return 'Search ' + names.slice(0, -1).join(', ') + ' and ' + names[names.length - 1];
  }

  // ------------------------------------------------------------------
  // Appearance
  // ------------------------------------------------------------------

  const mq = window.matchMedia ? window.matchMedia('(prefers-color-scheme: light)') : null;

  function resolvedTheme() {
    const o = localGet('ds-theme');
    if (o === 'dark' || o === 'light') return o;
    const t = (S.settings && S.settings.theme) || 'system';
    if (t === 'dark' || t === 'light') return t;
    return mq && mq.matches ? 'light' : 'dark';
  }

  function applyAppearance() {
    const root = document.documentElement;
    root.setAttribute('data-theme', resolvedTheme());
    if (!S.settings) return;
    const a = S.settings.accent || '#F0B45A';
    root.style.setProperty('--accent', a);
    root.style.setProperty('--on-accent', luminance(a) > 0.5 ? '#0D1220' : '#FFFFFF');
    root.style.setProperty('--accent-soft', hexToRgba(a, 0.14));
    document.title = S.settings.title || 'DotSlash';
    const hdr = $('.hdr');
    if (hdr) {
      if (S.settings.header_bg) {
        hdr.style.setProperty('--hdr-bg', S.settings.header_bg);
        hdr.style.setProperty('--hdr-fg', S.settings.header_text || (luminance(S.settings.header_bg) > 0.55 ? '#16191F' : '#F2F4F7'));
      } else {
        hdr.style.removeProperty('--hdr-bg');
        if (S.settings.header_text) hdr.style.setProperty('--hdr-fg', S.settings.header_text);
        else hdr.style.removeProperty('--hdr-fg');
      }
    }
  }

  function refreshFavicon() {
    const l = $('link[rel="icon"]');
    if (l) l.href = '/favicon?v=' + Date.now();
  }

  if (mq && mq.addEventListener) mq.addEventListener('change', () => { applyAppearance(); renderDock(); });

  // ------------------------------------------------------------------
  // Icons for links and the app
  // ------------------------------------------------------------------

  function iconSrcs(icon) {
    const v = String(icon || '').trim();
    if (!v) return [];
    if (/^https?:\/\//i.test(v) || v.startsWith('/')) return [v];
    const name = v.toLowerCase().replace(/\.(svg|png|webp)$/, '').replace(/[^a-z0-9._-]/g, '');
    if (!name) return [];
    const base = iconCDN();
    if (pngOnly.has(name)) return [`${base}/png/${name}.png`];
    return [`${base}/svg/${name}.svg`, `${base}/png/${name}.png`];
  }

  function linkIcon(link, cls = 'icon') {
    const tint = 't' + (hashStr(link.name || link.url || '') % 6);
    const box = h('span', { class: `${cls} ${tint}` }, initials(link.name || hostLabel(link.url)));
    const srcs = iconSrcs(link.icon);
    if (srcs.length) {
      let i = 0;
      // Not lazy: the image is only attached once it has loaded, and browsers never load detached lazy images.
      const img = h('img', { alt: '', decoding: 'async', referrerpolicy: 'no-referrer' });
      img.addEventListener('error', () => {
        i++;
        if (i < srcs.length) img.src = srcs[i];
        else { img.remove(); box.classList.remove('has-img'); box.textContent = initials(link.name || hostLabel(link.url)); }
      });
      img.addEventListener('load', () => { box.textContent = ''; box.append(img); box.classList.add('has-img'); });
      img.src = srcs[0];
    }
    return box;
  }

  const LOGO = '<svg viewBox="0 0 64 64" aria-hidden="true"><circle cx="22.5" cy="43" r="4.6" fill="currentColor"/><path d="M42 17 33 47" fill="none" stroke="currentColor" stroke-width="5.5" stroke-linecap="round"/></svg>';

  function appIcon(cls = 'app-icon') {
    const st = S.settings || {};
    const icon = st.icon || {};
    if (!icon.type || icon.type === 'logo' || (icon.type === 'library' && !icon.name) || (icon.type === 'image' && !icon.image)) {
      const el = h('span', { class: cls + ' logo', html: LOGO });
      if (icon.color) { el.style.background = icon.color; el.style.color = luminance(icon.color) > 0.5 ? '#0D1220' : '#FFFFFF'; }
      return el;
    }
    if (icon.type === 'image' && icon.image) return h('span', { class: cls }, h('img', { src: icon.image, alt: '' }));
    if (icon.type === 'library' && icon.name) {
      const srcs = iconSrcs(icon.name);
      if (srcs.length) {
        const el = h('span', { class: cls + ' lib' });
        const img = h('img', { src: srcs[0], alt: '', referrerpolicy: 'no-referrer' });
        let i = 0;
        img.addEventListener('error', () => {
          i++;
          if (i < srcs.length) img.src = srcs[i];
          else { el.classList.remove('lib'); el.classList.add('logo'); el.innerHTML = LOGO; }
        });
        el.append(img);
        return el;
      }
    }
    const el = h('span', { class: cls }, icon.letter || (st.title || 'H')[0]);
    if (icon.color) { el.style.background = icon.color; el.style.color = luminance(icon.color) > 0.5 ? '#0D1220' : '#FFFFFF'; }
    return el;
  }

  // ------------------------------------------------------------------
  // Icon library picker (names come from the server's embedded index)
  // ------------------------------------------------------------------

  let iconIdx = null;
  let pngOnly = new Set();

  function loadIconIndex() {
    if (!iconIdx) {
      iconIdx = fetch('/api/icons', { credentials: 'same-origin' })
        .then((r) => (r.ok ? r.json() : { svg: [], png: [] }))
        .then((d) => { pngOnly = new Set(d.png || []); return (d.svg || []).concat(d.png || []).sort(); })
        .catch(() => { iconIdx = null; return []; });
    }
    return iconIdx;
  }

  function slug(v) { return String(v || '').toLowerCase().trim().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, ''); }

  function matchIcons(all, q, limit) {
    const qs = slug(q);
    if (!qs) return [];
    const flat = qs.replace(/-/g, '');
    const scored = [];
    for (const n of all) {
      const nf = n.replace(/-/g, '');
      let sc;
      if (n === qs) sc = 0;
      else if (n.startsWith(qs) || nf.startsWith(flat)) sc = 1;
      else if (n.includes('-' + qs)) sc = 2;
      else if (nf.includes(flat)) sc = 3;
      else continue;
      if (/-(light|dark)$/.test(n)) sc += 0.5;
      scored.push([sc, n.length, n]);
    }
    scored.sort((a, b) => a[0] - b[0] || a[1] - b[1] || (a[2] < b[2] ? -1 : 1));
    return scored.slice(0, limit).map((x) => x[2]);
  }

  // Suggest an icon for a link name ("Pi-hole" -> "pi-hole"), or '' if none fits exactly.
  async function suggestIcon(name) {
    const all = await loadIconIndex();
    const sl = slug(name);
    if (!sl) return '';
    if (all.includes(sl)) return sl;
    const flat = sl.replace(/-/g, '');
    return all.find((n) => n.replace(/-/g, '') === flat) || '';
  }

  function iconImg(name) {
    const srcs = iconSrcs(name);
    const img = h('img', { alt: '', loading: 'lazy', decoding: 'async', referrerpolicy: 'no-referrer' });
    let i = 0;
    img.addEventListener('error', () => { i++; if (i < srcs.length) img.src = srcs[i]; else img.style.visibility = 'hidden'; });
    if (srcs.length) img.src = srcs[0];
    return img;
  }

  // A text field that shows matching icons (with previews) as you type.
  // onPick(value, typing) is called on every change; typing=false when chosen from the list.
  function iconPicker(value, onPick, opts = {}) {
    const input = h('input', {
      class: 'input', type: 'text', value: value || '', placeholder: opts.placeholder || 'Type to search icons, or paste an image URL',
      autocomplete: 'off', spellcheck: 'false', role: 'combobox', 'aria-expanded': 'false', 'aria-autocomplete': 'list', 'aria-label': opts.label || 'Icon',
    });
    const list = h('div', { class: 'ipick', role: 'listbox', hidden: true });
    const wrap = h('div', { class: 'ipick-wrap' }, input, list);
    let names = [];
    let active = -1;
    const close = () => { list.hidden = true; input.setAttribute('aria-expanded', 'false'); active = -1; };
    const paint = () => list.querySelectorAll('.ipick-item').forEach((el, i) => {
      el.classList.toggle('on', i === active);
      if (i === active) el.scrollIntoView({ block: 'nearest' });
    });
    const choose = (n) => { input.value = n; close(); onPick(n, false); };
    const draw = async () => {
      const q = input.value.trim();
      if (!q || /^(https?:)?\/\//i.test(q) || q.startsWith('/')) { close(); return; }
      const all = await loadIconIndex();
      if (input.value.trim() !== q || document.activeElement !== input) return;
      names = matchIcons(all, q, 30);
      active = -1;
      list.replaceChildren(...(names.length
        ? names.map((n) => h('button', { type: 'button', class: 'ipick-item' + (n === slug(q) ? ' exact' : ''), role: 'option', title: n, tabindex: '-1', onmousedown: (e) => e.preventDefault(), onclick: () => choose(n) },
          iconImg(n), h('span', null, n)))
        : [h('div', { class: 'ipick-empty' }, all.length ? 'No matching icons. Any image URL works too.' : 'The icon list is unavailable.')]));
      list.hidden = false;
      input.setAttribute('aria-expanded', 'true');
    };
    const drawSoon = debounce(draw, 110);
    input.addEventListener('input', () => { onPick(input.value.trim(), true); drawSoon(); });
    input.addEventListener('focus', drawSoon);
    input.addEventListener('blur', () => setTimeout(close, 150));
    input.addEventListener('keydown', (e) => {
      if (e.key === 'Escape' && !list.hidden) { e.preventDefault(); close(); return; }
      if (list.hidden || !names.length) return;
      const cols = Math.max(1, getComputedStyle(list).gridTemplateColumns.split(' ').length);
      const move = { ArrowRight: 1, ArrowLeft: -1, ArrowDown: cols, ArrowUp: -cols }[e.key];
      if (move) {
        if ((e.key === 'ArrowLeft' || e.key === 'ArrowRight') && active < 0) return; // let the caret move
        e.preventDefault();
        active = active < 0 ? 0 : Math.max(0, Math.min(names.length - 1, active + move));
        paint();
      } else if (e.key === 'Enter' && active >= 0) {
        e.preventDefault();
        choose(names[active]);
      }
    });
    loadIconIndex();
    return { el: wrap, input, set: (v) => { input.value = v; } };
  }

  // ------------------------------------------------------------------
  // Routing
  // ------------------------------------------------------------------

  function parseHash() {
    const parts = location.hash.replace(/^#\/?/, '').split('/');
    const section = ['links', 'notes', 'tasks', 'directory', 'settings', 'blank', 'login'].includes(parts[0]) ? parts[0] : '';
    return { section, id: decodeURIComponent(parts[1] || '') };
  }

  function startHash() {
    const so = (S.settings && S.settings.start_on) || 'links';
    if (so === 'last' && signedIn()) return localGet('ds-last') || '#/links';
    if (so === 'blank') return '#/blank';
    if (['notes', 'tasks', 'directory'].includes(so)) return sectionOn(so) ? '#/' + so : '#/links';
    if (so.startsWith('links:')) {
      const id = so.slice(6);
      if (S.pages.some((p) => p.id === id)) return '#/links/' + id;
    }
    return '#/links';
  }

  function go(hash, replace) {
    if (location.hash === hash) { onRoute(); return; }
    if (replace) { history.replaceState(null, '', hash); onRoute(); } else location.hash = hash;
  }

  function onRoute() {
    const r = parseHash();
    if (!r.section) { go(startHash(), true); return; }
    if (r.section === 'login') {
      history.replaceState(null, '', S.hash || startHash());
      onRoute();
      if (!signedIn()) openLogin();
      return;
    }
    if (signedIn() && !sectionOn(r.section) && ['notes', 'tasks', 'directory'].includes(r.section)) { go(startHash(), true); return; }
    if (S.editing) {
      // While editing, nothing else can be opened until Save or Close.
      // Switching between link pages is part of editing them, so that's allowed.
      const samePlace = r.section === S.route.section && (r.section === 'links' || r.id === S.route.id);
      if (!samePlace) {
        history.replaceState(null, '', S.hash || '#/' + S.route.section);
        toast('Save or close your changes first.');
        return;
      }
    }
    if (S.noteEdit && (r.section !== 'notes' || r.id !== S.noteEdit)) { flushNote(); S.noteEdit = null; }
    if (r.section === 'settings') S.scrollSettings = true;
    S.route = r;
    S.hash = location.hash;
    if (r.section !== 'settings') localSet('ds-last', location.hash);
    if (r.section === 'links' && r.id) localSet('ds-last-page', r.id);
    closeSearch();
    renderAll();
    window.scrollTo(0, 0);
    if (r.section === 'settings' && canEdit()) loadSettingsExtras();
  }

  window.addEventListener('hashchange', onRoute);

  // ------------------------------------------------------------------
  // Shell
  // ------------------------------------------------------------------

  let shell = null;

  function buildShell() {
    const app = $('#app');
    app.replaceChildren();
    const searchInput = h('input', {
      type: 'search', id: 'q', autocomplete: 'off', spellcheck: 'false',
      placeholder: 'Search links, notes and tasks',
      'aria-label': 'Search links, notes and tasks',
      'aria-controls': 'results',
    });
    const results = h('div', { class: 'results', id: 'results', role: 'listbox', hidden: true });
    shell = {
      hdr: h('header', { class: 'hdr' }),
      brand: h('a', { class: 'brand', href: '#' }),
      nav: h('nav', { class: 'sections', 'aria-label': 'Sections' }),
      clock: h('div', { class: 'clock' }, h('span', { class: 'clock-date' }), h('span', { class: 'clock-time' })),
      who: h('a', { class: 'who', href: '#/settings/profile' }),
      search: h('div', { class: 'search' },
        h('label', { class: 'search-box' }, ic('search', 16), searchInput, h('kbd', null, '/')),
        results),
      sub: h('div', { class: 'sub' }),
      main: h('main', { class: 'main', id: 'main' }),
      dock: h('div', { class: 'dock', role: 'toolbar', 'aria-label': 'Dashboard controls' }),
      toasts: h('div', { class: 'toasts', 'aria-live': 'polite' }),
      footer: footerEl(),
      input: searchInput,
      results,
      pad: h('section', { class: 'pad', 'aria-label': 'Scratchpad', hidden: true }),
    };
    shell.brand.addEventListener('click', (e) => { e.preventDefault(); if (!S.editing) go(startHash()); });
    shell.hdr.append(h('div', { class: 'hdr-in' },
      h('div', { class: 'hdr-row' }, shell.brand, shell.nav, h('div', { class: 'hdr-end' }, shell.clock, shell.who)),
      shell.search,
      shell.sub));
    app.append(shell.hdr, shell.main, shell.footer, shell.pad, shell.dock, shell.toasts);
    setupSearch();
    tickClock();
    clearInterval(clockTimer);
    clockTimer = setInterval(tickClock, 10000);
  }
  let clockTimer = null;
  document.addEventListener('click', (e) => { if (shell && !shell.search.contains(e.target)) closeSearch(true); });

  function tickClock() {
    if (!shell) return;
    const now = new Date();
    shell.clock.firstChild.textContent = now.toLocaleDateString(undefined, { weekday: 'short', month: 'short', day: 'numeric' });
    shell.clock.lastChild.textContent = now.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
  }

  function renderAll() {
    applyAppearance();
    renderHeader();
    renderMain();
    renderDock();
    renderPad();
  }

  function renderHeader() {
    shell.hdr.classList.toggle('locked', S.editing);
    shell.input.disabled = S.editing;
    shell.input.placeholder = S.editing ? 'Search is off while editing' : searchPlaceholder();
    shell.input.setAttribute('aria-label', shell.input.placeholder);
    shell.brand.setAttribute('aria-disabled', String(S.editing));
    shell.brand.replaceChildren(appIcon(), h('span', { class: 'brand-name' }, S.settings.title));
    const m = me();
    shell.who.hidden = !(m && authOn() && signedIn());
    if (!shell.who.hidden) {
      shell.who.title = `Signed in as ${m.username}${m.role === 'admin' ? ' (admin)' : ''} — your profile`;
      shell.who.setAttribute('aria-label', `${m.display}, your profile`);
      shell.who.replaceChildren(h('span', { class: 'who-avatar', 'aria-hidden': 'true' }, initials(m.display)), h('span', { class: 'who-name' }, m.display));
    }
    const sec = S.route.section;
    const lastPage = localGet('ds-last-page');
    const linksHash = '#/links' + (lastPage && pagesView().some((p) => p.id === lastPage) ? '/' + lastPage : '');
    const visible = SECTIONS.filter(([k]) => sectionOn(k));
    // Visitors only have Links, so the section tabs are left out entirely.
    shell.nav.replaceChildren(...(visible.length < 2 ? [] : visible.map(([k, label]) =>
      h('a', { class: 'tab' + (sec === k ? ' on' : ''), href: k === 'links' ? linksHash : '#/' + k, 'aria-current': sec === k ? 'page' : null }, label))));
    renderSub();
  }

  function renderSub() {
    const sec = S.route.section;
    const sub = shell.sub;
    sub.replaceChildren();
    if (sec === 'blank' || sec === 'settings') { sub.append(h('div', { class: 'hdr-spacer' })); return; }
    const tabs = h('nav', { class: 'sub-tabs', 'aria-label': sec === 'links' ? 'Pages' : 'Filters' });
    sub.append(tabs);
    if (sec === 'links') {
      const cur = currentPage();
      pagesView().forEach((p, i) => tabs.append(pageTab(p, i, cur && p.id === cur.id)));
      return;
    }
    if (sec === 'notes') {
      if (!signedIn()) return;
      const count = (f) => S.notes.filter(f).length;
      const filters = [['all', 'All', S.notes.length], ['note', 'Notes', count((n) => n.type !== 'snippet')], ['snippet', 'Snippets', count((n) => n.type === 'snippet')], ['pinned', 'Pinned', count((n) => n.pinned)]];
      if (authOn()) filters.push(['shared', 'Shared', count(isShared)]);
      filters.forEach(([k, label, n]) => {
        tabs.append(h('button', { class: 'tab' + (S.noteFilter === k ? ' on' : ''), 'aria-pressed': String(S.noteFilter === k), onclick: () => {
          S.noteFilter = k;
          if (S.noteEdit) { flushNote(); S.noteEdit = null; }
          // The Snippets filter is a grid of copy cards, so it doesn't keep a note open.
          if (k === 'snippet' || (S.route.id && !sortedNotes().some((x) => x.id === S.route.id))) { renderSub(); renderDock(); go('#/notes'); return; }
          renderSub(); renderMain(); renderDock();
        } }, label, h('span', { class: 'count' }, n)));
      });
      const sel = h('select', { 'aria-label': 'Sort notes', onchange: (e) => { S.noteSort = e.target.value; localSet('ds-note-sort', S.noteSort); renderMain(); } },
        h('option', { value: 'modified', selected: S.noteSort === 'modified' }, 'Recently edited'),
        h('option', { value: 'title', selected: S.noteSort === 'title' }, 'Title A–Z'),
        h('option', { value: 'created', selected: S.noteSort === 'created' }, 'Newest first'));
      sub.append(h('label', { class: 'sub-tools' }, h('span', null, 'Sort'), sel));
      return;
    }
    if (sec === 'directory') {
      if (!signedIn()) return;
      const counts = new Map();
      S.directory.forEach((x) => { const k = (x.category || '').trim(); counts.set(k, (counts.get(k) || 0) + 1); });
      const named = [...counts.keys()].filter(Boolean).sort((a, b) => a.localeCompare(b));
      if (S.dirCat !== 'all' && S.dirCat !== '~shared' && !counts.has(S.dirCat === '-' ? '' : S.dirCat)) S.dirCat = 'all';
      const opts = [['all', 'All', S.directory.length], ...named.map((n) => [n, n, counts.get(n)])];
      if (named.length && counts.get('')) opts.push(['-', 'Uncategorized', counts.get('')]);
      const nShared = S.directory.filter(isShared).length;
      if (authOn() && (nShared || S.dirCat === '~shared')) opts.push(['~shared', 'Shared', nShared]);
      opts.forEach(([k, label, n]) => tabs.append(h('button', { class: 'tab' + (S.dirCat === k ? ' on' : ''), 'aria-pressed': String(S.dirCat === k), onclick: () => { S.dirCat = k; renderSub(); renderMain(); } },
        label, h('span', { class: 'count' }, n))));
      return;
    }
    if (sec === 'tasks') {
      if (!signedIn()) return;
      let open = 0, done = 0;
      S.tasks.forEach((l) => l.items.forEach((it) => (it.done ? done++ : open++)));
      const tf = [['open', 'Open', open], ['all', 'All', open + done], ['done', 'Done', done]];
      if (authOn()) tf.push(['shared', 'Shared lists', S.tasks.filter(isShared).length]);
      tf.forEach(([k, label, n]) => {
        tabs.append(h('button', { class: 'tab' + (S.taskFilter === k ? ' on' : ''), 'aria-pressed': String(S.taskFilter === k), onclick: () => { S.taskFilter = k; renderSub(); renderMain(); } },
          label, h('span', { class: 'count' }, n)));
      });
    }
  }

  // ------------------------------------------------------------------
  // Dock
  // ------------------------------------------------------------------

  function renderDock() {
    if (!shell) return;
    const dark = resolvedTheme() === 'dark';
    const sec = S.route.section;
    const themeBtn = h('button', {
      class: 'dock-btn icon-only', 'aria-label': dark ? 'Switch to light theme' : 'Switch to dark theme', title: dark ? 'Light theme' : 'Dark theme',
      onclick: () => { localSet('ds-theme', dark ? 'light' : 'dark'); applyAppearance(); renderDock(); },
    }, ic(dark ? 'sun' : 'moon', 14));
    const kids = [themeBtn, h('span', { class: 'sep' })];
    if (!signedIn()) {
      // Visitors: just the theme and a quiet way to sign in.
      kids.push(h('button', { class: 'dock-btn icon-only', 'aria-label': 'Sign in', title: 'Sign in', onclick: () => openLogin() }, ic('lock', 15)));
      shell.dock.replaceChildren(...kids);
      return;
    }
    if (S.editing) {
      const addLabel = sec === 'notes' ? (S.noteFilter === 'snippet' ? 'Snippet' : 'Note') : ({ tasks: 'List', directory: 'Entry' }[sec] || 'Page');
      kids.push(
        h('button', { class: 'dock-btn', onclick: dockAdd }, ic('plus', 14), addLabel),
        h('button', { class: 'dock-btn muted', onclick: closeEdit }, 'Close'),
        h('button', { class: 'dock-btn primary', onclick: saveEdit }, 'Save'));
    } else {
      kids.push(
        h('button', { class: 'dock-btn', disabled: sec === 'settings', onclick: startEdit, title: 'Edit (E)' }, ic('pencil', 14), 'Edit'),
        scratchOn() ? h('button', { class: 'dock-btn icon-only' + (padOpen() ? ' active' : ''), 'aria-label': padOpen() ? 'Hide scratchpad' : 'Show scratchpad', 'aria-pressed': String(padOpen()), title: 'Scratchpad', onclick: togglePad }, ic('pad', 15)) : null,
        h('a', { class: 'dock-btn icon-only' + (sec === 'settings' ? ' active' : ''), href: '#/settings', 'aria-label': 'Settings', 'aria-current': sec === 'settings' ? 'page' : null, title: 'Settings', onclick: (e) => { if (!canEdit()) { e.preventDefault(); openLogin(() => go('#/settings')); } } }, ic('gear', 15)));
    }
    shell.dock.replaceChildren(...kids.filter(Boolean));
  }

  function startEdit() {
    if (!canEdit()) { openLogin(startEdit); return; }
    let sec = S.route.section;
    if (sec === 'blank' || sec === 'settings') {
      sec = 'links';
      S.route = { section: 'links', id: '' };
      location.hash = '#/links';
    }
    if (sec === 'links') { S.draft = clone(S.pages); S.dirty = false; }
    if ((sec === 'notes' || sec === 'tasks') && !signedIn()) return;
    S.editing = true;
    renderAll();
  }

  async function saveEdit() {
    await dropEmptyEntries();
    if (S.draft) {
      if (S.dirty) {
        try {
          const res = await api('PUT', '/api/layout', { pages: S.draft.map((p) => ({ id: p.id, name: p.name, visibility: p.visibility, groups: p.groups, rev: p.rev })) });
          S.pages = res.pages;
          if (res.conflicts && res.conflicts.length) {
            const names = res.conflicts.map((n) => `“${n}”`).join(', ');
            toast(`${names} ${res.conflicts.length === 1 ? 'was' : 'were'} changed by someone else while you edited, so your changes there weren’t saved. The rest is saved.`, 'err');
          } else toast('Saved');
        } catch (e) {
          if (e.status !== 401) toast(e.message, 'err');
          return;
        }
      }
    }
    S.editing = false; S.draft = null; S.dirty = false;
    const r = parseHash();
    if (r.section === 'links' && r.id && !S.pages.some((p) => p.id === r.id)) go('#/links', true);
    else renderAll();
  }

  async function closeEdit() {
    if (S.draft && S.dirty) {
      const ok = await confirmBox('Discard changes?', 'Your changes to pages, groups and links will be lost.', 'Discard', true);
      if (!ok) return;
    }
    await dropEmptyEntries();
    S.editing = false; S.draft = null; S.dirty = false;
    const r = parseHash();
    if (r.section === 'links' && r.id && !S.pages.some((p) => p.id === r.id)) go('#/links', true);
    else renderAll();
  }

  async function dockAdd() {
    const sec = S.route.section;
    if (sec === 'notes') return createNote(S.noteFilter === 'snippet' ? 'snippet' : 'note');
    if (sec === 'tasks') return createList();
    if (sec === 'directory') return addEntry();
    const name = await promptBox('New page', 'Page name', '', 'Create');
    if (!name) return;
    // New pages start private so nothing becomes visible by accident.
    const p = { id: uid(), name, visibility: 'private', groups: [], rev: 0, owner: me() && me().id, owner_name: '', can_edit: true, can_manage: true };
    S.draft.push(p);
    S.dirty = true;
    go('#/links/' + p.id);
  }

  // ------------------------------------------------------------------
  // Links
  // ------------------------------------------------------------------

  function pagesView() { return S.editing && S.draft ? S.draft : S.pages; }

  function currentPage() {
    const pages = pagesView();
    return pages.find((p) => p.id === S.route.id) || pages[0];
  }

  let drag = null;

  function pageTab(p, index, active) {
    const editing = S.editing;
    const mark = signedIn() ? sharedMark(p, 11) : null;
    const tab = h('a', { class: 'tab' + (active ? ' on' : ''), href: '#/links/' + p.id, 'aria-current': active ? 'page' : null, draggable: 'false' },
      index === 0 && editing ? ic('home', 13) : null, p.name, mark ? h('span', { class: 'tab-lock' }, mark) : null);
    const wrap = h('span', { class: 'ptab' + (editing ? ' editing' : '') }, editing ? h('span', { class: 'grip' }, ic('grip', 11)) : null, tab);
    if (editing) {
      wrap.append(h('button', { class: 'ptab-menu', 'aria-label': 'Page options for ' + p.name, onclick: (e) => pageMenu(e, p, index) }, ic('dots', 13)));
      wrap.draggable = true;
      wrap.addEventListener('dragstart', (e) => { drag = { type: 'page', from: index }; e.dataTransfer.effectAllowed = 'move'; e.dataTransfer.setData('text/plain', 'page'); });
      wrap.addEventListener('dragend', () => { drag = null; clearDropMarks(); });
      wrap.addEventListener('dragover', (e) => {
        if (!drag || drag.type !== 'page') return;
        e.preventDefault();
        clearDropMarks();
        const r = wrap.getBoundingClientRect();
        wrap.classList.add(e.clientX < r.left + r.width / 2 ? 'drop-before' : 'drop-after');
      });
      wrap.addEventListener('drop', (e) => {
        if (!drag || drag.type !== 'page') return;
        e.preventDefault();
        const r = wrap.getBoundingClientRect();
        let to = index + (e.clientX < r.left + r.width / 2 ? 0 : 1);
        if (drag.from < to) to--;
        moveIn(S.draft, drag.from, to);
        S.dirty = true;
        drag = null;
        renderAll();
      });
    }
    return wrap;
  }

  function clearDropMarks() {
    document.querySelectorAll('.drop-before,.drop-after,.drop-target').forEach((el) => el.classList.remove('drop-before', 'drop-after', 'drop-target'));
  }

  function moveIn(arr, from, to) {
    if (from === to || from < 0) return;
    const [x] = arr.splice(from, 1);
    arr.splice(Math.max(0, Math.min(to, arr.length)), 0, x);
  }

  const VIS_LABEL = { private: 'Private — only me', shared: 'Shared — everyone signed in', public: 'Public — anyone, no sign-in' };

  function pageMenu(e, p, index) {
    e.preventDefault();
    e.stopPropagation();
    const setVis = (v) => () => { p.visibility = v; S.dirty = true; renderAll(); };
    const visItems = !authOn() ? [] : !p.can_manage ? [[`Shared by ${p.owner_name}`, () => toast(p.can_edit ? 'You can edit this page; only its owner or an admin can change who sees it.' : 'Only its owner or an admin can change this page.')]]
      : ['private', 'shared', 'public'].filter((v) => v !== 'public' || isAdmin()).map((v) => [(p.visibility === v ? '✓ ' : '\u2003') + VIS_LABEL[v], setVis(v)]);
    openMenu(e.currentTarget, [
      p.can_edit ? ['Rename', async () => { const n = await promptBox('Rename page', 'Page name', p.name, 'Rename'); if (n) { p.name = n; S.dirty = true; renderAll(); } }] : null,
      ...visItems,
      visItems.length ? '-' : null,
      index > 0 ? ['Move to the front', () => { moveIn(S.draft, index, 0); S.dirty = true; renderAll(); }] : null,
      ['Duplicate as my page', () => {
        const c = clone(p);
        Object.assign(c, { id: uid(), name: p.name + ' copy', visibility: 'private', owner: me() && me().id, owner_name: '', can_edit: true, can_manage: true, rev: 0 });
        c.groups.forEach((g) => { g.id = uid(); g.links.forEach((l) => { l.id = uid(); }); });
        S.draft.splice(index + 1, 0, c); S.dirty = true; go('#/links/' + c.id);
      }],
      p.can_manage ? '-' : null,
      !p.can_manage ? null : ['Delete page…', async () => {
        if (S.draft.length <= 1) { toast('You need at least one page.', 'err'); return; }
        const links = p.groups.reduce((n, g) => n + g.links.length, 0);
        const ok = await confirmBox(`Delete “${p.name}”?`, links ? `It has ${links} link${links === 1 ? '' : 's'} in ${p.groups.length} group${p.groups.length === 1 ? '' : 's'}.` : 'The page is empty.', 'Delete', true);
        if (!ok) return;
        S.draft.splice(index, 1); S.dirty = true; go('#/links/' + S.draft[0].id);
      }, true],
    ]);
  }

  function renderLinks() {
    const page = currentPage();
    // Shared pages the person can't edit stay read-only in edit mode.
    const editing = S.editing && !!page && page.can_edit !== false;
    const layout = (S.settings && S.settings.layout) || 'cards';
    const root = h('div', { class: `wrap layout-${layout}${editing ? ' editing' : ''}` });
    if (!signedIn() && S.route.id && !pagesView().some((p) => p.id === S.route.id)) return lockedView('This page is private', 'Sign in to view it.');
    if (!page) {
      root.append(h('div', { class: 'empty' }, ic('link', 22), h('h2', null, 'No pages yet'),
        h('p', null, canEdit() ? 'Click Edit in the corner, then + Page to make your first page.' : 'Nothing is public here yet.')));
      return root;
    }
    if (S.editing && !editing) {
      root.append(h('div', { class: 'notice readonly' }, ic('users', 14), h('span', null, `“${page.name}” is shared by ${page.owner_name} — read-only. You can still reorder it among your pages or duplicate it as your own.`)));
    }
    if (!page.groups.length && !editing) {
      root.append(h('div', { class: 'empty' },
        ic('link', 22), h('h2', null, 'No links on this page yet'),
        h('p', null, canEdit() ? (page.can_edit === false ? 'This shared page is empty.' : 'Click Edit in the corner to add groups and links.') : 'Sign in and click Edit to add groups and links.')));
      return root;
    }
    const grid = h('div', { class: 'grid' });
    page.groups.forEach((g, gi) => grid.append(groupEl(page, g, gi)));
    if (editing) {
      grid.append(h('button', { class: 'add-btn add-group', onclick: async () => {
        const n = await promptBox('New group', 'Group name', '', 'Add');
        if (n) { page.groups.push({ id: uid(), name: n, links: [] }); S.dirty = true; renderMain(); }
      } }, ic('plus', 16), 'Add group'));
    }
    root.append(grid);
    return root;
  }

  function groupEl(page, g, gi) {
    const editing = S.editing && page.can_edit !== false;
    const sec = h('section', { class: 'group', 'aria-label': g.name });
    const head = h('div', { class: 'group-head' },
      editing ? h('span', { class: 'grip', title: 'Drag to move group' }, ic('grip', 12)) : null,
      h('h2', { class: 'group-name' }, g.name),
      editing
        ? h('button', { class: 'icon-btn', 'aria-label': 'Group options for ' + g.name, onclick: (e) => groupMenu(e, page, g, gi) }, ic('dots', 14))
        : h('span', { class: 'group-count' }, g.links.length || ''));
    const list = h('div', { class: 'links' });
    g.links.forEach((l, li) => list.append(editing ? linkEditEl(page, g, gi, l, li) : linkEl(l)));
    sec.append(head, list);
    if (editing) {
      sec.append(h('button', { class: 'add-btn', onclick: () => openLinkModal(page, gi, -1) }, ic('plus', 14), 'Add link'));
      sec.draggable = true;
      sec.addEventListener('dragstart', (e) => {
        if (drag) return; // a link drag already started
        drag = { type: 'group', page: page.id, from: gi };
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/plain', 'group');
      });
      sec.addEventListener('dragend', () => { drag = null; clearDropMarks(); });
      sec.addEventListener('dragover', (e) => {
        if (!drag) return;
        if (drag.type === 'group' || drag.type === 'link') { e.preventDefault(); }
        if (drag.type === 'group' && drag.from !== gi) { clearDropMarks(); sec.classList.add('drop-target'); }
        if (drag.type === 'link' && !e.target.closest('.link')) { clearDropMarks(); sec.classList.add('drop-target'); }
      });
      sec.addEventListener('drop', (e) => {
        if (!drag) return;
        e.preventDefault();
        if (drag.type === 'group') {
          moveIn(page.groups, drag.from, gi);
        } else if (drag.type === 'link' && !e.target.closest('.link')) {
          const src = page.groups[drag.gi];
          const [l] = src.links.splice(drag.li, 1);
          g.links.push(l);
        } else return;
        S.dirty = true; drag = null; renderMain();
      });
    }
    return sec;
  }

  function groupMenu(e, page, g, gi) {
    e.stopPropagation();
    openMenu(e.currentTarget, [
      ['Rename', async () => { const n = await promptBox('Rename group', 'Group name', g.name, 'Rename'); if (n) { g.name = n; S.dirty = true; renderMain(); } }],
      ['Add link', () => openLinkModal(page, gi, -1)],
      '-',
      ['Delete group…', async () => {
        const ok = await confirmBox(`Delete “${g.name}”?`, g.links.length ? `Its ${g.links.length} link${g.links.length === 1 ? '' : 's'} will be removed too.` : 'The group is empty.', 'Delete', true);
        if (ok) { page.groups.splice(gi, 1); S.dirty = true; renderMain(); }
      }, true],
    ]);
  }

  function linkEl(l) {
    return h('a', { class: 'link', href: safeHref(l.url), target: l.new_tab ? '_blank' : null, rel: l.new_tab ? 'noopener noreferrer' : 'noreferrer', title: l.description || l.url },
      linkIcon(l),
      h('span', { class: 'link-text' }, h('span', { class: 'link-name' }, l.name), h('span', { class: 'link-host' }, l.description || hostLabel(l.url))));
  }

  function linkEditEl(page, g, gi, l, li) {
    const el = h('div', { class: 'link link-edit', draggable: true },
      h('span', { class: 'grip' }, ic('grip', 12)),
      linkIcon(l),
      h('span', { class: 'link-text' }, h('span', { class: 'link-name' }, l.name), h('span', { class: 'link-host' }, hostLabel(l.url))),
      h('button', { class: 'icon-btn edit-btn', 'aria-label': 'Edit ' + l.name, onclick: () => openLinkModal(page, gi, li) }, ic('pencil', 14)));
    el.addEventListener('dblclick', () => openLinkModal(page, gi, li));
    el.addEventListener('dragstart', (e) => {
      e.stopPropagation();
      drag = { type: 'link', gi, li };
      e.dataTransfer.effectAllowed = 'move';
      e.dataTransfer.setData('text/plain', 'link');
      requestAnimationFrame(() => el.classList.add('dragging'));
    });
    el.addEventListener('dragend', () => { el.classList.remove('dragging'); drag = null; clearDropMarks(); });
    el.addEventListener('dragover', (e) => {
      if (!drag || drag.type !== 'link') return;
      e.preventDefault();
      e.stopPropagation();
      clearDropMarks();
      const r = el.getBoundingClientRect();
      const tiles = el.closest('.layout-tiles');
      const before = tiles ? e.clientX < r.left + r.width / 2 : e.clientY < r.top + r.height / 2;
      el.classList.add(before ? 'drop-before' : 'drop-after');
    });
    el.addEventListener('drop', (e) => {
      if (!drag || drag.type !== 'link') return;
      e.preventDefault();
      e.stopPropagation();
      const r = el.getBoundingClientRect();
      const tiles = el.closest('.layout-tiles');
      const before = tiles ? e.clientX < r.left + r.width / 2 : e.clientY < r.top + r.height / 2;
      const src = page.groups[drag.gi];
      const [moved] = src.links.splice(drag.li, 1);
      let to = li + (before ? 0 : 1);
      if (drag.gi === gi && drag.li < to) to--;
      g.links.splice(to, 0, moved);
      S.dirty = true; drag = null; renderMain();
    });
    return el;
  }

  function openLinkModal(page, gi, li) {
    const isNew = li < 0;
    const g = page.groups[gi];
    const l = isNew ? { id: uid(), name: '', url: '', icon: '', description: '', new_tab: S.settings.new_tab } : clone(g.links[li]);
    const target = { page: page.id, group: g.id };
    const preview = h('div', { class: 'preview-row' });
    const name = h('input', { class: 'input', type: 'text', value: l.name, placeholder: 'Jellyfin', required: true });
    const url = h('input', { class: 'input mono', type: 'text', value: l.url, placeholder: 'http://192.168.1.10:8096', inputmode: 'url' });
    let iconVal = l.icon;
    let iconTouched = !!l.icon;
    const picker = iconPicker(l.icon, (v, typing) => { iconVal = v; iconTouched = !!v; if (typing) refreshSoon(); else refresh(); }, { placeholder: 'Search icons, e.g. jellyfin — or paste an image URL' });
    const icon = picker.input;
    const autoIcon = async () => {
      if (iconTouched) return;
      const n = await suggestIcon(name.value);
      if (n && !iconTouched) { iconVal = n; picker.set(n); refresh(); }
    };
    const desc = h('input', { class: 'input', type: 'text', value: l.description, placeholder: 'Optional; shown instead of the address' });
    const newTab = h('input', { type: 'checkbox', checked: l.new_tab });
    const where = h('select', { class: 'select' });
    pagesView().filter((p) => p.can_edit !== false).forEach((p) => {
      const og = h('optgroup', { label: p.name });
      p.groups.forEach((gg) => og.append(h('option', { value: p.id + '|' + gg.id, selected: p.id === target.page && gg.id === target.group }, gg.name)));
      if (p.groups.length) where.append(og);
    });
    const refresh = () => {
      const cur = { name: name.value.trim() || hostLabel(url.value) || 'New link', url: url.value, icon: iconVal };
      preview.replaceChildren(linkIcon(cur), h('span', { class: 'link-text' }, h('span', { class: 'link-name' }, cur.name), h('span', { class: 'link-host' }, desc.value || hostLabel(url.value) || 'Preview')));
    };
    const refreshSoon = debounce(refresh, 250);
    [name, url, desc].forEach((i) => i.addEventListener('input', refresh));
    name.addEventListener('change', autoIcon);
    url.addEventListener('blur', () => {
      const v = url.value.trim();
      if (v && !/^[a-z][a-z0-9+.-]*:/i.test(v)) url.value = 'http://' + v;
      if (!name.value.trim() && url.value) {
        try {
          const host = new URL(url.value).hostname;
          if (!/^[\d.]+$/.test(host) && !host.includes(':')) {
            const n = host.split('.').filter((x) => !/^(www|lan|local|home|internal)$/.test(x))[0] || '';
            name.value = n.charAt(0).toUpperCase() + n.slice(1);
          }
        } catch (e) { /* ignore */ }
      }
      refresh();
      autoIcon();
    });
    refresh();
    const body = [
      preview,
      h('label', { class: 'field' }, 'Name', name),
      h('label', { class: 'field' }, 'URL', url),
      h('div', { class: 'field' }, h('span', null, 'Icon'), picker.el, h('span', { class: 'hint' }, 'Pick from ', h('a', { href: 'https://dashboardicons.com', target: '_blank', rel: 'noopener noreferrer' }, 'Dashboard Icons'), ', paste an image URL, or leave empty for initials.')),
      h('label', { class: 'field' }, 'Description', desc),
      h('div', { class: 'row2' }, h('label', { class: 'field' }, 'Group', where), h('label', { class: 'check', style: { alignSelf: 'end', height: '40px' } }, newTab, 'Open in new tab')),
    ];
    const err = h('p', { class: 'form-error', role: 'alert' });
    body.push(err);
    openModal(isNew ? 'Add link' : 'Edit link', body, [
      !isNew ? { label: 'Delete', cls: 'danger', left: true, run: () => { g.links.splice(li, 1); S.dirty = true; renderMain(); return true; } } : null,
      { label: 'Cancel', run: () => true },
      { label: isNew ? 'Add link' : 'Save link', cls: 'primary', submit: true, run: () => {
        if (!url.value.trim()) { err.textContent = 'Add a URL.'; url.focus(); return false; }
        const n = { ...l, name: name.value.trim() || hostLabel(url.value), url: url.value.trim(), icon: iconVal.trim(), description: desc.value.trim(), new_tab: newTab.checked };
        const [pid, gid] = where.value.split('|');
        const tp = pagesView().find((p) => p.id === pid) || page;
        const tg = tp.groups.find((x) => x.id === gid) || g;
        if (isNew) tg.links.push(n);
        else if (tg === g) g.links[li] = n;
        else { g.links.splice(li, 1); tg.links.push(n); }
        S.dirty = true;
        renderMain();
        return true;
      } },
    ], isNew ? url : name);
  }

  // ------------------------------------------------------------------
  // Notes
  // ------------------------------------------------------------------

  function sortedNotes() {
    let list = S.notes.slice();
    if (S.noteFilter === 'pinned') list = list.filter((n) => n.pinned);
    else if (S.noteFilter === 'snippet') list = list.filter((n) => n.type === 'snippet');
    else if (S.noteFilter === 'note') list = list.filter((n) => n.type !== 'snippet');
    else if (S.noteFilter === 'shared') list = list.filter(isShared);
    const by = {
      modified: (a, b) => (b.modified_at > a.modified_at ? 1 : -1),
      created: (a, b) => (b.created_at > a.created_at ? 1 : -1),
      title: (a, b) => a.title.localeCompare(b.title),
    }[S.noteSort] || ((a, b) => (b.modified_at > a.modified_at ? 1 : -1));
    list.sort((a, b) => (a.pinned === b.pinned ? by(a, b) : a.pinned ? -1 : 1));
    return list;
  }

  function lockedView(title, msg) {
    return h('div', { class: 'wrap' }, h('div', { class: 'empty' },
      ic('lock', 22), h('h2', null, title),
      h('p', null, msg || 'Sign in to see it.'),
      h('button', { class: 'btn primary', onclick: () => openLogin() }, 'Sign in')));
  }

  let notePane = null;
  let noteListPane = null;

  function renderNotes() {
    if (!signedIn()) return lockedView('Notes are private', 'Sign in to see them.');
    const list = sortedNotes();
    let id = S.route.id;
    const wrap = h('div', { class: 'wrap' });
    if (S.noteFilter === 'snippet' && !id) return renderSnippetGrid(wrap, list);
    const wide = window.innerWidth > 760;
    if (!id && wide && list.length) { id = list[0].id; history.replaceState(null, '', '#/notes/' + id); S.route.id = id; S.hash = location.hash; }
    if (!S.notes.length) {
      wrap.append(h('div', { class: 'empty' }, ic('note', 22), h('h2', null, 'No notes yet'),
        h('p', null, 'Write down setup steps, ideas and how-tos — or save a snippet: a command or config you copy often.'),
        h('div', { style: { display: 'flex', gap: '8px' } },
          h('button', { class: 'btn primary', onclick: () => createNote('note') }, ic('plus', 14), 'New note'),
          h('button', { class: 'btn', onclick: () => createNote('snippet') }, ic('code', 14), 'New snippet'))));
      return wrap;
    }
    const box = h('div', { class: 'notes' + (id ? ' has-note' : '') });
    noteListPane = h('div', { class: 'note-list' });
    notePane = h('article', { class: 'note' });
    fillNoteList(list, id);
    box.append(noteListPane, notePane);
    wrap.append(box);
    if (id) showNote(id);
    else notePane.append(h('div', { class: 'note-body' }, h('p', { class: 'hint' }, 'Pick a note.')));
    return wrap;
  }

  function fillNoteList(list, activeId) {
    if (!noteListPane) return;
    const kind = S.noteFilter === 'snippet' ? 'snippet' : 'note';
    noteListPane.replaceChildren(h('div', { class: 'note-list-head' },
      h('span', null, list.length + ' ' + (S.noteFilter === 'snippet' ? 'snippet' : S.noteFilter === 'note' ? 'note' : 'item') + (list.length === 1 ? '' : 's')),
      canEdit() ? h('button', { class: 'icon-btn', 'aria-label': 'New ' + kind, title: 'New ' + kind, onclick: () => createNote(kind) }, ic('plus', 15)) : null));
    if (!list.length) noteListPane.append(h('p', { class: 'tempty' }, { pinned: 'No pinned notes.', snippet: 'No snippets yet.', note: 'No notes yet.', shared: 'Nothing shared yet. Open a note and click Share to let everyone who signs in see it.' }[S.noteFilter] || 'Nothing here.'));
    list.forEach((n) => noteListPane.append(h('a', { class: 'note-row' + (n.id === activeId ? ' on' : ''), href: '#/notes/' + n.id, 'aria-current': n.id === activeId ? 'true' : null },
      h('span', { class: 'note-row-top' },
        n.type === 'snippet' ? h('span', { class: 'row-kind', title: 'Snippet' }, ic('code', 13, 'Snippet')) : null,
        h('span', { class: 'note-row-title' }, n.title),
        sharedMark(n),
        n.pinned ? h('span', { class: 'icon-btn on', style: { width: '16px', height: '16px' } }, ic('pin', 13, 'Pinned')) : null),
      n.snippet ? h('span', { class: 'note-row-snip' + (n.type === 'snippet' ? ' mono' : '') }, n.snippet) : null,
      h('span', { class: 'note-row-time' }, 'Edited ' + timeAgo(n.modified_at)))));
  }

  async function showNote(id) {
    let n = S.noteCache[id];
    if (!n) {
      notePane.replaceChildren(h('div', { class: 'note-body' }, h('p', { class: 'hint' }, 'Loading…')));
      try {
        n = await api('GET', '/api/notes/' + encodeURIComponent(id));
        S.noteCache[id] = n;
      } catch (e) {
        if (e.status === 404) { toast('That note no longer exists.', 'err'); go('#/notes', true); }
        return;
      }
    }
    if (S.route.section !== 'notes' || S.route.id !== id) return;
    if (S.noteEdit === id) renderNoteEditor(n);
    else renderNoteView(n);
  }

  function noteHead(n, titleNode, metaNode, actions, editing) {
    return h('div', { class: 'note-head' + (editing ? ' is-edit' : '') },
      h('a', { class: 'icon-btn note-back', href: '#/notes', 'aria-label': 'All notes' }, ic('back', 16)),
      h('div', { class: 'note-titles' }, titleNode, metaNode),
      h('div', { class: 'note-actions' }, actions));
  }

  function renderSnippetGrid(wrap, list) {
    if (!list.length) {
      wrap.append(h('div', { class: 'empty' }, ic('code', 22), h('h2', null, 'No snippets yet'),
        h('p', null, 'Keep commands, config blocks and other text you copy often, one click away.'),
        h('button', { class: 'btn primary', onclick: () => createNote('snippet') }, ic('plus', 14), 'New snippet')));
      return wrap;
    }
    const grid = h('div', { class: 'sgrid' });
    list.forEach((n) => {
      const pre = h('pre', { class: 'snip-body', title: 'Double-click to copy' }, h('code', null, n.content || ''));
      pre.addEventListener('dblclick', () => copyValue(n.content, n.title));
      grid.append(h('article', { class: 'snip', dataset: { item: n.id } },
        h('div', { class: 'snip-head' },
          h('a', { class: 'snip-title', href: '#/notes/' + n.id, title: 'Open' }, n.title),
          sharedMark(n),
          n.pinned ? h('span', { class: 'icon-btn on', style: { width: '16px', height: '16px' } }, ic('pin', 13, 'Pinned')) : null,
          copyButton(() => n.content, n.title)),
        pre));
    });
    wrap.append(grid);
    flashLater();
    return wrap;
  }

  function flashLater() {
    requestAnimationFrame(() => {
      if (!S.flashItem) return;
      const el = document.querySelector(`[data-item="${CSS.escape(S.flashItem)}"]`);
      if (el) { el.classList.add('flash'); el.scrollIntoView({ block: 'center', behavior: 'smooth' }); }
      S.flashItem = null;
    });
  }

  // A Copy button that confirms with "Copied" in place.
  function copyButton(getText, label) {
    const b = h('button', { class: 'btn copy-btn', type: 'button', 'aria-label': 'Copy ' + (label || '') }, ic('copy', 14), 'Copy');
    b.addEventListener('click', (e) => {
      e.stopPropagation();
      copyText(getText(), () => {
        b.replaceChildren(ic('check', 14), 'Copied');
        b.classList.add('ok');
        setTimeout(() => { b.replaceChildren(ic('copy', 14), 'Copy'); b.classList.remove('ok'); }, 1400);
      });
    });
    return b;
  }

  // Copy a value and say so ("Copied phone").
  function copyValue(text, what) {
    if (!text) return;
    copyText(text, () => toast(what ? `Copied ${what}` : 'Copied'));
    const sel = window.getSelection && window.getSelection();
    if (sel) sel.removeAllRanges();
  }

  // Add a small Copy button to every code block in rendered Markdown.
  function addCodeCopy(root) {
    root.querySelectorAll('pre').forEach((pre) => {
      const wrapEl = h('div', { class: 'code-wrap' });
      pre.replaceWith(wrapEl);
      const btn = h('button', { class: 'code-copy', type: 'button', 'aria-label': 'Copy code', title: 'Copy' }, ic('copy', 13));
      btn.addEventListener('click', () => copyText(pre.textContent.replace(/\n$/, ''), () => {
        btn.replaceChildren(ic('check', 13)); btn.classList.add('ok');
        setTimeout(() => { btn.replaceChildren(ic('copy', 13)); btn.classList.remove('ok'); }, 1400);
      }));
      wrapEl.append(pre, btn);
    });
  }

  function renderNoteView(n) {
    const isSnip = n.type === 'snippet';
    const actions = [];
    if (canEdit()) {
      if (n.can_edit) actions.push(h('button', { class: 'btn', onclick: () => { S.noteEdit = n.id; renderNoteEditor(n); } }, ic('pencil', 14), 'Edit'));
      if (n.can_manage && authOn()) actions.push(shareButton('note', n, () => showNote(n.id)));
      actions.push(h('button', { class: 'icon-btn' + (n.pinned ? ' on' : ''), 'aria-label': n.pinned ? 'Unpin note' : 'Pin note', title: n.pinned ? 'Unpin (just for you)' : 'Pin (just for you)', onclick: () => togglePin(n) }, ic('pin', 15)));
    }
    if (isSnip) actions.unshift(copyButton(() => n.content, n.title));
    else actions.push(h('button', { class: 'icon-btn', 'aria-label': 'Copy as Markdown', title: 'Copy as Markdown', onclick: () => copyText(n.content) }, ic('copy', 15)));
    if (n.can_manage) actions.push(h('button', { class: 'icon-btn', 'aria-label': 'Delete', title: 'Delete', onclick: () => deleteNote(n) }, ic('trash', 15)));
    let body;
    if (isSnip) {
      body = h('div', { class: 'note-body' }, h('pre', { class: 'snip-body snip-full', title: 'Double-click to copy' }, h('code', null, n.content || '')));
      body.addEventListener('dblclick', () => copyValue(n.content, 'snippet'));
    } else {
      body = h('div', { class: 'note-body md', html: markdown(n.content) || '<p class="hint">Empty note.</p>' });
      addCodeCopy(body);
      if (canEdit() && n.can_edit) body.addEventListener('dblclick', (e) => { if (!e.target.closest('pre,.code-copy')) { S.noteEdit = n.id; renderNoteEditor(n); } });
    }
    notePane.replaceChildren(
      noteHead(n, h('h1', { class: 'note-title' }, isSnip ? h('span', { class: 'row-kind big', title: 'Snippet' }, ic('code', 16, 'Snippet')) : null, n.title),
        h('span', { class: 'note-meta' }, sharedMark(n), `${!mine(n) ? 'Shared by ' + n.owner_name + ' · ' : isShared(n) && authOn() ? 'Shared · ' : ''}${isSnip ? 'Snippet · ' : ''}Edited ${timeAgo(n.modified_at)} · created ${fmtDate(n.created_at)}${n.can_edit ? '' : ' · read-only'}`), actions),
      body);
  }

  // autosave state for the note being edited
  const NS = { id: null, base: null, timer: null, saving: false, again: false, statusEl: null, title: null, text: null, conflict: null };

  function renderNoteEditor(n) {
    NS.id = n.id;
    NS.base = n.modified_at;
    NS.conflict = null;
    const isSnip = n.type === 'snippet';
    const title = h('input', { class: 'input note-title-input', type: 'text', value: n.title, 'aria-label': 'Title', maxlength: '200' });
    const text = h('textarea', { class: 'textarea' + (isSnip ? ' code' : ''), value: n.content, 'aria-label': isSnip ? 'Snippet text' : 'Note text (Markdown)', spellcheck: isSnip ? 'false' : 'true', placeholder: isSnip ? 'docker compose pull && docker compose up -d' : '' });
    const typeSeg = seg([['note', 'Note'], ['snippet', 'Snippet']], isSnip ? 'snippet' : 'note', async (t) => {
      if (t === n.type) return;
      await flushNote();
      try {
        const u = await api('PUT', '/api/notes/' + encodeURIComponent(n.id), { type: t });
        S.noteCache[n.id] = u;
        const meta = S.notes.find((m) => m.id === n.id);
        if (meta) { meta.type = u.type; meta.modified_at = u.modified_at; meta.content = u.type === 'snippet' ? u.content : undefined; }
        renderSub();
        renderNoteEditor(u);
      } catch (e) { if (e.status !== 401) toast(e.message, 'err'); }
    }, 'Type');
    typeSeg.classList.add('type-seg');
    const status = h('span', { class: 'note-meta' }, ic('check', 12), 'Saved');
    NS.title = title; NS.text = text; NS.statusEl = status;
    const onEdit = () => { setNoteStatus('dirty'); scheduleNoteSave(); };
    title.addEventListener('input', onEdit);
    text.addEventListener('input', () => { autoGrow(text); onEdit(); });
    text.addEventListener('keydown', (e) => {
      if (e.key === 'Tab' && !e.shiftKey) { e.preventDefault(); text.setRangeText('  ', text.selectionStart, text.selectionEnd, 'end'); onEdit(); }
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 's') { e.preventDefault(); flushNote(); }
    });
    const done = h('button', { class: 'btn primary', onclick: async () => { await flushNote(); S.noteEdit = null; showNote(n.id); } }, 'Done');
    notePane.replaceChildren(
      noteHead(n, title, status, [typeSeg, done], true),
      h('div', { class: 'note-body' }, text, h('p', { class: 'hint', style: { marginTop: '8px' } },
        isSnip ? 'Plain text, shown exactly as written and copied with one click. Saves automatically.'
          : 'Markdown: # heading, - list, - [ ] task, **bold**, `code`, ``` code block ```, [text](https://…). Saves automatically.')));
    requestAnimationFrame(() => { autoGrow(text); text.focus(); });
  }

  function autoGrow(t) {
    t.style.height = 'auto';
    t.style.height = Math.max(320, t.scrollHeight + 4) + 'px';
  }

  function setNoteStatus(kind, msg) {
    const el = NS.statusEl;
    if (!el) return;
    el.replaceChildren();
    if (kind === 'saved') el.append(ic('check', 12), 'Saved');
    else if (kind === 'saving') el.append('Saving…');
    else if (kind === 'dirty') el.append('Unsaved changes');
    else el.append(h('span', { class: 'warn' }, msg || 'Not saved'));
  }

  function scheduleNoteSave() {
    clearTimeout(NS.timer);
    NS.timer = setTimeout(() => saveNote(), 1200);
  }

  async function saveNote(force, keepalive) {
    clearTimeout(NS.timer);
    if (!NS.id || !NS.title) return;
    if (NS.saving) { NS.again = true; return; }
    const cached = S.noteCache[NS.id];
    const title = NS.title.value;
    const content = NS.text.value;
    if (cached && cached.title === title && cached.content === content && !force) { setNoteStatus('saved'); return; }
    NS.saving = true;
    setNoteStatus('saving');
    const id = NS.id;
    try {
      const n = await api('PUT', '/api/notes/' + encodeURIComponent(id), { title, content, base_modified: force ? '' : NS.base }, { keepalive });
      S.noteCache[id] = n;
      if (NS.id === id) NS.base = n.modified_at;
      const meta = S.notes.find((m) => m.id === id);
      if (meta) {
        meta.title = n.title; meta.modified_at = n.modified_at;
        meta.snippet = n.type === 'snippet' ? (n.content.split('\n').find((l) => l.trim()) || '').trim().slice(0, 140) : snippet(n.content);
        if (n.type === 'snippet') meta.content = n.content;
      }
      if (NS.id === id) setNoteStatus('saved');
      if (S.route.section === 'notes') fillNoteList(sortedNotes(), S.route.id);
    } catch (e) {
      if (e.status === 409) showConflict(id);
      else setNoteStatus('error', e.status === 401 ? 'Sign in to save' : 'Not saved — ' + e.message);
    } finally {
      NS.saving = false;
      if (NS.again) { NS.again = false; saveNote(); }
    }
  }

  function flushNote(keepalive) {
    if (!NS.id || !NS.title) return Promise.resolve();
    return saveNote(false, keepalive);
  }

  function snippet(content) {
    const out = [];
    let fence = false;
    for (const line of String(content).split('\n')) {
      const t = line.trim();
      if (t.startsWith('```')) { fence = !fence; continue; }
      if (!t || fence) continue;
      out.push(t.replace(/^[#>\-*+\s]+/, '').replace(/\*\*|__|`|\[ \]|\[x\]/g, ''));
      if (out.join(' ').length > 140) break;
    }
    return out.join(' ').slice(0, 140);
  }

  function showConflict(id) {
    setNoteStatus('error', 'Changed somewhere else');
    if (!notePane || notePane.querySelector('.conflict')) return;
    const bar = h('div', { class: 'conflict', role: 'alert' },
      h('span', null, 'This note was changed in another tab or device.'),
      h('button', { class: 'btn', onclick: async () => {
        delete S.noteCache[id];
        S.noteEdit = null;
        const n = await api('GET', '/api/notes/' + encodeURIComponent(id));
        S.noteCache[id] = n;
        bar.remove();
        showNote(id);
      } }, 'Load theirs'),
      h('button', { class: 'btn primary', onclick: async () => { bar.remove(); await saveNote(true); } }, 'Keep mine'));
    notePane.insertBefore(bar, notePane.children[1] || null);
  }

  async function createNote(type) {
    type = type === 'snippet' ? 'snippet' : 'note';
    if (!canEdit()) { openLogin(() => createNote(type)); return; }
    try {
      const n = await api('POST', '/api/notes', { type, title: 'Untitled', content: '', visibility: S.noteFilter === 'shared' && authOn() ? 'shared' : 'private' });
      S.noteCache[n.id] = n;
      S.notes.unshift({ ...n, content: type === 'snippet' ? '' : undefined, snippet: '' });
      if (S.noteFilter === 'pinned' || (S.noteFilter === 'note' && type === 'snippet') || (S.noteFilter === 'snippet' && type === 'note')) S.noteFilter = 'all';
      S.noteEdit = n.id;
      go('#/notes/' + n.id);
      requestAnimationFrame(() => { if (NS.title) { NS.title.focus(); NS.title.select(); } });
    } catch (e) { if (e.status !== 401) toast(e.message, 'err'); }
  }

  async function togglePin(n) {
    try {
      const u = await api('PUT', '/api/notes/' + encodeURIComponent(n.id), { pinned: !n.pinned });
      S.noteCache[n.id] = u;
      const meta = S.notes.find((m) => m.id === n.id);
      if (meta) { meta.pinned = u.pinned; meta.modified_at = u.modified_at; }
      renderSub();
      renderMain();
    } catch (e) { if (e.status !== 401) toast(e.message, 'err'); }
  }

  async function deleteNote(n) {
    const ok = await confirmBox(`Delete “${n.title}”?`, 'This cannot be undone (except by restoring a backup).', 'Delete', true);
    if (!ok) return;
    try {
      await api('DELETE', '/api/notes/' + encodeURIComponent(n.id));
      S.notes = S.notes.filter((m) => m.id !== n.id);
      delete S.noteCache[n.id];
      go('#/notes', true);
      renderSub();
    } catch (e) { if (e.status !== 401) toast(e.message, 'err'); }
  }

  // Share toggle for notes, task lists and directory entries (pages use the page menu).
  function shareButton(kind, item, after) {
    const on = isShared(item);
    return h('button', { class: 'btn share-btn' + (on ? ' on' : ''), 'aria-pressed': String(on), title: on ? 'Shared with everyone signed in — click to make it private' : 'Private — click to share with everyone signed in',
      onclick: () => setVisibility(kind, item, on ? 'private' : 'shared', after) }, ic(on ? 'users' : 'lock', 14), on ? 'Shared' : 'Share');
  }

  async function setVisibility(kind, item, visibility, after) {
    try {
      const acc = await api('PUT', '/api/visibility', { kind, id: item.id, visibility });
      Object.assign(item, acc);
      if (kind === 'note') {
        const meta = S.notes.find((m) => m.id === item.id);
        if (meta) Object.assign(meta, acc);
        if (S.noteCache[item.id]) Object.assign(S.noteCache[item.id], acc);
      }
      toast(visibility === 'private' ? 'Private — only you can see it' : visibility === 'public' ? 'Public — anyone can see it' : 'Shared with everyone signed in');
      renderSub();
      if (after) after(); else renderMain();
    } catch (e) { if (e.status !== 401) toast(e.message, 'err'); }
  }

  function copyText(t, cb) {
    const done = cb || (() => toast('Copied'));
    if (navigator.clipboard && window.isSecureContext) navigator.clipboard.writeText(t).then(done, () => fallbackCopy(t, done));
    else fallbackCopy(t, done);
  }

  function fallbackCopy(t, done) {
    const ta = h('textarea', { value: t, style: { position: 'fixed', opacity: '0' } });
    document.body.append(ta);
    ta.select();
    try { document.execCommand('copy'); done(); } catch (e) { toast('Could not copy', 'err'); }
    ta.remove();
  }

  // ------------------------------------------------------------------
  // Tasks
  // ------------------------------------------------------------------

  function renderTasks() {
    if (!signedIn()) return lockedView('Tasks are private', 'Sign in to see them.');
    const wrap = h('div', { class: 'wrap' + (S.editing ? ' editing' : '') });
    if (!S.tasks.length) {
      wrap.append(h('div', { class: 'empty' }, ic('task', 22), h('h2', null, 'No task lists yet'),
        h('p', null, canEdit() ? 'Create a list for chores, upgrades or ideas.' : 'Sign in to add tasks.'),
        canEdit() ? h('button', { class: 'btn primary', onclick: createList }, ic('plus', 14), 'New list') : null));
      return wrap;
    }
    const grid = h('div', { class: 'grid tgrid' });
    const lists = S.taskFilter === 'shared' ? S.tasks.filter(isShared) : S.tasks;
    if (!lists.length) grid.append(h('p', { class: 'hint' }, 'No shared lists yet. Click Edit, then Share on a list, to let everyone who signs in see it and check things off.'));
    lists.forEach((l) => grid.append(taskListEl(l, S.tasks.indexOf(l))));
    wrap.append(grid);
    requestAnimationFrame(() => {
      if (S.focusAdd) { const inp = document.querySelector(`[data-add="${CSS.escape(S.focusAdd)}"]`); if (inp) inp.focus(); S.focusAdd = null; }
      if (S.flashItem) {
        const el = document.querySelector(`[data-item="${CSS.escape(S.flashItem)}"]`);
        if (el) { el.classList.add('flash'); el.scrollIntoView({ block: 'center', behavior: 'smooth' }); }
        S.flashItem = null;
      }
    });
    return wrap;
  }

  function taskListEl(l, index) {
    const editing = S.editing && canEdit() && l.can_edit;
    const total = l.items.length;
    const done = l.items.filter((x) => x.done).length;
    const visible = l.items.filter((x) => (S.taskFilter === 'all' || S.taskFilter === 'shared' ? true : S.taskFilter === 'done' ? x.done : !x.done));
    const card = h('section', { class: 'tlist', 'aria-label': l.title });
    let titleNode;
    if (editing) {
      titleNode = h('input', { class: 'input tlist-title-input', type: 'text', value: l.title, 'aria-label': 'List name' });
      const commit = async () => {
        const v = titleNode.value.trim();
        if (!v || v === l.title) { titleNode.value = l.title; return; }
        try { const u = await api('PUT', '/api/tasks/' + l.id, { title: v }); l.title = u.title; } catch (e) { titleNode.value = l.title; if (e.status !== 401) toast(e.message, 'err'); }
      };
      titleNode.addEventListener('blur', commit);
      titleNode.addEventListener('keydown', (e) => { if (e.key === 'Enter') titleNode.blur(); if (e.key === 'Escape') { titleNode.value = l.title; titleNode.blur(); } });
    } else titleNode = h('h2', { class: 'tlist-title' }, l.title);
    card.append(h('div', { class: 'tlist-head' },
      h('div', { class: 'tlist-top' },
        editing ? h('span', { class: 'grip', title: 'Drag to reorder' }, ic('grip', 12)) : null,
        titleNode,
        sharedMark(l),
        h('span', { class: 'tlist-prog' }, `${done} of ${total}`),
        editing && l.can_manage && authOn() ? shareButton('list', l) : null,
        editing && l.can_manage ? h('button', { class: 'icon-btn', 'aria-label': 'Delete list ' + l.title, onclick: () => deleteList(l) }, ic('trash', 14)) : null),
      h('div', { class: 'bar' }, h('div', { style: { width: (total ? Math.round((100 * done) / total) : 0) + '%' } }))));
    const items = h('div', { class: 'titems' });
    visible.forEach((it) => items.append(taskItemEl(l, it)));
    if (!visible.length) items.append(h('p', { class: 'tempty' }, total ? (S.taskFilter === 'done' ? 'Nothing finished yet.' : 'All done.') : 'No tasks yet.'));
    card.append(items);
    if (canEdit() && l.can_edit) {
      const text = h('input', { type: 'text', placeholder: 'Add a task', 'aria-label': 'Add a task to ' + l.title, dataset: { add: l.id }, maxlength: '500' });
      const due = h('input', { type: 'date', class: 'sr-only', tabindex: '-1', 'aria-hidden': 'true' });
      const dueBtn = h('button', { type: 'button', class: 'icon-btn', 'aria-label': 'Due date (optional)', title: 'Due date (optional)' }, ic('calendar', 14));
      const showDue = () => {
        dueBtn.replaceChildren(due.value ? h('span', { class: 'due ' + dueClass(due.value) }, dueLabel(due.value)) : ic('calendar', 14));
        dueBtn.style.width = due.value ? 'auto' : '';
      };
      due.addEventListener('change', showDue);
      dueBtn.addEventListener('click', (e) => {
        e.preventDefault();
        if (due.value) { due.value = ''; showDue(); return; } // click again to clear
        if (due.showPicker) { try { due.showPicker(); return; } catch (err) { /* fall through */ } }
        due.classList.remove('sr-only'); due.classList.add('date-input'); due.focus();
      });
      text.addEventListener('keydown', async (e) => {
        if (e.key !== 'Enter' || !text.value.trim()) return;
        const v = text.value.trim();
        text.value = '';
        try {
          const it = await api('POST', `/api/tasks/${l.id}/items`, { text: v, due: due.value || '' });
          l.items.push(it);
          if (S.taskFilter === 'done') S.taskFilter = 'open';
          S.focusAdd = l.id;
          renderSub(); renderMain();
        } catch (err) { text.value = v; if (err.status !== 401) toast(err.message, 'err'); }
      });
      card.append(h('div', { class: 'tadd' }, ic('plus', 14), text, due, dueBtn));
    }
    if (S.editing && canEdit()) {
      card.draggable = true;
      card.addEventListener('dragstart', (e) => {
        if (e.target.closest('input')) { e.preventDefault(); return; }
        drag = { type: 'list', from: index }; e.dataTransfer.effectAllowed = 'move'; e.dataTransfer.setData('text/plain', 'list');
      });
      card.addEventListener('dragend', () => { drag = null; clearDropMarks(); });
      card.addEventListener('dragover', (e) => { if (drag && drag.type === 'list') { e.preventDefault(); clearDropMarks(); card.classList.add('drop-target'); } });
      card.addEventListener('drop', async (e) => {
        if (!drag || drag.type !== 'list') return;
        e.preventDefault();
        moveIn(S.tasks, drag.from, index);
        drag = null;
        renderMain();
        try { S.tasks = await api('PUT', '/api/tasks/order', { ids: S.tasks.map((x) => x.id) }); } catch (err) { if (err.status !== 401) toast(err.message, 'err'); }
      });
    }
    return card;
  }

  function taskItemEl(l, it) {
    const can = canEdit() && l.can_edit;
    const row = h('div', { class: 'titem' + (it.done ? ' done' : ''), dataset: { item: it.id } });
    const box = h('input', { type: 'checkbox', checked: it.done, disabled: !can, 'aria-label': (it.done ? 'Mark not done: ' : 'Mark done: ') + it.text });
    box.addEventListener('change', async () => {
      const prev = it.done;
      it.done = box.checked;
      row.classList.toggle('done', it.done);
      try {
        const u = await api('PUT', '/api/items/' + it.id, { done: it.done });
        Object.assign(it, u);
        setTimeout(() => { renderSub(); if (S.taskFilter !== 'all') renderMain(); else renderMain(); }, S.taskFilter === 'all' ? 0 : 350);
      } catch (e) { it.done = prev; box.checked = prev; row.classList.toggle('done', prev); if (e.status !== 401) toast(e.message, 'err'); }
    });
    const text = h('span', { class: 'titem-text', title: it.done && it.done_by_name && isShared(l) ? `Done by ${it.done_by_name}` : null }, it.text);
    if (can) {
      text.title = 'Double-click to edit';
      text.addEventListener('dblclick', () => {
        const inp = h('input', { class: 'input', type: 'text', value: it.text, maxlength: '500', 'aria-label': 'Task text' });
        let doneEditing = false;
        const finish = async (save) => {
          if (doneEditing) return;
          doneEditing = true;
          const v = inp.value.trim();
          if (save && v && v !== it.text) {
            try { const u = await api('PUT', '/api/items/' + it.id, { text: v }); Object.assign(it, u); } catch (e) { if (e.status !== 401) toast(e.message, 'err'); }
          }
          renderMain();
        };
        inp.addEventListener('keydown', (e) => { if (e.key === 'Enter') finish(true); if (e.key === 'Escape') finish(false); });
        inp.addEventListener('blur', () => finish(true));
        text.replaceWith(inp);
        inp.focus();
        inp.select();
      });
    }
    row.append(box, text);
    if (it.done && it.done_by_name && isShared(l) && authOn()) row.append(h('span', { class: 'done-by' }, it.done_by_name));
    // due date
    const setDue = () => {
      const di = h('input', { type: 'date', class: 'date-input', value: it.due || '', 'aria-label': 'Due date' });
      let fired = false;
      const commit = async () => {
        if (fired) return;
        fired = true;
        if ((di.value || '') !== (it.due || '')) {
          try { const u = await api('PUT', '/api/items/' + it.id, { due: di.value || '' }); Object.assign(it, u); } catch (e) { if (e.status !== 401) toast(e.message, 'err'); }
        }
        renderMain();
      };
      di.addEventListener('change', commit);
      di.addEventListener('blur', commit);
      di.addEventListener('keydown', (e) => { if (e.key === 'Escape') { fired = true; renderMain(); } });
      dueSlot.replaceChildren(di);
      di.focus();
      if (di.showPicker) { try { di.showPicker(); } catch (e) { /* not allowed */ } }
    };
    const dueSlot = h('span');
    if (it.due && !it.done) {
      dueSlot.append(h('button', { class: 'due ' + dueClass(it.due), title: can ? 'Change due date' : it.due, disabled: !can, onclick: setDue }, dueLabel(it.due)));
    } else if (can && !it.done) {
      dueSlot.append(h('button', { class: 'icon-btn hover-only', 'aria-label': 'Set due date', title: 'Set due date', onclick: setDue }, ic('calendar', 14)));
    }
    row.append(dueSlot);
    if (can) {
      row.append(h('button', { class: 'icon-btn hover-only', 'aria-label': 'Delete task', title: 'Delete', onclick: async () => {
        try {
          await api('DELETE', '/api/items/' + it.id);
          l.items = l.items.filter((x) => x.id !== it.id);
          renderSub(); renderMain();
        } catch (e) { if (e.status !== 401) toast(e.message, 'err'); }
      } }, ic('x', 14)));
    }
    return row;
  }

  async function createList() {
    if (!canEdit()) { openLogin(createList); return; }
    const title = await promptBox('New task list', 'List name', '', 'Create');
    if (!title) return;
    try {
      const l = await api('POST', '/api/tasks', { title });
      S.tasks.push(l);
      S.focusAdd = l.id;
      if (S.route.section !== 'tasks') go('#/tasks'); else { renderSub(); renderMain(); }
    } catch (e) { if (e.status !== 401) toast(e.message, 'err'); }
  }

  async function deleteList(l) {
    const ok = await confirmBox(`Delete “${l.title}”?`, l.items.length ? `Its ${l.items.length} task${l.items.length === 1 ? '' : 's'} will be deleted too.` : 'The list is empty.', 'Delete', true);
    if (!ok) return;
    try {
      await api('DELETE', '/api/tasks/' + l.id);
      S.tasks = S.tasks.filter((x) => x.id !== l.id);
      renderSub(); renderMain();
    } catch (e) { if (e.status !== 401) toast(e.message, 'err'); }
  }

  // ------------------------------------------------------------------
  // Settings
  // ------------------------------------------------------------------

  const ACCENTS = [['#7AA2F7', 'Blue'], ['#5ED3BE', 'Teal'], ['#B08CF7', 'Violet'], ['#F0B45A', 'Amber'], ['#F0788C', 'Rose'], ['#A0AAB9', 'Gray']];
  const HEADERS = [['#1E2A3B', 'Navy'], ['#0F3D3E', 'Deep teal'], ['#2E3A2F', 'Moss'], ['#33293D', 'Plum'], ['#3A2E26', 'Walnut']];
  const LETTER_COLORS = [['', 'Accent'], ['#F0B45A', 'Amber'], ['#1E2A3B', 'Navy'], ['#0F3D3E', 'Deep teal'], ['#33293D', 'Plum'], ['#E6E8EB', 'Light'], ['#16191F', 'Black']];
  const LAYOUTS = [['cards', 'Cards'], ['tiles', 'Tiles'], ['compact', 'Compact'], ['rows', 'Rows']];

  // The settings object mixes the dashboard's (admins) and the person's own;
  // accent and layout fall back from the person's choice to the dashboard default.
  function resolveSettings() {
    const st = S.settings;
    st.accent = st.pref_accent || st.default_accent || '#F0B45A';
    st.layout = st.pref_layout || st.default_layout || 'cards';
  }

  let faviconStale = false;
  const pushSettings = debounce(async () => {
    try {
      const out = await api('PUT', '/api/settings', S.settings);
      Object.assign(S.settings, out); // keep object identity for open settings controls
      applyAppearance();
      if (faviconStale) { faviconStale = false; refreshFavicon(); }
    } catch (e) { if (e.status !== 401) toast('Settings not saved: ' + e.message, 'err'); }
  }, 450);

  function setSetting(patch, opts = {}) {
    Object.assign(S.settings, patch);
    resolveSettings();
    applyAppearance();
    renderHeader();
    if (patch.icon || 'default_accent' in patch) faviconStale = true;
    if (opts.rerender) renderMain();
    pushSettings();
    armIdle();
  }

  async function loadSettingsExtras() {
    try {
      const jobs = [api('GET', '/api/about', undefined, { quiet401: true })];
      if (isAdmin()) jobs.push(api('GET', '/api/backups', undefined, { quiet401: true }), authOn() ? api('GET', '/api/users', undefined, { quiet401: true }) : Promise.resolve(null));
      const [about, b, users] = await Promise.all(jobs);
      S.about = about;
      S.backups = b || null;
      S.users = users || null;
      if (S.route.section === 'settings') renderMain();
    } catch (e) { /* not signed in */ }
  }

  function srow(label, help, ctl, inline) {
    return h('div', { class: 'srow' },
      h('div', { class: 'srow-label' }, h('b', null, label), help ? h('span', null, help) : null),
      h('div', { class: 'srow-ctl' + (inline ? ' inline' : '') }, ctl));
  }

  function seg(options, value, onPick, label) {
    return h('div', { class: 'seg', role: 'group', 'aria-label': label },
      options.map(([v, l]) => h('button', { type: 'button', 'aria-pressed': String(v === value), onclick: () => onPick(v) }, l)));
  }

  function settingsSections() {
    return [['profile', 'Profile'], ['general', 'Preferences'], ['appearance', 'Appearance'],
      isAdmin() ? ['dashboard', 'Dashboard'] : null, isAdmin() && authOn() ? ['users', 'Users'] : null,
      ['data', 'Data & backups'], ['security', 'Security'], ['about', 'About']].filter(Boolean);
  }

  function renderSettings() {
    if (!canEdit()) {
      return h('div', { class: 'wrap' }, h('div', { class: 'empty' }, ic('lock', 22), h('h2', null, 'Sign in to change settings'),
        h('button', { class: 'btn primary', onclick: () => openLogin(() => renderAll()) }, 'Sign in')));
    }
    const st = S.settings;
    const sections = settingsSections();
    const section = sections.some(([k]) => k === S.route.id) ? S.route.id : sections[0][0];
    const navItem = (id, label) => h('a', { class: section === id ? 'on' : '', href: '#/settings/' + id, onclick: (e) => {
      e.preventDefault();
      history.replaceState(null, '', '#/settings/' + id);
      S.route.id = id;
      document.querySelectorAll('.snav a').forEach((a) => a.classList.toggle('on', a.getAttribute('href') === '#/settings/' + id));
      const el = document.getElementById('s-' + id);
      if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' });
    } }, label);
    const cards = { profile: profileCard, general: () => generalCard(st), appearance: () => appearanceCard(st), dashboard: () => dashboardCard(st), users: usersCard, data: dataCard, security: securityCard, about: aboutCard };
    const wrap = h('div', { class: 'wrap' },
      h('div', { class: 'settings-head' }, h('h1', null, 'Settings'), h('p', null, 'Changes save automatically.')),
      h('div', { class: 'settings' },
        h('nav', { class: 'snav', 'aria-label': 'Settings sections' }, sections.map(([k, l]) => navItem(k, l))),
        h('div', { class: 'spanel' }, sections.map(([k]) => cards[k]()))));
    if (S.scrollSettings && section !== sections[0][0]) requestAnimationFrame(() => { const el = document.getElementById('s-' + section); if (el) el.scrollIntoView({ block: 'start' }); });
    S.scrollSettings = false;
    return wrap;
  }

  function card(id, title, extra, ...rows) {
    return h('section', { class: 'scard', id: 's-' + id }, h('div', { class: 'scard-head' }, h('h2', null, title), extra ? h('span', null, extra) : null), rows);
  }

  // ---------- profile ----------

  function profileCard() {
    const m = me() || {};
    const name = h('input', { class: 'input', type: 'text', value: m.name || '', placeholder: m.username, maxlength: '60', 'aria-label': 'Your name', autocomplete: 'name' });
    const saveName = debounce(async () => {
      try { S.session.me = await api('PUT', '/api/me', { name: name.value }); renderHeader(); } catch (e) { if (e.status !== 401) toast(e.message, 'err'); }
    }, 500);
    name.addEventListener('input', saveName);
    const rows = [
      srow('Your name', 'Shown in the header and on everything you share.', name),
      srow('Username', 'What you sign in with.', h('div', { class: 'who-line' }, h('code', null, m.username || ''), h('span', { class: 'chip' }, m.role === 'admin' ? 'Admin' : 'Member'))),
    ];
    if (authOn()) {
      const cur = h('input', { class: 'input', type: 'password', autocomplete: 'current-password', placeholder: 'Current password', 'aria-label': 'Current password' });
      const next = h('input', { class: 'input', type: 'password', autocomplete: 'new-password', placeholder: 'New password (8+ characters)', 'aria-label': 'New password', minlength: '8' });
      const again = h('input', { class: 'input', type: 'password', autocomplete: 'new-password', placeholder: 'New password again', 'aria-label': 'New password again' });
      const err = h('p', { class: 'form-error', role: 'alert' });
      const btn = h('button', { class: 'btn', type: 'button', onclick: async () => {
        err.textContent = '';
        if (next.value.length < 8) { err.textContent = 'Use at least 8 characters.'; next.focus(); return; }
        if (next.value !== again.value) { err.textContent = 'The new passwords don’t match.'; again.focus(); return; }
        try {
          await api('PUT', '/api/me/password', { current: cur.value, password: next.value });
          cur.value = next.value = again.value = '';
          toast('Password changed. Other devices are signed out.');
        } catch (e) { if (e.status !== 401) err.textContent = e.message; }
      } }, ic('key', 14), 'Change password');
      rows.push(srow('Password', 'Changing it signs you out on every other device.', h('div', { class: 'stack' }, cur, next, again, err, h('div', null, btn))));
      rows.push(srow('Sessions', S.session.expires ? `This browser stays signed in until ${fmtDate(S.session.expires)}.` : null,
        h('div', { class: 'btn-row' },
          h('button', { class: 'btn', onclick: async () => {
            try { await api('POST', '/api/me/signout-all'); toast('Signed out everywhere else'); } catch (e) { if (e.status !== 401) toast(e.message, 'err'); }
          } }, 'Sign out everywhere else'),
          h('button', { class: 'btn', onclick: logout }, ic('logout', 14), 'Sign out'))));
    }
    return card('profile', 'Profile', null, rows);
  }

  // ---------- preferences (each person's own) ----------

  function generalCard(st) {
    const startSel = h('select', { class: 'select', 'aria-label': 'Start on' },
      S.pages.map((p, i) => h('option', { value: i === 0 ? 'links' : 'links:' + p.id, selected: (i === 0 && st.start_on === 'links') || st.start_on === 'links:' + p.id }, 'Links · ' + p.name + (i === 0 ? ' (first page)' : ''))),
      SECTIONS.slice(1).filter(([k]) => sectionOn(k)).map(([k, l]) => h('option', { value: k, selected: st.start_on === k }, l)),
      h('option', { value: 'blank', selected: st.start_on === 'blank' }, 'Blank page'),
      h('option', { value: 'last', selected: st.start_on === 'last' }, 'Last visited'));
    const idleSel = h('select', { class: 'select', 'aria-label': 'Return to start page when idle', disabled: st.start_on === 'last' },
      [[0, 'Never'], [1, 'After 1 minute'], [2, 'After 2 minutes'], [5, 'After 5 minutes'], [10, 'After 10 minutes'], [15, 'After 15 minutes'], [30, 'After 30 minutes'], [60, 'After 1 hour']]
        .map(([v, l]) => h('option', { value: v, selected: Number(st.idle_minutes) === v }, l)));
    startSel.addEventListener('change', () => { setSetting({ start_on: startSel.value }); idleSel.disabled = startSel.value === 'last'; });
    idleSel.addEventListener('change', () => setSetting({ idle_minutes: Number(idleSel.value) }));

    const hidden = hiddenSet();
    const sectionBoxes = h('div', { class: 'checks' },
      h('label', { class: 'check', title: 'Links are always on' }, h('input', { type: 'checkbox', checked: true, disabled: true }), 'Links'),
      [['notes', 'Notes'], ['tasks', 'Tasks'], ['directory', 'Directory'], ['scratchpad', 'Scratchpad']].map(([k, l]) => {
        const cb = h('input', { type: 'checkbox', checked: !hidden.has(k) });
        cb.addEventListener('change', () => {
          const next = new Set(hiddenSet());
          if (cb.checked) next.delete(k); else next.add(k);
          setSetting({ hidden_sections: [...next] });
          renderDock(); renderPad();
        });
        return h('label', { class: 'check' }, cb, l);
      }));
    const newTab = h('input', { type: 'checkbox', role: 'switch', checked: st.new_tab, 'aria-label': 'Open links in a new tab' });
    newTab.addEventListener('change', () => setSetting({ new_tab: newTab.checked }));

    return card('general', 'Preferences', authOn() ? 'Just for you' : null,
      srow('Sections', 'Choose what appears in your header. Turning a section off hides it; nothing is deleted.', sectionBoxes),
      srow('Start on', 'What opens when you visit the dashboard.', startSel),
      srow('Return to start page', 'After this long without activity, go back to the start page. Handy for wall displays.', [idleSel, st.start_on === 'last' ? h('span', { class: 'hint' }, 'Not available when starting on the last visited page.') : null]),
      srow('Open links in a new tab', 'Default for new links; each link can override it.', h('label', { class: 'check' }, newTab, 'New tab'), true));
  }

  function accentPicker(value, onPick, withDefault, defaultColor) {
    const hex = h('input', { class: 'input mono', type: 'text', value: value || '', placeholder: withDefault ? 'Default' : '', maxlength: '7', style: { width: '110px' }, 'aria-label': 'Custom accent color' });
    hex.addEventListener('change', () => { const v = hex.value.trim(); if ((withDefault && !v) || /^#[0-9a-f]{6}$/i.test(v)) onPick(v.toUpperCase()); else hex.value = value || ''; });
    return h('div', { class: 'swatches' },
      withDefault ? h('button', { type: 'button', class: 'swatch default-accent', 'aria-label': 'Dashboard default accent', title: 'Dashboard default', 'aria-pressed': String(!value), style: { '--sw': defaultColor }, onclick: () => onPick('') }) : null,
      ACCENTS.map(([c, l]) => h('button', { type: 'button', class: 'swatch', 'aria-label': l + ' accent', 'aria-pressed': String((value || '').toUpperCase() === c), style: { background: c }, onclick: () => onPick(c) })),
      hex);
  }

  function appearanceCard(st) {
    const localOverride = localGet('ds-theme');
    const themeSeg = seg([['dark', 'Dark'], ['light', 'Light'], ['system', 'System']], st.theme, (t) => { localSet('ds-theme', null); setSetting({ theme: t }, { rerender: true }); renderDock(); }, 'Theme');
    const multi = authOn() && isAdmin();
    const layoutSeg = seg([['', 'Default'], ...LAYOUTS], st.pref_layout || '', (v) => setSetting({ pref_layout: v }, { rerender: true }), 'Link layout');
    return card('appearance', 'Appearance', authOn() ? 'Just for you' : null,
      srow('Theme', localOverride ? `System follows your device. This browser is set to ${localOverride} using the toggle in the corner; picking here resets that.` : 'The toggle in the corner switches it just for this browser.', themeSeg),
      srow('Accent color', 'Active tabs, buttons and highlights. “Default” follows the dashboard' + (multi ? ' (set under Dashboard).' : '.'), accentPicker(st.pref_accent, (v) => setSetting({ pref_accent: v }, { rerender: true }), true, st.default_accent)),
      srow('Link layout', `How links are shown. “Default” is ${(LAYOUTS.find(([k]) => k === st.default_layout) || LAYOUTS[0])[1].toLowerCase()}.`, layoutSeg));
  }

  // ---------- dashboard (admins) ----------

  function dashboardCard(st) {
    const name = h('input', { class: 'input', type: 'text', value: st.title, maxlength: '60', 'aria-label': 'Dashboard name' });
    name.addEventListener('input', () => {
      const patch = { title: name.value };
      if (st.icon.type === 'letter') patch.icon = { ...st.icon, letter: (name.value.trim()[0] || 'D').toUpperCase() };
      setSetting(patch);
    });

    const iconCtl = h('div', { class: 'srow-ctl' });
    const drawIcon = () => {
      const icon = st.icon;
      const typeSeg = seg([['logo', 'Logo'], ['letter', 'Letter'], ['library', 'Library'], ['image', 'Upload']], icon.type, (t) => {
        if (t === 'image' && !st.icon.image) { pickImage(); return; }
        setSetting({ icon: { ...st.icon, type: t } }); drawIcon();
      }, 'Icon type');
      const top = h('div', { style: { display: 'flex', gap: '16px', alignItems: 'center' } }, appIcon('app-icon icon-preview'), h('div', { style: { flex: '1' } }, typeSeg));
      const rows = [top];
      const colorSwatches = () => LETTER_COLORS.map(([c, l]) => h('button', { type: 'button', class: 'swatch', 'aria-label': l + ' background', 'aria-pressed': String((icon.color || '') === c), style: { background: c || st.accent }, onclick: () => { setSetting({ icon: { ...st.icon, color: c } }); drawIcon(); } }));
      if (icon.type === 'logo') {
        rows.push(h('div', { class: 'swatches' }, colorSwatches()));
      } else if (icon.type === 'letter') {
        const letter = h('input', { class: 'input', type: 'text', value: icon.letter, maxlength: '2', style: { width: '64px', textAlign: 'center' }, 'aria-label': 'Letter' });
        letter.addEventListener('input', () => { setSetting({ icon: { ...st.icon, letter: letter.value } }); top.firstChild.replaceWith(appIcon('app-icon icon-preview')); });
        rows.push(h('div', { class: 'swatches' }, letter, colorSwatches()));
      } else if (icon.type === 'library') {
        const apply = (v) => { setSetting({ icon: { ...st.icon, name: v } }); top.firstChild.replaceWith(appIcon('app-icon icon-preview')); };
        const applySoon = debounce(apply, 400);
        const pk = iconPicker(icon.name, (v, typing) => (typing ? applySoon(v) : apply(v)), { label: 'Icon name', placeholder: 'Search icons, e.g. homarr — or paste an image URL' });
        rows.push(pk.el, h('span', { class: 'hint' }, 'From ', h('a', { href: 'https://dashboardicons.com', target: '_blank', rel: 'noopener noreferrer' }, 'Dashboard Icons'), ', or any image URL.'));
      } else {
        rows.push(h('div', null, h('button', { class: 'btn', type: 'button', onclick: pickImage }, ic('upload', 14), 'Choose image…'), ' ', h('span', { class: 'hint' }, 'PNG, JPG, WebP, GIF or ICO, up to 256 KB.')));
      }
      iconCtl.replaceChildren(...rows);
    };
    const pickImage = () => {
      const fi = h('input', { type: 'file', accept: 'image/png,image/jpeg,image/webp,image/gif,image/x-icon,image/vnd.microsoft.icon' });
      fi.addEventListener('change', () => {
        const f = fi.files[0];
        if (!f) return;
        if (f.size > 256 * 1024) { toast('That image is larger than 256 KB.', 'err'); return; }
        const rd = new FileReader();
        rd.onload = () => { setSetting({ icon: { ...st.icon, type: 'image', image: String(rd.result) } }); drawIcon(); };
        rd.readAsDataURL(f);
      });
      fi.click();
    };
    drawIcon();

    const hdrHex = h('input', { class: 'input mono', type: 'text', value: st.header_bg || '', placeholder: 'Default', maxlength: '7', style: { width: '110px' }, 'aria-label': 'Custom header color' });
    hdrHex.addEventListener('change', () => { const v = hdrHex.value.trim(); if (!v || /^#[0-9a-f]{6}$/i.test(v)) setSetting({ header_bg: v.toUpperCase() }, { rerender: true }); else hdrHex.value = st.header_bg; });
    const headers = h('div', { class: 'swatches' },
      h('button', { type: 'button', class: 'swatch default', 'aria-label': 'Default header', 'aria-pressed': String(!st.header_bg), onclick: () => setSetting({ header_bg: '' }, { rerender: true }) }),
      HEADERS.map(([c, l]) => h('button', { type: 'button', class: 'swatch', 'aria-label': l + ' header', 'aria-pressed': String((st.header_bg || '').toUpperCase() === c), style: { background: c }, onclick: () => setSetting({ header_bg: c }, { rerender: true }) })),
      hdrHex);
    const textSeg = seg([['', 'Auto'], ['#F2F4F7', 'Light'], ['#16191F', 'Dark']], st.header_text || '', (v) => setSetting({ header_text: v }, { rerender: true }), 'Header text');
    const layoutSeg = seg(LAYOUTS, st.default_layout, (v) => setSetting({ default_layout: v }, { rerender: true }), 'Default link layout');

    const rows = [
      srow('Dashboard name', 'Shown in the header and the browser tab, for everyone.', name),
      h('div', { class: 'srow' }, h('div', { class: 'srow-label' }, h('b', null, 'Icon'), h('span', null, 'Used in the header and as the browser tab icon.')), iconCtl),
      srow('Default accent', 'For visitors, the tab icon, and anyone who hasn’t picked their own.', accentPicker(st.default_accent, (v) => setSetting({ default_accent: v }, { rerender: true }), false)),
      srow('Default link layout', 'For visitors and anyone who hasn’t picked their own.', layoutSeg),
      srow('Header', 'Default blends with the page. Pick a color to make it a band.', [headers, textSeg]),
    ];
    if (authOn()) {
      const pub = S.pages.filter((p) => p.visibility === 'public');
      const landSel = h('select', { class: 'select', 'aria-label': 'Visitors land on' },
        h('option', { value: '', selected: !st.public_start }, 'The first public page'),
        pub.map((p) => h('option', { value: 'links:' + p.id, selected: st.public_start === 'links:' + p.id }, 'Links · ' + p.name)),
        h('option', { value: 'blank', selected: st.public_start === 'blank' }, 'Blank page'));
      landSel.addEventListener('change', () => setSetting({ public_start: landSel.value }));
      const membersEdit = h('input', { type: 'checkbox', role: 'switch', checked: !!st.members_edit_shared });
      membersEdit.addEventListener('change', async () => {
        // Save right away, then reload: what everyone can edit depends on it.
        S.settings.members_edit_shared = membersEdit.checked;
        try { await api('PUT', '/api/settings', S.settings); await loadState(); renderMain(); } catch (e) { if (e.status !== 401) toast(e.message, 'err'); }
      });
      rows.push(
        srow('Visitors land on', pub.length ? 'Where people who aren’t signed in start.' : 'No page is public yet, so visitors see the sign-in screen.', landSel),
        srow('Members can edit shared things', 'Off: members view shared pages, notes and directory entries; only owners and admins change them. Shared task lists can always be checked off by everyone.',
          h('label', { class: 'check' }, membersEdit, 'Allow editing'), true));
    }
    return card('dashboard', 'Dashboard', 'For everyone · admins only', rows);
  }

  // ---------- users (admins) ----------

  function usersCard() {
    const list = h('div', { class: 'ulist' });
    const users = S.users;
    if (!users) list.append(h('p', { class: 'hint' }, 'Loading…'));
    else users.forEach((u) => {
      const self = me() && u.id === me().id;
      const display = u.name || u.username;
      const roleSeg = seg([['admin', 'Admin'], ['member', 'Member']], u.role, (v) => updateUser(u, { role: v }), 'Role of ' + display);
      list.append(h('div', { class: 'urow' + (u.disabled ? ' off' : '') },
        h('span', { class: 'who-avatar', 'aria-hidden': 'true' }, initials(display)),
        h('div', { class: 'urow-name' }, h('b', null, display, self ? h('span', { class: 'chip' }, 'You') : null), h('span', null, '@' + u.username + (u.disabled ? ' · disabled' : ''))),
        roleSeg,
        h('div', { class: 'urow-act' },
          h('button', { class: 'icon-btn', title: 'Reset password', 'aria-label': 'Reset password for ' + display, onclick: () => resetUserPassword(u) }, ic('key', 15)),
          self ? null : h('button', { class: 'icon-btn', title: u.disabled ? 'Enable' : 'Disable', 'aria-label': (u.disabled ? 'Enable ' : 'Disable ') + display, onclick: () => updateUser(u, { disabled: !u.disabled }) }, ic(u.disabled ? 'check' : 'lock', 15)),
          self ? null : h('button', { class: 'icon-btn', title: 'Delete', 'aria-label': 'Delete ' + display, onclick: () => deleteUser(u) }, ic('trash', 15)))));
    });
    return card('users', 'Users', users ? `${users.length} ${users.length === 1 ? 'person' : 'people'}` : null,
      h('div', { class: 'srow', style: { flexDirection: 'column' } },
        h('div', { style: { display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '12px', flexWrap: 'wrap' } },
          h('div', { class: 'srow-label' }, h('b', null, 'People who can sign in'), h('span', null, 'Everyone has their own pages, notes, tasks and directory. Members see what others share; admins also manage the dashboard and its users.')),
          h('button', { class: 'btn primary', onclick: addUser }, ic('plus', 14), 'Add person')),
        list));
  }

  async function updateUser(u, patch) {
    try {
      const out = await api('PUT', '/api/users/' + u.id, patch);
      Object.assign(u, out);
      if (me() && u.id === me().id) { await loadSession(); await loadState(); renderAll(); }
      else renderMain();
    } catch (e) { if (e.status !== 401) toast(e.message, 'err'); loadSettingsExtras(); }
  }

  function passwordField(placeholder) {
    const inp = h('input', { class: 'input mono', type: 'text', autocomplete: 'new-password', spellcheck: 'false', placeholder: placeholder || '8+ characters', minlength: '8' });
    const gen = h('button', { type: 'button', class: 'btn', onclick: () => { const b = new Uint8Array(9); crypto.getRandomValues(b); inp.value = btoa(String.fromCharCode(...b)).replace(/[+/=]/g, (c) => ({ '+': 'k', '/': 'x', '=': '' })[c]); } }, 'Generate');
    return { inp, el: h('div', { class: 'inline-field' }, inp, gen) };
  }

  function showCredentials(username, password) {
    const text = `Dashboard: ${location.origin}\nUsername: ${username}\nPassword: ${password}`;
    openModal('Share these sign-in details', [
      h('p', null, 'Send them to the person; they can change the password under Settings → Profile.'),
      h('pre', { class: 'creds' }, text),
    ], [{ label: 'Copy', run: () => { copyText(text); return false; } }, { label: 'Done', cls: 'primary', submit: true, run: () => true }]);
  }

  function addUser() {
    const name = h('input', { class: 'input', type: 'text', placeholder: 'Ana Lima', maxlength: '60' });
    const username = h('input', { class: 'input', type: 'text', placeholder: 'ana', maxlength: '40', autocapitalize: 'off', spellcheck: 'false' });
    const pw = passwordField();
    let role = 'member';
    const roleSeg = seg([['member', 'Member'], ['admin', 'Admin']], role, (v) => { role = v; roleSeg.querySelectorAll('button').forEach((b) => b.setAttribute('aria-pressed', String(b.textContent.toLowerCase() === v))); }, 'Role');
    name.addEventListener('input', () => { if (!username.dataset.touched) username.value = (name.value.trim().split(/\s+/)[0] || '').toLowerCase().replace(/[^a-z0-9._-]/g, ''); });
    username.addEventListener('input', () => { username.dataset.touched = '1'; });
    const err = h('p', { class: 'form-error', role: 'alert' });
    openModal('Add person', [
      h('label', { class: 'field' }, 'Name', name),
      h('label', { class: 'field' }, 'Username', username),
      h('div', { class: 'field' }, h('span', null, 'Password'), pw.el),
      h('div', { class: 'field' }, h('span', null, 'Role'), roleSeg, h('span', { class: 'hint' }, 'Members have their own space and see what others share. Admins also manage the dashboard and users.')),
      err,
    ], [
      { label: 'Cancel', run: () => true },
      { label: 'Add person', cls: 'primary', submit: true, run: async () => {
        err.textContent = '';
        if (pw.inp.value.length < 8) { err.textContent = 'Use a password of at least 8 characters, or click Generate.'; return false; }
        try {
          const u = await api('POST', '/api/users', { username: username.value.trim(), name: name.value.trim(), password: pw.inp.value, role });
          await loadSettingsExtras();
          setTimeout(() => showCredentials(u.username, pw.inp.value), 0);
          return true;
        } catch (e) { err.textContent = e.message; return false; }
      } },
    ], name);
  }

  function resetUserPassword(u) {
    const pw = passwordField('New password');
    const err = h('p', { class: 'form-error', role: 'alert' });
    openModal('Reset password for ' + (u.name || u.username), [h('p', null, 'They will be signed out everywhere and need the new password to sign in.'), pw.el, err], [
      { label: 'Cancel', run: () => true },
      { label: 'Set password', cls: 'primary', submit: true, run: async () => {
        if (pw.inp.value.length < 8) { err.textContent = 'Use at least 8 characters, or click Generate.'; return false; }
        try {
          await api('PUT', '/api/users/' + u.id, { password: pw.inp.value });
          if (me() && u.id === me().id) toast('Password changed');
          else setTimeout(() => showCredentials(u.username, pw.inp.value), 0);
          return true;
        } catch (e) { err.textContent = e.message; return false; }
      } },
    ], pw.inp);
  }

  function deleteUser(u) {
    const hand = h('input', { type: 'checkbox', checked: true });
    openModal(`Delete ${u.name || u.username}?`, [
      h('p', null, 'Their account and everything private to them is deleted. This cannot be undone (except by restoring a backup).'),
      h('label', { class: 'check' }, hand, 'Keep what they shared — it becomes mine'),
    ], [
      { label: 'Cancel', run: () => true },
      { label: 'Delete', cls: 'primary', submit: true, run: async () => {
        try {
          await api('DELETE', '/api/users/' + u.id + (hand.checked ? '?handover=1' : ''));
          toast('Deleted');
          await loadState();
          await loadSettingsExtras();
          return true;
        } catch (e) { toast(e.message, 'err'); return false; }
      } },
    ]);
  }

  // ---------- data ----------

  function fileDrop(label, accept, onFile) {
    const drop = h('label', { class: 'drop', tabindex: '0' }, ic('upload', 18), h('span', null, label + ', or ', h('span', { style: { color: 'var(--accent)' } }, 'browse')));
    const fi = h('input', { type: 'file', accept, class: 'sr-only' });
    drop.append(fi);
    fi.addEventListener('change', () => { if (fi.files[0]) onFile(fi.files[0]); fi.value = ''; });
    drop.addEventListener('dragover', (e) => { e.preventDefault(); drop.classList.add('over'); });
    drop.addEventListener('dragleave', () => drop.classList.remove('over'));
    drop.addEventListener('drop', (e) => { e.preventDefault(); drop.classList.remove('over'); const f = e.dataTransfer.files[0]; if (f) onFile(f); });
    return drop;
  }

  function dataCard() {
    const a = S.about;
    const c = a && a.mine;
    const stats = c ? `Yours: ${c.pages} pages · ${c.links} links · ${c.notes} notes · ${c.tasks} tasks · ${c.directory} directory entries` : '';
    const rows = [
      srow('Export my data', 'Everything you own — pages, notes, tasks, directory, scratchpad and preferences — in one JSON file.', h('div', null, h('a', { class: 'btn', href: '/api/export', download: '' }, ic('download', 15), 'Download JSON'))),
      srow('Import my data', 'Replaces everything you own with a DotSlash JSON export. Things others share are not touched. A backup is taken first.', fileDrop('Drop a .json file here', 'application/json,.json', importFile)),
    ];
    if (isAdmin()) {
      const b = S.backups;
      const list = h('div', { class: 'backups' });
      if (!b) list.append(h('div', { class: 'backup' }, h('span', { class: 'when' }, 'Loading…')));
      else if (!b.backups.length) list.append(h('div', { class: 'backup' }, h('span', { class: 'when' }, 'No backups yet.')));
      else b.backups.forEach((x) => list.append(h('div', { class: 'backup' },
        h('span', { class: 'when' }, new Date(x.time).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })),
        h('span', { class: 'why' }, ({ daily: 'Daily', manual: 'Manual', 'before-import': 'Before import', 'before-restore': 'Before restore' })[x.reason] || x.reason),
        h('span', { class: 'size' }, fmtBytes(x.size)),
        h('a', { class: 'icon-btn', href: '/api/backups/' + encodeURIComponent(x.name), download: '', title: 'Download', 'aria-label': 'Download backup ' + x.name }, ic('download', 14)),
        h('button', { class: 'btn', style: { height: '30px', fontSize: '12px' }, onclick: () => restoreBackup(x) }, 'Restore'))));
      rows.push(
        srow('Database', `The whole dashboard — every person, page, note, task and setting — as one SQLite file${a && a.data_bytes ? ` (${fmtBytes(a.data_bytes)})` : ''}.`,
          h('div', { class: 'btn-row' }, h('a', { class: 'btn', href: '/api/db', download: '' }, ic('download', 15), 'Download database'))),
        srow('Restore the database', 'Replaces the whole dashboard with a downloaded .db file. Everyone may need to sign in again. A backup is taken first.', fileDrop('Drop a .db file here', '.db,application/vnd.sqlite3,application/x-sqlite3', restoreDBFile)),
        h('div', { class: 'srow', style: { flexDirection: 'column' } },
          h('div', { style: { display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '12px', flexWrap: 'wrap' } },
            h('div', { class: 'srow-label' }, h('b', null, 'Automatic backups'), h('span', null, `Of the whole database: daily, and before every import or restore · last ${b ? b.keep : 7} kept`)),
            h('button', { class: 'btn', onclick: backupNow }, 'Back up now')),
          list));
    } else if (authOn()) {
      rows.push(srow('Backups', 'Backups of the whole dashboard are made automatically and managed by an admin.', null));
    }
    return card('data', 'Data & backups', stats, rows);
  }

  async function afterReplace(msg) {
    await loadSession();
    if (!signedIn() && authOn()) { await boot(); return; }
    await loadState();
    S.noteCache = {};
    toast(msg);
    refreshFavicon();
    renderAll();
    loadSettingsExtras();
  }

  async function importFile(f) {
    if (f.size > 50 * 1024 * 1024) { toast('That file is larger than 50 MB.', 'err'); return; }
    const text = await f.text();
    let sum;
    try { sum = (await api('POST', '/api/import?dry_run=1', text, { raw: true })).summary; } catch (e) { if (e.status !== 401) toast(e.message, 'err'); return; }
    const ok = await confirmBox('Replace your things?',
      `“${f.name}” has ${sum.pages} page${sum.pages === 1 ? '' : 's'}, ${sum.links} links, ${sum.notes} notes, ${sum.task_lists} task lists (${sum.tasks} tasks) and ${sum.directory} directory entries. Everything you own is replaced; things others share stay. A backup is taken first.`,
      'Import and replace', true);
    if (!ok) return;
    try {
      await api('POST', '/api/import', text, { raw: true });
      await afterReplace('Imported');
    } catch (e) { if (e.status !== 401) toast(e.message, 'err'); }
  }

  async function restoreDBFile(f) {
    const ok = await confirmBox('Replace the whole dashboard?', `Everything — all people, pages, notes, tasks, the directory and settings — is replaced with “${f.name}”. The current database is backed up first.`, 'Restore', true);
    if (!ok) return;
    try {
      await api('POST', '/api/db/restore', await f.arrayBuffer(), { raw: true });
      await afterReplace('Database restored');
    } catch (e) { if (e.status !== 401) toast(e.message, 'err'); }
  }

  async function backupNow() {
    try { await api('POST', '/api/backups'); toast('Backup saved'); loadSettingsExtras(); } catch (e) { if (e.status !== 401) toast(e.message, 'err'); }
  }

  async function restoreBackup(x) {
    const ok = await confirmBox('Restore this backup?', `The whole dashboard is replaced with the backup from ${new Date(x.time).toLocaleString()}. The current database is backed up first.`, 'Restore', true);
    if (!ok) return;
    try {
      await api('POST', '/api/backups/' + encodeURIComponent(x.name) + '/restore');
      await afterReplace('Backup restored');
    } catch (e) { if (e.status !== 401) toast(e.message, 'err'); }
  }

  // ---------- security ----------

  function securityCard() {
    const se = S.session || {};
    const rows = [];
    if (!se.auth_enabled) {
      rows.push(h('div', { class: 'srow' }, h('div', { class: 'notice' }, 'Sign-in is off: anyone who can open this page can see and change everything, and there is only one account. Set DS_PASSWORD on the server to turn on sign-in, user accounts and sharing.')));
      return card('security', 'Security', null, rows);
    }
    const list = h('div', { class: 'vis-list' });
    const draw = () => {
      const manageable = S.pages.filter((p) => p.can_manage);
      if (!manageable.length) { list.replaceChildren(h('div', { class: 'vis-row' }, h('span', { class: 'hint' }, 'You have no pages yet.'))); return; }
      list.replaceChildren(...manageable.map((p) => h('div', { class: 'vis-row' },
        h('span', { class: 'vis-name' }, ic(p.visibility === 'public' ? 'globe' : p.visibility === 'shared' ? 'users' : 'lock', 14), p.name, mine(p) ? null : h('span', { class: 'smark-who' }, p.owner_name)),
        seg([['private', 'Private'], ['shared', 'Shared'], ...(isAdmin() ? [['public', 'Public']] : [])], p.visibility, async (v) => {
          await setVisibility('page', p, v, async () => { await loadState(); draw(); renderSub(); });
        }, 'Who sees ' + p.name))));
    };
    draw();
    rows.push(srow('Who sees your pages', `Private: only you. Shared: everyone who signs in.${isAdmin() ? ' Public: anyone, without signing in.' : ' Admins can also make pages public.'} New pages start private.`, list));
    rows.push(srow('Visitors', 'People who aren’t signed in see public link pages only — never notes, tasks, the directory, the scratchpad or settings.', null));
    rows.push(srow('Sign-in', isAdmin() ? 'Add and remove people under Users. The first admin comes from DS_USER and DS_PASSWORD on first start.' : 'Ask an admin to change your role or reset your password.', null));
    return card('security', 'Security', null, rows);
  }

  function aboutCard() {
    const a = S.about;
    return card('about', 'About', null, h('div', { class: 'kv' },
      h('div', null, h('span', null, 'Version'), h('code', null, (S.session && S.session.version) || '')),
      a && a.data_file ? h('div', null, h('span', null, 'Database'), h('code', null, a.data_file)) : null,
      h('div', null, h('span', null, 'Memory in use'), h('code', null, a ? fmtBytes(a.memory) : '…')),
      a && a.mine && authOn() ? h('div', null, h('span', null, 'People'), h('code', null, String(a.mine.users))) : null,
      h('div', null, h('span', null, 'Built with'), h('code', null, a ? a.go : '…'))));
  }

  async function logout() {
    try { await api('POST', '/api/logout'); } catch (e) { /* ignore */ }
    await boot();
  }

  // ------------------------------------------------------------------
  // Directory — people, businesses, places and devices in one sheet
  // ------------------------------------------------------------------

  const DIR_COLS = [
    ['name', 'Name', 'Name'], ['category', 'Category', 'People, Services, Devices…'], ['phone', 'Phone', '+1 555 0100'],
    ['email', 'Email', 'name@example.com'], ['address', 'Address', '12 Main St, Springfield'], ['url', 'URL', 'https://…'],
    ['ip', 'IP', '192.168.1.10'], ['mac', 'MAC', 'aa:bb:cc:dd:ee:ff'], ['notes', 'Notes', 'Account #, hours, anything'],
  ];
  const DIR_LABEL = Object.fromEntries(DIR_COLS.map(([k, l]) => [k, ['ip', 'mac', 'url'].includes(k) ? l : l.toLowerCase()]));
  const touchOnly = () => window.matchMedia && window.matchMedia('(hover: none)').matches;

  function dirFiltered() {
    const c = S.dirCat;
    return S.directory.filter((e) => c === 'all' || (c === '~shared' ? isShared(e) : c === '-' ? !e.category : e.category === c));
  }

  function dirSorted() {
    const { key, dir } = S.dirSort;
    return dirFiltered().slice().sort((a, b) => {
      const x = (a[key] || '').toLowerCase(), y = (b[key] || '').toLowerCase();
      if (!x && y) return 1;
      if (x && !y) return -1;
      return x.localeCompare(y, undefined, { numeric: true }) * dir || (a.name || '').localeCompare(b.name || '');
    });
  }

  function mapsURL(address) {
    const q = encodeURIComponent(address);
    return /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent) ? `https://maps.apple.com/?q=${q}` : `https://www.google.com/maps/search/?api=1&query=${q}`;
  }

  function renderDirectory() {
    const editing = S.editing;
    const wrap = h('div', { class: 'wrap' + (editing ? ' editing' : '') });
    if (!S.directory.length && !editing) {
      wrap.append(h('div', { class: 'empty' }, ic('user', 22), h('h2', null, 'Your directory is empty'),
        h('p', null, 'People, services, places and devices — phone numbers, addresses, URLs, IPs and MACs, all in one place.'),
        h('button', { class: 'btn primary', onclick: () => { startEdit(); addEntry(); } }, ic('plus', 14), 'Add entry')));
      return wrap;
    }
    const rows = editing ? dirFiltered() : dirSorted();
    // In view mode, show only the columns these rows use (and no Category column inside one category).
    const cols = editing ? DIR_COLS : DIR_COLS.filter(([k]) => k === 'name' || (!(k === 'category' && S.dirCat !== 'all') && rows.some((e) => e[k])));
    const head = h('tr', null, cols.map(([k, label]) => {
      const on = S.dirSort.key === k;
      return h('th', { scope: 'col', class: 'col-' + k, 'aria-sort': !editing && on ? (S.dirSort.dir > 0 ? 'ascending' : 'descending') : null },
        editing ? label : h('button', { class: 'th-btn', onclick: () => { S.dirSort = { key: k, dir: on ? -S.dirSort.dir : 1 }; renderMain(); } },
          label, on ? h('span', { class: 'sort' }, S.dirSort.dir > 0 ? '↑' : '↓') : null));
    }), editing ? h('th', { class: 'col-act' }, h('span', { class: 'sr-only' }, 'Actions')) : null);
    const body = h('tbody');
    rows.forEach((e) => body.append(editing ? entryEditRow(e) : entryRow(e, cols)));
    if (!rows.length) body.append(h('tr', null, h('td', { colspan: String(cols.length + 1), class: 'hint' }, 'Nothing in this category.')));
    wrap.append(h('div', { class: 'sheet-wrap' }, h('table', { class: 'sheet dir' + (editing ? ' editing' : ' viewing') }, h('thead', null, head), body)));
    if (editing) wrap.append(h('button', { class: 'add-btn sheet-add', onclick: addEntry }, ic('plus', 14), 'Add entry'));
    else wrap.append(h('p', { class: 'hint dir-hint' }, touchOnly() ? 'Tap a value to copy it.' : 'Double-click any value to copy it.'));
    flashLater();
    return wrap;
  }

  function entryRow(e, cols) {
    const tr = h('tr', { dataset: { item: e.id } });
    cols.forEach(([k, label]) => {
      const v = e[k] || '';
      const td = h('td', { class: 'col-' + k, dataset: { label } });
      if (!v) { tr.append(td); return; }
      if (k === 'category') { td.append(h('span', { class: 'chip' }, v)); tr.append(td); return; }
      const shown = k === 'url' ? hostLabel(v) : v;
      const val = h('span', { class: 'val' + (['ip', 'mac', 'phone'].includes(k) ? ' mono' : ''), title: touchOnly() ? 'Tap to copy' : 'Double-click to copy' }, shown);
      const copy = () => copyValue(v, DIR_LABEL[k]);
      val.addEventListener('dblclick', copy);
      if (touchOnly()) val.addEventListener('click', copy);
      const act = {
        phone: ['phone', 'Call', 'tel:' + v.replace(/[^\d+*#,]/g, '')],
        email: ['mail', 'Write an email', 'mailto:' + v],
        address: ['mappin', 'Open in maps', mapsURL(v)],
        url: ['external', 'Open in a new tab', safeHref(/^[a-z][a-z0-9+.-]*:/i.test(v) ? v : 'https://' + v)],
      }[k];
      if (k === 'name') { td.replaceChildren(...[val, sharedMark(e)].filter(Boolean)); } else td.append(val);
      if (act) {
        const external = k === 'url' || k === 'address';
        td.append(h('a', { class: 'cell-act', href: act[2], title: act[1], 'aria-label': `${act[1]}: ${v}`, target: external ? '_blank' : null, rel: external ? 'noopener noreferrer' : null }, ic(act[0], 14)));
      }
      tr.append(td);
    });
    return tr;
  }

  function entryEditRow(e) {
    const tr = h('tr', { dataset: { item: e.id }, class: e.can_edit ? null : 'readonly', title: e.can_edit ? null : `Shared by ${e.owner_name} — read-only` });
    DIR_COLS.forEach(([k, label, ph], col) => {
      const inp = h('input', { class: 'cell' + (['ip', 'mac'].includes(k) ? ' mono' : ''), type: k === 'email' ? 'email' : k === 'phone' ? 'tel' : k === 'url' ? 'url' : 'text', value: e[k] || '', placeholder: e.can_edit ? ph : '', 'aria-label': label, spellcheck: ['ip', 'mac', 'url', 'email'].includes(k) ? 'false' : null, dataset: { col: String(col) }, readonly: !e.can_edit });
      if (k === 'category') inp.setAttribute('list', 'dir-categories');
      const save = async () => {
        if ((e[k] || '') === inp.value.trim()) return;
        const prev = e[k];
        e[k] = inp.value.trim();
        tr.classList.add('saving');
        try { Object.assign(e, await api('PUT', '/api/directory/' + e.id, { [k]: e[k] })); inp.value = e[k] || ''; }
        catch (err) { e[k] = prev; inp.value = prev || ''; if (err.status !== 401) toast(err.message, 'err'); }
        tr.classList.remove('saving');
        if (k === 'category') renderSub();
      };
      inp.addEventListener('change', save);
      inp.addEventListener('keydown', (ev) => {
        if (ev.key === 'Enter' || ev.key === 'ArrowDown' || ev.key === 'ArrowUp') {
          ev.preventDefault();
          const all = [...tr.parentNode.children];
          const next = all[all.indexOf(tr) + (ev.key === 'ArrowUp' ? -1 : 1)];
          if (next && next.querySelector(`[data-col="${col}"]`)) next.querySelector(`[data-col="${col}"]`).focus();
          else if (ev.key === 'Enter') { inp.blur(); addEntry(); }
        }
        if (ev.key === 'Escape') { inp.value = e[k] || ''; inp.blur(); }
      });
      tr.append(h('td', { class: 'col-' + k }, inp));
    });
    const shareCell = e.can_manage && authOn()
      ? h('button', { class: 'icon-btn' + (isShared(e) ? ' on' : ''), 'aria-pressed': String(isShared(e)), 'aria-label': (isShared(e) ? 'Make private: ' : 'Share: ') + (e.name || 'entry'), title: isShared(e) ? 'Shared with everyone signed in — click to make private' : 'Private — click to share', onclick: () => setVisibility('entry', e, isShared(e) ? 'private' : 'shared') }, ic(isShared(e) ? 'users' : 'lock', 14))
      : sharedMark(e);
    tr.append(h('td', { class: 'col-act' }, shareCell, !e.can_manage ? null : h('button', { class: 'icon-btn', 'aria-label': 'Delete ' + (e.name || 'entry'), onclick: async () => {
      const filled = DIR_COLS.some(([k]) => e[k]);
      if (filled && !(await confirmBox(`Delete ${e.name || 'this entry'}?`, 'This cannot be undone (except by restoring a backup).', 'Delete', true))) return;
      try { await api('DELETE', '/api/directory/' + e.id); S.directory = S.directory.filter((x) => x.id !== e.id); renderSub(); renderMain(); }
      catch (err) { if (err.status !== 401) toast(err.message, 'err'); }
    } }, ic('trash', 14))));
    if (!$('#dir-categories')) document.body.append(h('datalist', { id: 'dir-categories' }));
    $('#dir-categories').replaceChildren(...[...new Set(S.directory.map((x) => x.category).filter(Boolean))].sort().map((g) => h('option', { value: g })));
    return tr;
  }

  async function addEntry() {
    if (!S.editing) startEdit();
    try {
      const cat = S.dirCat !== 'all' && S.dirCat !== '-' && S.dirCat !== '~shared' ? S.dirCat : '';
      const e = await api('POST', '/api/directory', { name: '', category: cat, visibility: S.dirCat === '~shared' ? 'shared' : 'private' });
      S.directory.push(e);
      renderSub(); renderMain();
      requestAnimationFrame(() => {
        const el = document.querySelector(`tr[data-item="${CSS.escape(e.id)}"] input`);
        if (el) { el.focus(); el.scrollIntoView({ block: 'nearest' }); }
      });
    } catch (err) { if (err.status !== 401) toast(err.message, 'err'); }
  }

  // Rows added in edit mode but never filled in are removed on Save/Close.
  async function dropEmptyEntries() {
    const empty = S.directory.filter((e) => e.can_manage && mine(e) && !DIR_COLS.some(([k]) => (e[k] || '').trim()));
    for (const e of empty) {
      try { await api('DELETE', '/api/directory/' + e.id); } catch (err) { /* ignore */ }
    }
    if (empty.length) S.directory = S.directory.filter((e) => !empty.includes(e));
  }

  // ------------------------------------------------------------------
  // Scratchpad — a floating panel that stays open across pages
  // ------------------------------------------------------------------

  const PS = { el: null, text: null, status: null, dirty: false, saving: false, again: false, timer: null };

  const padOpen = () => scratchOn() && localGet('ds-pad-open') === '1';

  function togglePad() {
    localSet('ds-pad-open', padOpen() ? null : '1');
    renderPad();
    renderDock();
    if (padOpen() && PS.text) PS.text.focus();
  }

  function padStatus(kind, msg, action) {
    if (!PS.status) return;
    PS.status.replaceChildren();
    if (kind === 'saved') PS.status.append(ic('check', 11), S.scratchpad.modified_at ? 'Saved ' + timeAgo(S.scratchpad.modified_at) : 'Saved');
    else if (kind === 'saving') PS.status.append('Saving…');
    else if (kind === 'dirty') PS.status.append('Typing…');
    else {
      PS.status.append(h('span', { class: 'warn' }, msg));
      if (action) PS.status.append(' ', h('button', { class: 'link-btn', onclick: action[1] }, action[0]));
    }
  }

  function renderPad() {
    if (!shell) return;
    const pad = shell.pad;
    if (!padOpen()) { pad.hidden = true; return; }
    if (!PS.el || PS.el !== pad || !pad.firstChild) {
      PS.el = pad;
      PS.text = h('textarea', { class: 'pad-text', spellcheck: 'true', placeholder: 'Jot something down — it saves by itself and follows you to every page and device.', 'aria-label': 'Scratchpad text' });
      PS.status = h('span', { class: 'pad-status' });
      const headEl = h('div', { class: 'pad-head' },
        h('span', { class: 'pad-title' }, ic('pad', 14), 'Scratchpad'),
        PS.status,
        h('button', { class: 'icon-btn', 'aria-label': 'Copy scratchpad', title: 'Copy', onclick: () => copyText(PS.text.value) }, ic('copy', 14)),
        h('button', { class: 'icon-btn', 'aria-label': 'Close scratchpad', title: 'Close', onclick: togglePad }, ic('x', 14)));
      pad.replaceChildren(headEl, PS.text);
      PS.text.addEventListener('input', () => { PS.dirty = true; padStatus('dirty'); clearTimeout(PS.timer); PS.timer = setTimeout(() => savePad(), 900); });
      PS.text.addEventListener('blur', () => { if (PS.dirty) savePad(); });
      makeDraggable(pad, headEl);
      try {
        const sz = JSON.parse(localGet('ds-pad-size') || 'null');
        if (sz) { pad.style.width = sz.w + 'px'; pad.style.height = sz.h + 'px'; }
      } catch (e) { /* ignore */ }
      if (window.ResizeObserver) {
        const ro = new ResizeObserver(debounce(() => { if (!pad.hidden && pad.offsetWidth) localSet('ds-pad-size', JSON.stringify({ w: pad.offsetWidth, h: pad.offsetHeight })); }, 300));
        ro.observe(pad);
      }
    }
    if (!PS.dirty && document.activeElement !== PS.text) PS.text.value = S.scratchpad.content || '';
    pad.hidden = false;
    placePad();
    padStatus(PS.dirty ? 'dirty' : 'saved');
  }

  function placePad() {
    const pad = shell.pad;
    if (window.innerWidth <= 760) { pad.style.left = ''; pad.style.top = ''; return; }
    let pos = null;
    try { pos = JSON.parse(localGet('ds-pad-pos') || 'null'); } catch (e) { /* ignore */ }
    const w = pad.offsetWidth || 340, hgt = pad.offsetHeight || 280;
    const x = pos ? pos.x : window.innerWidth - w - 24;
    const y = pos ? pos.y : window.innerHeight - hgt - 80;
    pad.style.left = Math.max(8, Math.min(x, window.innerWidth - w - 8)) + 'px';
    pad.style.top = Math.max(8, Math.min(y, window.innerHeight - 48)) + 'px';
  }

  function makeDraggable(panel, handle) {
    handle.addEventListener('pointerdown', (e) => {
      if (e.button !== 0 || e.target.closest('button') || window.innerWidth <= 760) return;
      e.preventDefault();
      const r = panel.getBoundingClientRect();
      const dx = e.clientX - r.left, dy = e.clientY - r.top;
      handle.setPointerCapture(e.pointerId);
      panel.classList.add('dragging');
      const move = (ev) => {
        panel.style.left = Math.max(8, Math.min(ev.clientX - dx, window.innerWidth - panel.offsetWidth - 8)) + 'px';
        panel.style.top = Math.max(8, Math.min(ev.clientY - dy, window.innerHeight - 48)) + 'px';
      };
      const up = () => {
        handle.removeEventListener('pointermove', move);
        handle.removeEventListener('pointerup', up);
        panel.classList.remove('dragging');
        localSet('ds-pad-pos', JSON.stringify({ x: parseInt(panel.style.left, 10), y: parseInt(panel.style.top, 10) }));
      };
      handle.addEventListener('pointermove', move);
      handle.addEventListener('pointerup', up);
    });
  }
  window.addEventListener('resize', debounce(() => { if (shell && padOpen()) placePad(); }, 150));

  async function savePad(keepalive, force) {
    clearTimeout(PS.timer);
    if (!PS.text || !PS.dirty || !signedIn()) return;
    if (PS.saving) { PS.again = true; return; }
    PS.saving = true;
    padStatus('saving');
    const content = PS.text.value;
    try {
      const pad = await api('PUT', '/api/scratchpad', { content, base_modified: force ? '' : S.scratchpad.modified_at || '' }, { keepalive });
      S.scratchpad = pad;
      if (PS.text.value === content) PS.dirty = false;
      padStatus(PS.dirty ? 'dirty' : 'saved');
    } catch (e) {
      if (e.status === 409 && e.data && e.data.scratchpad) {
        const theirs = e.data.scratchpad;
        padStatus('error', 'Changed elsewhere.', ['Load theirs', () => { S.scratchpad = theirs; PS.dirty = false; PS.text.value = theirs.content; padStatus('saved'); }]);
        PS.status.append(' ', h('button', { class: 'link-btn', onclick: () => { S.scratchpad = theirs; savePad(false, true); } }, 'Keep mine'));
      } else padStatus('error', e.status === 401 ? 'Sign in to save' : 'Not saved');
    } finally {
      PS.saving = false;
      if (PS.again) { PS.again = false; savePad(); }
    }
  }

  // ------------------------------------------------------------------
  // Main area
  // ------------------------------------------------------------------

  function renderMain() {
    if (!shell) return;
    const sec = S.route.section;
    let view;
    if (!signedIn() && ['notes', 'tasks', 'directory'].includes(sec)) view = lockedView('This section is private', 'Sign in to see it.');
    else if (sec === 'links') view = renderLinks();
    else if (sec === 'notes') view = renderNotes();
    else if (sec === 'tasks') view = renderTasks();
    else if (sec === 'directory') view = renderDirectory();
    else if (sec === 'settings') view = renderSettings();
    else view = h('div', { class: 'blank' }, h('span', null, 'Press ', h('kbd', null, '/'), ' to search'));
    shell.main.replaceChildren(view);
  }

  // ------------------------------------------------------------------
  // Search
  // ------------------------------------------------------------------

  let results = [];
  let sel = -1;

  function setupSearch() {
    const input = shell.input;
    const run = debounce(async () => {
      const q = input.value.trim();
      if (!q) { closeSearch(); return; }
      try {
        results = await api('GET', '/api/search?q=' + encodeURIComponent(q), undefined, { quiet401: true });
      } catch (e) { results = []; }
      if (input.value.trim() !== q) return;
      sel = results.length ? 0 : -1;
      drawResults();
    }, 120);
    input.addEventListener('input', run);
    input.addEventListener('focus', () => { if (input.value.trim() && results.length) drawResults(); });
    input.addEventListener('keydown', (e) => {
      if (e.key === 'ArrowDown') { e.preventDefault(); if (results.length) { sel = (sel + 1) % results.length; drawResults(); } }
      else if (e.key === 'ArrowUp') { e.preventDefault(); if (results.length) { sel = (sel - 1 + results.length) % results.length; drawResults(); } }
      else if (e.key === 'Enter') { e.preventDefault(); if (sel >= 0 && results[sel]) activate(results[sel], e.metaKey || e.ctrlKey); }
      else if (e.key === 'Escape') { input.value = ''; closeSearch(); input.blur(); }
    });
  }

  function drawResults() {
    const box = shell.results;
    box.replaceChildren();
    if (!shell.input.value.trim()) { box.hidden = true; return; }
    if (!results.length) {
      box.append(h('div', { class: 'results-empty' }, 'No matches'));
    } else {
      results.forEach((r, i) => {
        let glyph;
        if (r.kind === 'link') glyph = linkIcon({ name: r.title, url: r.url, icon: r.icon }, 'icon');
        else glyph = h('span', { class: 'result-glyph' }, ic({ note: 'note', task: 'task', snippet: 'code', entry: 'user' }[r.kind] || 'note', 15));
        const attrs = { class: 'result' + (i === sel ? ' sel' : ''), role: 'option', 'aria-selected': String(i === sel) };
        const kids = [glyph, h('span', { class: 'result-main' }, h('span', { class: 'result-title' }, r.title), h('span', { class: 'result-ctx' }, r.context || hostLabel(r.url))), h('span', { class: 'result-kind' }, r.kind === 'task' && r.done ? 'done' : r.kind === 'snippet' ? 'snippet · copy' : r.kind === 'entry' ? 'directory' : r.kind)];
        let el;
        if (r.kind === 'link') el = h('a', { ...attrs, href: safeHref(r.url), target: r.new_tab ? '_blank' : null, rel: 'noopener noreferrer', onclick: () => closeSearch(true) }, kids);
        else el = h('button', { ...attrs, type: 'button', onclick: () => activate(r) }, kids);
        el.addEventListener('mousemove', () => { if (sel !== i) { sel = i; box.querySelectorAll('.result').forEach((x, j) => x.classList.toggle('sel', j === i)); } });
        box.append(el);
      });
    }
    box.hidden = false;
    const s = box.querySelector('.sel');
    if (s) s.scrollIntoView({ block: 'nearest' });
  }

  function activate(r, newTab) {
    closeSearch(true);
    if (r.kind === 'link') {
      if (r.new_tab || newTab) window.open(safeHref(r.url), '_blank', 'noopener,noreferrer');
      else location.href = safeHref(r.url);
      return;
    }
    shell.input.value = '';
    shell.input.blur();
    if (r.kind === 'note') { S.noteFilter = 'all'; go('#/notes/' + r.id); }
    else if (r.kind === 'snippet') { if (newTab) { S.noteFilter = 'all'; go('#/notes/' + r.id); } else copyValue(r.content, r.title); }
    else if (r.kind === 'entry') { S.flashItem = r.id; S.dirCat = 'all'; go('#/directory'); }
    else { S.flashItem = r.id; S.taskFilter = r.done ? 'all' : (S.taskFilter === 'done' ? 'all' : S.taskFilter); go('#/tasks'); }
  }

  function closeSearch(keepText) {
    if (!shell) return;
    shell.results.hidden = true;
    if (!keepText) { shell.input.value = ''; results = []; }
  }

  // ------------------------------------------------------------------
  // Modals, menus, toasts
  // ------------------------------------------------------------------

  function openModal(title, bodyKids, actions, focusEl, opts = {}) {
    const prev = document.activeElement;
    const overlay = h('div', { class: 'overlay' });
    const form = h('form', { class: 'modal' + (opts.flat ? ' flat' : ''), role: 'dialog', 'aria-modal': 'true', 'aria-label': title, novalidate: true });
    const close = (result) => {
      overlay.remove();
      document.removeEventListener('keydown', onKey, true);
      if (prev && prev.focus) prev.focus();
      if (opts.onClose) opts.onClose(result);
    };
    const onKey = (e) => {
      if (e.key !== 'Escape' || opts.locked) return;
      if (document.querySelector('.ipick:not([hidden])')) return; // the icon picker closes first
      e.stopPropagation();
      close(null);
    };
    document.addEventListener('keydown', onKey, true);
    const foot = h('div', { class: 'modal-foot' });
    const right = h('div', { class: 'right' });
    let submitAction = null;
    actions.filter(Boolean).forEach((a) => {
      const b = h('button', { type: a.submit ? 'submit' : 'button', class: 'btn ' + (a.cls || '') }, a.label);
      if (a.submit) submitAction = a;
      else b.addEventListener('click', async () => { const r = await a.run(); if (r !== false) close(r); });
      (a.left ? foot : right).append(b);
    });
    foot.append(right);
    form.addEventListener('submit', async (e) => { e.preventDefault(); if (!submitAction) return; const r = await submitAction.run(); if (r !== false) close(r); });
    form.append(
      h('div', { class: 'modal-head' }, h('h3', null, title), opts.locked ? null : h('button', { type: 'button', class: 'icon-btn', 'aria-label': 'Close', onclick: () => close(null) }, ic('x', 16))),
      h('div', { class: 'modal-body' }, bodyKids),
      foot);
    overlay.append(form);
    overlay.addEventListener('mousedown', (e) => { if (e.target === overlay && !opts.locked) close(null); });
    document.body.append(overlay);
    requestAnimationFrame(() => { const f = focusEl || form.querySelector('input,select,textarea,button.primary'); if (f) f.focus(); });
    return close;
  }

  function confirmBox(title, msg, okLabel = 'OK', danger = false) {
    return new Promise((resolve) => {
      openModal(title, [h('p', null, msg)], [
        { label: 'Cancel', run: () => { resolve(false); return true; } },
        { label: okLabel, cls: danger ? 'primary' : 'primary', submit: true, run: () => { resolve(true); return true; } },
      ], null, { onClose: (r) => { if (r == null) resolve(false); } });
    });
  }

  function promptBox(title, label, value = '', okLabel = 'OK') {
    return new Promise((resolve) => {
      const input = h('input', { class: 'input', type: 'text', value, maxlength: '80' });
      let done = false;
      openModal(title, [h('label', { class: 'field' }, label, input)], [
        { label: 'Cancel', run: () => { done = true; resolve(null); return true; } },
        { label: okLabel, cls: 'primary', submit: true, run: () => { const v = input.value.trim(); if (!v) { input.focus(); return false; } done = true; resolve(v); return true; } },
      ], input, { onClose: () => { if (!done) resolve(null); } });
      requestAnimationFrame(() => input.select());
    });
  }

  let menuEl = null;
  function openMenu(anchor, items) {
    closeMenu();
    menuEl = h('div', { class: 'menu', role: 'menu' });
    items.filter(Boolean).forEach((it) => {
      if (it === '-') { menuEl.append(h('hr')); return; }
      const [label, fn, danger] = it;
      menuEl.append(h('button', { type: 'button', role: 'menuitem', class: danger ? 'danger' : '', onclick: () => { closeMenu(); fn(); } }, label));
    });
    document.body.append(menuEl);
    const r = anchor.getBoundingClientRect();
    const mw = menuEl.offsetWidth, mh = menuEl.offsetHeight;
    menuEl.style.left = Math.max(8, Math.min(r.left, window.innerWidth - mw - 8)) + 'px';
    menuEl.style.top = (r.bottom + 6 + mh > window.innerHeight ? r.top - mh - 6 : r.bottom + 6) + 'px';
    menuEl.querySelector('button').focus();
    setTimeout(() => document.addEventListener('mousedown', menuOutside), 0);
  }
  function menuOutside(e) { if (menuEl && !menuEl.contains(e.target)) closeMenu(); }
  function closeMenu() { if (menuEl) { menuEl.remove(); menuEl = null; } document.removeEventListener('mousedown', menuOutside); }

  function toast(msg, kind, actionLabel, action) {
    if (!shell) return;
    const t = h('div', { class: 'toast' + (kind === 'err' ? ' err' : ''), role: kind === 'err' ? 'alert' : 'status' }, h('span', null, msg));
    if (actionLabel) t.append(h('button', { type: 'button', onclick: () => { t.remove(); action(); } }, actionLabel));
    shell.toasts.append(t);
    setTimeout(() => t.remove(), kind === 'err' ? 7000 : 2600);
  }

  // ------------------------------------------------------------------
  // Login
  // ------------------------------------------------------------------

  function loginForm(onDone, full) {
    const user = h('input', { class: 'input', type: 'text', autocomplete: 'username', value: '', 'aria-label': 'Username', required: true });
    const pass = h('input', { class: 'input', type: 'password', autocomplete: 'current-password', 'aria-label': 'Password', required: true });
    const remember = h('input', { type: 'checkbox', checked: true });
    const err = h('p', { class: 'form-error', role: 'alert' });
    const submit = async () => {
      err.textContent = '';
      try {
        await api('POST', '/api/login', { username: user.value, password: pass.value, remember: remember.checked }, { quiet401: true });
      } catch (e) {
        err.textContent = e.message;
        pass.select();
        return false;
      }
      return true;
    };
    const body = [
      h('p', { class: 'hint' }, full ? 'This dashboard is private.' : 'Viewing stays open; changes need a sign-in.'),
      h('label', { class: 'field' }, 'Username', user),
      h('label', { class: 'field' }, 'Password', pass),
      h('label', { class: 'check' }, remember, `Keep me signed in for ${(S.session && S.session.session_days) || 30} days`),
      err,
    ];
    return { body, submit, user, onDone };
  }

  function openLogin(then) {
    if (S.session && !S.session.auth_enabled) { if (then) then(); return; }
    const f = loginForm();
    openModal('Sign in', f.body, [
      { label: 'Cancel', run: () => true },
      { label: 'Sign in', cls: 'primary', submit: true, run: async () => {
        const ok = await f.submit();
        if (!ok) return false;
        await loadSession();
        await loadState();
        renderAll();
        if (S.route.section === 'settings') loadSettingsExtras();
        armIdle();
        toast('Signed in');
        if (then) setTimeout(then, 0);
        return true;
      } },
    ], f.user, { flat: true });
  }

  function footerEl() {
    return h('footer', { class: 'foot' }, 'With ', h('span', { class: 'heart', role: 'img', 'aria-label': 'love' }, '❤️'), ' ',
      h('a', { href: (S.session && S.session.repo) || 'https://github.com/mardix/dotslash', target: '_blank', rel: 'noopener noreferrer' }, 'DotSlash'));
  }

  function renderLoginPage() {
    const app = $('#app');
    shell = null;
    const f = loginForm(null, true);
    const brand = h('div', { class: 'login-brand' }, appIcon('app-icon login-logo'), h('span', { class: 'login-name' }, (S.settings && S.settings.title) || 'DotSlash'));
    const form = h('form', { class: 'modal flat', novalidate: true },
      h('div', { class: 'modal-head' }, h('h3', null, 'Sign in')),
      h('div', { class: 'modal-body' }, f.body),
      h('div', { class: 'modal-foot' }, h('div', { class: 'right' }, h('button', { type: 'submit', class: 'btn primary' }, 'Sign in'))));
    form.addEventListener('submit', async (e) => { e.preventDefault(); if (await f.submit()) boot(); });
    app.replaceChildren(h('div', { class: 'login-page' }, h('div', { class: 'login-main' }, brand, form), footerEl()));
    requestAnimationFrame(() => f.user.focus());
  }

  // ------------------------------------------------------------------
  // Idle return
  // ------------------------------------------------------------------

  let idleTimer = null;
  function armIdle() {
    clearTimeout(idleTimer);
    const m = S.settings ? Number(S.settings.idle_minutes) : 0;
    if (!m || S.settings.start_on === 'last') return;
    idleTimer = setTimeout(() => {
      const busy = S.editing || S.noteEdit || $('.overlay') || (document.activeElement && /^(INPUT|TEXTAREA|SELECT)$/.test(document.activeElement.tagName) && document.activeElement.id !== 'q');
      if (busy) { armIdle(); return; }
      const target = startHash();
      if (location.hash !== target) go(target);
      armIdle();
    }, m * 60 * 1000);
  }
  ['pointerdown', 'pointermove', 'keydown', 'wheel', 'touchstart', 'scroll'].forEach((ev) =>
    window.addEventListener(ev, debounce(armIdle, 500), { passive: true }));

  // ------------------------------------------------------------------
  // Global keys
  // ------------------------------------------------------------------

  document.addEventListener('keydown', (e) => {
    if (!shell) return;
    const t = e.target;
    const typing = t && (/^(INPUT|TEXTAREA|SELECT)$/.test(t.tagName) || t.isContentEditable);
    if ($('.overlay')) return;
    if (e.key === 'Escape') closeMenu();
    if (typing || e.metaKey || e.ctrlKey || e.altKey) return;
    if (e.key === '/') { e.preventDefault(); shell.input.focus(); shell.input.select(); }
    else if (e.key === 'e' && !S.editing && S.route.section !== 'settings') { e.preventDefault(); startEdit(); }
  });

  window.addEventListener('beforeunload', (e) => {
    if (PS.dirty) savePad(true);
    if ((S.editing && S.dirty) || (NS.id && NS.title && S.noteCache[NS.id] && (NS.title.value !== S.noteCache[NS.id].title || NS.text.value !== S.noteCache[NS.id].content))) {
      flushNote(true);
      e.preventDefault();
      e.returnValue = '';
    }
  });
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'hidden') { flushNote(true); savePad(true); return; }
    refreshShared();
  });
  window.addEventListener('focus', () => refreshShared());

  // Other people change shared things: pick up their changes when this tab
  // comes back into view, unless something here is being edited.
  let lastRefresh = Date.now();
  async function refreshShared() {
    if (!shell || !signedIn() || Date.now() - lastRefresh < 15000) return;
    if (S.editing || S.noteEdit || PS.dirty || $('.overlay') || (document.activeElement && /^(INPUT|TEXTAREA|SELECT)$/.test(document.activeElement.tagName))) return;
    lastRefresh = Date.now();
    try {
      await loadState();
      S.noteCache = {};
      renderAll();
    } catch (e) { /* offline: try again next time */ }
  }
  window.addEventListener('resize', debounce(() => { if (S.route.section === 'notes' && !S.noteEdit) renderMain(); }, 250));

  // ------------------------------------------------------------------
  // Boot
  // ------------------------------------------------------------------

  async function boot() {
    try {
      await loadSession();
      await loadState();
      // No public pages and not signed in: the sign-in screen is all there is.
      if (authOn() && !signedIn() && !S.pages.length) { applyAppearance(); renderLoginPage(); return; }
    } catch (e) {
      $('#app').replaceChildren(h('div', { class: 'boot' }, 'Could not load the dashboard: ' + e.message));
      return;
    }
    S.editing = false; S.draft = null; S.dirty = false; S.noteEdit = null;
    buildShell();
    armIdle();
    onRoute();
  }

  boot();
})();
