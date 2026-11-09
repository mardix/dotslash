#!/usr/bin/env python3
"""Load the demo data into a fresh DotSlash: admin's things, a second person
(Ana) with her own and shared things, and a few dashboard settings.

  DS_PASSWORD=secret123 DS_PORT=8098 DS_DATA_DIR=$(mktemp -d) ./dotslash &
  python3 scripts/dev/seed-demo.py            # DS_URL defaults to http://127.0.0.1:8098
"""
import http.cookiejar, json, os, urllib.request

BASE = os.environ.get("DS_URL", "http://127.0.0.1:8098")
DOCS = os.path.join(os.path.dirname(__file__), "..", "..", "docs")


def client():
    jar = http.cookiejar.CookieJar()
    op = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))

    def call(method, path, body=None, raw=False):
        data = None if body is None else (body if raw else json.dumps(body).encode())
        req = urllib.request.Request(BASE + path, data=data, method=method,
                                     headers={"X-Requested-With": "DotSlash", "Content-Type": "application/json"})
        with op.open(req) as r:
            txt = r.read()
            return json.loads(txt) if txt else None
    return call


admin = client()
admin("POST", "/api/login", {"username": "admin", "password": "secret123"})
admin("PUT", "/api/me", {"name": "Mardix"})
admin("POST", "/api/import", open(os.path.join(DOCS, "demo.json"), "rb").read(), raw=True)
admin("POST", "/api/users", {"username": "ana", "name": "Ana", "password": "password1", "role": "member"})
admin("POST", "/api/users", {"username": "sam", "name": "Sam", "password": "password1", "role": "member"})
ana = client()
ana("POST", "/api/login", {"username": "ana", "password": "password1"})
ana("POST", "/api/import", open(os.path.join(DOCS, "demo-ana.json"), "rb").read(), raw=True)
st = admin("GET", "/api/state")["settings"]
st["members_edit_shared"] = False
admin("PUT", "/api/settings", st)
print("demo loaded: admin/secret123 (Mardix), ana/password1, sam/password1")
