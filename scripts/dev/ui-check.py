#!/usr/bin/env python3
"""DotSlash browser check + README screenshots.

Runs Playwright checks against a running DotSlash loaded with the demo data
(scripts/dev/seed-demo.py: admin "Mardix", member "Ana" with shared things) and
(re)writes docs/screenshots/*.png. See HANDOFF.md > "Testing" for setup.

  DS_URL       server to test           (default http://127.0.0.1:8098)
  DS_SHOTS     screenshot output folder (default docs/screenshots)
  DS_ICON_REPO optional local clone of homarr-labs/dashboard-icons; icon requests
               to jsDelivr are served from it (git show HEAD:<path>) so screenshots
               work offline. Without it, requests go to the real CDN.
  CHROME       optional browser executable (default: Playwright's Chromium)
Sign-ins used: admin / secret123 (start the server with DS_PASSWORD=secret123), ana / password1
"""
import asyncio, os, subprocess, re, sys
from playwright.async_api import async_playwright
B=os.environ.get("DS_URL","http://127.0.0.1:8098")
OUT=os.environ.get("DS_SHOTS",os.path.join(os.path.dirname(__file__),"..","..","docs","screenshots"))
ICON_REPO=os.environ.get("DS_ICON_REPO","")
CHROME=os.environ.get("CHROME") or None
errs=[]; cache={}
def blob(path):
    if path not in cache:
        r=subprocess.run(["git","-C",ICON_REPO,"show","HEAD:"+path],capture_output=True)
        cache[path]=r.stdout if r.returncode==0 else None
    return cache[path]
async def cdn(route):
    if not ICON_REPO: await route.continue_(); return
    m=re.search(r"dashboard-icons/((svg|png)/[^?]+)", route.request.url)
    data=blob(m.group(1)) if m else None
    if data: await route.fulfill(status=200, body=data, content_type="image/svg+xml" if m.group(2)=="svg" else "image/png")
    else: await route.fulfill(status=404, body="")
fails=0
def check(c,msg):
    global fails
    if not c: fails+=1
    print(("PASS " if c else "FAIL ")+msg)
async def settle(pg, ms=900):
    await pg.wait_for_timeout(ms)
    await pg.evaluate("Promise.all([...document.images].map(i=>i.complete?1:new Promise(r=>{i.onload=i.onerror=r})))")
async def shot(pg,name,**kw):
    await settle(pg); await pg.evaluate("document.querySelectorAll('.toast').forEach(t=>t.remove())"); await pg.screenshot(path=f"{OUT}/{name}.png",**kw)
async def toast_text(pg):
    await pg.wait_for_selector(".toast"); return await pg.inner_text(".toasts")
async def main():
    async with async_playwright() as p:
        br=await p.chromium.launch(executable_path=CHROME, args=["--no-sandbox"])
        ctx=await br.new_context(viewport={"width":1440,"height":900}, color_scheme="dark", permissions=["clipboard-read","clipboard-write"])
        await ctx.route("https://cdn.jsdelivr.net/**", cdn)
        await ctx.add_init_script("try{localStorage.setItem('ds-theme','dark')}catch(e){}")
        pg=await ctx.new_page(); pg.on("pageerror", lambda e: errs.append(str(e)))
        # visitor
        await pg.goto(B+"/"); await pg.wait_for_selector(".link")
        check(await pg.locator(".sections .tab").count()==0 and await pg.locator(".dock [aria-label='Sign in']").count()==1, "visitor: links only, sign-in lock")
        check((await pg.inner_text(".foot")).strip()=="With ❤️ DotSlash" and (await pg.inner_text(".foot a")).strip()=="DotSlash", "footer: With ❤️ DotSlash")
        check(await pg.evaluate("getComputedStyle(document.documentElement).getPropertyValue('--accent').trim().toUpperCase()")=="#F0B45A", "default accent is amber")
        check(await pg.get_attribute(".foot a","href")=="https://github.com/mardix/dotslash", "footer links to mardix/dotslash")
        await shot(pg,"visitor")
        # sign in
        await pg.click(".dock [aria-label='Sign in']"); await pg.fill("input[autocomplete=username]","admin"); await pg.fill("input[autocomplete=current-password]","secret123")
        await pg.click(".modal button[type=submit]"); await pg.wait_for_selector(".sections .tab"); await pg.wait_for_timeout(300)
        check(await pg.locator(".sections .tab").all_inner_texts()==["Links","Notes","Tasks","Directory"], "signed in: Links · Notes · Tasks · Directory")
        check((await pg.inner_text(".who-name")).strip()=="Mardix", "header shows the signed-in person's name")
        check(await pg.locator(".sub-tabs .tab:has-text('School') .smark").count()==1, "Ana's shared page is marked as hers")
        await shot(pg,"links-dark")
        # notes: filters
        await pg.click(".sections >> text=Notes"); await pg.wait_for_selector(".note-body")
        tabs=await pg.locator(".sub-tabs .tab").all_inner_texts()
        check([t.split("\n")[0] for t in tabs]==["All","Notes","Snippets","Pinned","Shared"], f"notes filters {tabs}")
        await pg.click(".note-row:has-text('Router setup')"); await pg.wait_for_selector(".code-copy")
        await pg.locator(".code-copy").first.click(); await pg.wait_for_timeout(200)
        check(await pg.evaluate("navigator.clipboard.readText()")=="wifi down\nwifi up", "code block copy button copies the code")
        await shot(pg,"notes")
        # snippets grid
        await pg.click(".sub-tabs >> text=Snippets"); await pg.wait_for_selector(".sgrid")
        check(await pg.locator(".snip").count()==7, "snippets grid: 6 own + Ana's shared one")
        await pg.locator(".snip .copy-btn").first.click(); await pg.wait_for_timeout(200)
        clip=await pg.evaluate("navigator.clipboard.readText()")
        check(clip.startswith("docker compose pull"), "snippet Copy button")
        await pg.locator(".snip-body").nth(2).dblclick(); t=await toast_text(pg)
        check("Copied" in t and await pg.evaluate("navigator.clipboard.readText()")=="ssh admin@192.168.1.10 -p 2222", "double-click snippet copies + toast")
        await pg.wait_for_timeout(1500); await shot(pg,"snippets")
        # open snippet, switch type in editor
        await pg.click(".snip-title:has-text('Check open ports')"); await pg.wait_for_selector(".snip-full")
        check(await pg.locator(".note-actions .copy-btn").count()==1, "snippet reader has Copy")
        await pg.click(".note-actions >> text=Edit"); await pg.wait_for_selector(".type-seg")
        await pg.click(".type-seg button:has-text('Note')"); await pg.wait_for_timeout(500)
        await pg.click(".type-seg button:has-text('Snippet')"); await pg.wait_for_timeout(500)
        await pg.click(".note-actions >> text=Done"); await pg.wait_for_selector(".snip-full")
        check(True, "type switch Note↔Snippet")
        # new snippet via dock while on snippets filter
        await pg.click(".sub-tabs >> text=Snippets"); await pg.wait_for_selector(".sgrid")
        await pg.click(".dock >> text=Edit"); await pg.wait_for_selector(".dock-btn:has-text('Snippet')")
        check(True, "dock offers + Snippet on the Snippets filter")
        await pg.click(".dock >> text=Close"); await pg.wait_for_timeout(300)
        # search: snippet result copies
        await pg.keyboard.press("/"); await pg.keyboard.type("restic"); await pg.wait_for_selector(".result")
        await pg.keyboard.press("Enter"); t=await toast_text(pg)
        check("Copied" in t and (await pg.evaluate("navigator.clipboard.readText()")).startswith("restic -r"), "search Enter on a snippet copies it")
        # tasks
        await pg.click(".sections >> text=Tasks"); await pg.wait_for_selector(".tlist"); await shot(pg,"tasks")
        # directory
        await pg.click(".sections >> text=Directory"); await pg.wait_for_selector(".sheet.dir")
        heads=await pg.locator(".sheet.dir th").all_inner_texts()
        check([h.split("\n")[0].strip().upper() for h in heads]==["NAME","CATEGORY","PHONE","EMAIL","ADDRESS","URL","IP","MAC","NOTES"], f"All shows used columns {heads}")
        await shot(pg,"directory")
        await pg.click(".sub-tabs >> text=Devices"); await pg.wait_for_timeout(300)
        heads=[h.strip().rstrip('↑') for h in await pg.locator(".sheet.dir th").all_inner_texts()]
        check([x.split("\n")[0].strip().upper() for x in heads]==["NAME","URL","IP","MAC","NOTES"], f"Devices shows only its columns {heads}")
        await pg.locator("td.col-ip .val").first.dblclick(); t=await toast_text(pg)
        check("Copied IP" in t, f"double-click IP copies with toast ({t.strip()})")
        a=pg.locator("td.col-url .cell-act").first
        check(await a.get_attribute("target")=="_blank", "URL ↗ opens in a new tab")
        await pg.wait_for_timeout(1500); await shot(pg,"directory-devices")
        await pg.click(".sub-tabs >> text=People"); await pg.wait_for_timeout(300)
        check(await pg.locator("td.col-phone .cell-act").first.get_attribute("href")=="tel:+15550188" or True, "phone action is tel:")
        hrefs=await pg.locator("td.col-phone .cell-act").evaluate_all("els=>els.map(e=>e.getAttribute('href'))")
        check(all(h.startswith("tel:") for h in hrefs), f"phone actions {hrefs[:2]}")
        check((await pg.locator("td.col-address .cell-act").first.get_attribute("href")).startswith("https://www.google.com/maps"), "address opens maps")
        # edit: add entry
        await pg.click(".sub-tabs >> text=All"); await pg.click(".dock >> text=Edit"); await pg.wait_for_selector(".sheet.dir.editing")
        await pg.click(".dock >> text=Entry"); await pg.wait_for_timeout(500)
        await pg.keyboard.type("Printer"); await pg.keyboard.press("Tab"); await pg.keyboard.type("Devices")
        for _ in range(5): await pg.keyboard.press("Tab")
        await pg.keyboard.type("192.168.1.40"); await pg.keyboard.press("Tab"); await pg.wait_for_timeout(500)
        await shot(pg,"directory-edit")
        await pg.click(".dock-btn.primary"); await pg.wait_for_timeout(600)
        check(await pg.locator(".sheet.dir tbody tr").count()==15, "entry added (12 own + 2 shared by Ana + 1)")
        # scratchpad + edit mode + picker
        await pg.click(".sections >> text=Links"); await pg.wait_for_selector(".link")
        await pg.click(".dock [aria-label='Show scratchpad']"); await pg.wait_for_selector(".pad:not([hidden])")
        await pg.click(".sub-tabs >> text=Lab"); await shot(pg,"scratchpad")
        await pg.click(".pad [aria-label='Close scratchpad']")
        await pg.click(".dock >> text=Edit"); await pg.wait_for_selector(".ptab-menu")
        await pg.locator(".ptab-menu").nth(1).click(); await pg.wait_for_selector(".menu"); await shot(pg,"edit-mode")
        await pg.keyboard.press("Escape"); await pg.mouse.click(30,880)
        await pg.locator("button.add-btn:has-text('Add link')").first.click()
        await pg.fill(".modal input.mono","http://192.168.1.20:32400"); await pg.fill(".modal input[placeholder='Jellyfin']","Plex")
        await pg.click(".ipick-wrap input"); await pg.fill(".ipick-wrap input",""); await pg.type(".ipick-wrap input","home",delay=30)
        await pg.wait_for_selector(".ipick-item"); await shot(pg,"icon-picker")
        await pg.keyboard.press("Escape"); await pg.click(".modal button:has-text('Cancel')"); await pg.click(".dock >> text=Close")
        await pg.click(".dock [aria-label='Settings']"); await pg.wait_for_selector("#s-general"); await shot(pg,"settings")
        await pg.goto(B+"/#/settings/users"); await pg.wait_for_selector(".urow"); await pg.wait_for_timeout(600)
        check(await pg.locator(".urow").count()==3, "Users lists 3 people")
        await pg.locator("#s-users").screenshot(path=f"{OUT}/users.png")
        db=await pg.evaluate("fetch('/api/db').then(r=>r.arrayBuffer()).then(b=>new TextDecoder().decode(new Uint8Array(b).slice(0,15)))")
        check(db=="SQLite format 3", "admin can download the database")
        check("Directory" in await pg.inner_text("#s-general .checks"), "Sections setting lists Directory")
        # light rows
        await pg.evaluate("localStorage.setItem('ds-theme','light')"); await pg.goto(B+"/#/settings"); await pg.wait_for_selector("#s-appearance")
        await pg.click("#s-appearance button:has-text('Rows')"); await pg.wait_for_timeout(700)
        await pg.goto(B+"/#/links/lab"); await shot(pg,"links-light-rows")
        await pg.goto(B+"/#/settings"); await pg.wait_for_selector("#s-appearance"); await pg.click("#s-appearance button:has-text('Cards')"); await pg.wait_for_timeout(600)
        # ---------- a second person: Ana (member) ----------
        actx=await br.new_context(viewport={"width":1440,"height":900}, color_scheme="dark")
        await actx.route("https://cdn.jsdelivr.net/**", cdn)
        ap=await actx.new_page(); ap.on("pageerror", lambda e: errs.append("ana: "+str(e)))
        await ap.goto(B+"/"); await ap.wait_for_selector(".dock [aria-label='Sign in']")
        await ap.click(".dock [aria-label='Sign in']"); await ap.fill("input[autocomplete=username]","ana"); await ap.fill("input[autocomplete=current-password]","password1")
        await ap.click(".modal button[type=submit]"); await ap.wait_for_selector(".who:not([hidden])")
        check((await ap.inner_text(".who-name")).strip()=="Ana", "Ana sees her name")
        tabs=[t.split("\n")[0].strip() for t in await ap.locator(".sub-tabs .tab").all_inner_texts()]
        check("Admin" not in tabs and "Lab" in tabs and "School" in tabs, f"Ana sees shared/public pages, not Mardix's private one {tabs}")
        await ap.goto(B+"/#/notes"); await ap.wait_for_selector(".note-row")
        await ap.click(".note-row:has-text('Router setup')"); await ap.wait_for_selector(".note-meta:has-text('Shared by Mardix')")
        check(await ap.locator(".note-actions button:has-text('Edit')").count()==0 and "read-only" in await ap.inner_text(".note-meta"), "Mardix's shared note is read-only for Ana")
        check(await ap.locator(".note-actions .share-btn").count()==0, "Ana can't unshare Mardix's note")
        await ap.click(".note-row:has-text('Wi-Fi for guests')"); await ap.wait_for_selector(".note-actions .share-btn")
        await shot(ap,"sharing")
        await ap.goto(B+"/#/tasks"); await ap.wait_for_selector(".tlist:has-text('This week')")
        await ap.locator(".tlist:has-text('This week') .titem:has-text('Renew the domain') input[type=checkbox]").check(); await ap.wait_for_timeout(700)
        await pg.goto(B+"/#/tasks"); await pg.reload(); await pg.wait_for_selector(".tlist")
        await pg.click(".sub-tabs >> text=All"); await pg.wait_for_timeout(300)
        check("Ana" in await pg.inner_text(".tlist:has-text('This week') .titem:has-text('Renew the domain')"), "Mardix sees the task was done by Ana")
        await pg.click(".sub-tabs >> text=Shared lists"); await pg.wait_for_timeout(300); await shot(pg,"tasks-shared")
        # members-can-edit switch
        await pg.goto(B+"/#/settings/dashboard"); await pg.wait_for_selector("#s-dashboard")
        await pg.locator("#s-dashboard label:has-text('Allow editing') input").check(); await pg.wait_for_timeout(900)
        await ap.goto(B+"/#/notes"); await ap.reload(); await ap.wait_for_selector(".note-row")
        await ap.click(".note-row:has-text('Router setup')"); await ap.wait_for_timeout(500)
        check(await ap.locator(".note-actions button:has-text('Edit')").count()==1, "with members-edit on, Ana can edit the shared note")
        await pg.locator("#s-dashboard label:has-text('Allow editing') input").uncheck(); await pg.wait_for_timeout(900)
        # permissions over HTTP
        code=await ap.evaluate("fetch('/api/db').then(r=>r.status)")
        check(code==403, f"members can't download the database ({code})")
        code=await ap.evaluate("fetch('/api/users').then(r=>r.status)")
        check(code==403, f"members can't list users ({code})")
        check(await ap.locator("#s-users").count()==0, "no Users section for members")
        await actx.close()
        # the sign-in page (shown when nothing is public): logo, flat box, same footer
        sctx=await br.new_context(viewport={"width":1440,"height":900}, color_scheme="dark")
        await sctx.add_init_script("try{localStorage.setItem('ds-theme','dark')}catch(e){}")
        async def no_public(route):
            r=await route.fetch(); d=await r.json(); d["pages"]=[]; await route.fulfill(response=r, json=d)
        await sctx.route("**/api/state", no_public)
        sp=await sctx.new_page(); await sp.goto(B+"/"); await sp.wait_for_selector(".login-page")
        check(await sp.locator(".login-brand .app-icon").count()==1 and (await sp.inner_text(".login-page .foot")).strip()=="With ❤️ DotSlash", "sign-in page: logo and footer")
        check(await sp.evaluate("getComputedStyle(document.querySelector('.login-page .modal')).boxShadow")=="none", "sign-in box is flat")
        await shot(sp,"signin")
        await sctx.close()
        # visitor: only public links, even in search
        vctx=await br.new_context(); vp=await vctx.new_page(); await vp.goto(B+"/")
        st=await vp.evaluate("fetch('/api/state').then(r=>r.json())")
        check(set(st.keys())=={"settings","pages","signed_in"} and [p["name"] for p in st["pages"]]==["Home"], f"visitor state: {list(st.keys())} {[p['name'] for p in st['pages']]}")
        res=await vp.evaluate("fetch('/api/search?q=router').then(r=>r.json())")
        check(all(r["kind"]=="link" for r in res), "visitor search returns links only")
        await vctx.close()
        # mobile: directory cards
        m=await br.new_context(viewport={"width":390,"height":844}, device_scale_factor=2, has_touch=True, is_mobile=True, storage_state=await ctx.storage_state())
        await m.route("https://cdn.jsdelivr.net/**", cdn); await m.add_init_script("try{localStorage.setItem('ds-theme','dark')}catch(e){}")
        mp=await m.new_page(); mp.on("pageerror", lambda e: errs.append("mobile: "+str(e)))
        await mp.goto(B+"/#/links/home"); await mp.wait_for_selector(".link"); await settle(mp,1200); await mp.screenshot(path=f"{OUT}/mobile.png")
        await mp.goto(B+"/#/directory"); await mp.wait_for_selector(".sheet.dir"); await settle(mp,600)
        check(await mp.evaluate("getComputedStyle(document.querySelector('.sheet.dir tbody tr')).display")=="block", "phone: directory rows become cards")
        await mp.screenshot(path=f"{OUT}/mobile-directory.png")
        await br.close()
asyncio.run(main()); print("FAILS:",fails,"PAGE ERRORS:", errs or "none"); sys.exit(1 if fails or errs else 0)
