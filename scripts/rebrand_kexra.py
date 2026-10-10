#!/usr/bin/env python3
"""Scoped, repeatable Kexra rebrand. Original LICENSE and web/LICENSE remain intact."""
from pathlib import Path
import re, shutil
root=Path(__file__).resolve().parents[1]
changed=[]
def upd(path, pairs=(), fn=None):
 p=root/path
 if not p.is_file(): raise FileNotFoundError(path)
 a=p.read_text("utf-8"); b=a
 for x,y in pairs:
  if x not in b: raise RuntimeError(path+" missing "+repr(x))
  b=b.replace(x,y)
 if fn: b=fn(b)
 if b!=a: p.write_text(b,"utf-8"); changed.append(path)
upd("go.mod",[("module github.com/Autumn-27/artex","module github.com/ForceMind/Kexra")])
for p in root.rglob("*.go"):
 if ".git" in p.parts: continue
 old=p.read_text("utf-8")
 new=old.replace("github.com/Autumn-27/artex/","github.com/ForceMind/Kexra/")
 if old!=new: p.write_text(new,"utf-8"); changed.append(str(p.relative_to(root)))
if (root/"cmd/artex").exists():
 shutil.move(str(root/"cmd/artex"),str(root/"cmd/kexra"));changed.append("cmd/artex -> cmd/kexra")
upd("cmd/kexra/main.go",fn=lambda x: re.sub(r"const banner = \x60[\s\S]*?\x60",
 'const banner = "\\n KEXRA  |  科巡\\n"', x.replace("ARTEX","KEXRA").replace("artex","kexra"),count=1))
for p in (root/"selfupdate").glob("*.go"):
 a=p.read_text("utf8")
 b=a.replace('const Repo = "Autumn-27/artex"','const Repo = "ForceMind/Kexra"').replace("artex","kexra")
 if a!=b: p.write_text(b,"utf8");changed.append(str(p.relative_to(root)))
upd("server/auth.go",[
 ('Subject:   "ARTEX",','Subject:   "Kexra",'),
 ('if req.Username != "ARTEX" {','if req.Username != "Kexra" && req.Username != "ARTEX" {'),
 ('if c, err := r.Cookie("artex_token"); err == nil && c.Value != "" {\n\t\treturn c.Value\n\t}',
  'for _, cookieName := range []string{"kexra_token", "artex_token"} {\n\t\tif c, err := r.Cookie(cookieName); err == nil && c.Value != "" {\n\t\t\treturn c.Value\n\t\t}\n\t}')
])
for f in ["agent/prompt.go","server/orchestration.go"]:upd(f,fn=lambda x:x.replace("ARTEX","Kexra"))
for f in ["build.sh","start.sh","start.bat","Dockerfile"]:upd(f,fn=lambda x:x.replace("artex","kexra"))
upd("dev.sh",[("./cmd/artex","./cmd/kexra")])
for f in ["install.sh","update.sh"]:
 upd(f,fn=lambda x:x.replace("./cmd/artex","./cmd/kexra").replace("./artex","./kexra")
    .replace("-o artex ","-o kexra ").replace("docker compose pull artex","docker compose pull kexra")
    .replace("docker compose up -d artex","docker compose up -d kexra")
    .replace("docker compose logs -f artex","docker compose logs -f kexra")
    .replace("autumn27/artex","kexra:local"))
upd("docker-compose.yml",fn=lambda x:
 x.replace("  artex:\n    image: autumn27/artex:"+chr(36)+"{ARTEX_TAG:-latest}",
  "  kexra:\n    # Build this local image before compose up; no official image is published.\n    image: kexra:local"))
upd("web/package.json",[('"name": "artex-web"','"name": "kexra-web"')])
upd("web/package-lock.json",[('"name": "artex-web"','"name": "kexra-web"')])
upd("web/src/config/app-config.ts",fn=lambda x:x.replace("ARTEX","Kexra"))
for f in ["web/messages/zh.json","web/messages/ko.json"]:
 upd(f,fn=lambda x:x.replace("ARTEX","Kexra"))
for f in ["web/src/app/(main)/_components/sidebar/app-sidebar.tsx",
          "web/src/app/(auth)/login/page.tsx","web/src/app/(auth)/setup/page.tsx"]:
 upd(f,[('src="/logo.png" alt="ARTEX"','src="/logo.svg" alt="Kexra"')])
upd("web/src/app/(auth)/login/page.tsx",[
 ('api.login("ARTEX", password)','api.login("Kexra", password)'),
 ('value="ARTEX"','value="Kexra"')])
upd("web/src/hooks/use-current-user.ts",[
 ('name: "ARTEX", username: "artex"','name: "Kexra", username: "kexra"')])
upd("web/src/lib/auth.ts",[
 ('const TOKEN_KEY = "artex_token";','const TOKEN_KEY = "kexra_token";\nconst LEGACY_TOKEN_KEY = "artex_token";'),
 ('localStorage.getItem(TOKEN_KEY) ?? (process.env',
  'localStorage.getItem(TOKEN_KEY) ?? localStorage.getItem(LEGACY_TOKEN_KEY) ?? (process.env'),
 ('localStorage.removeItem(TOKEN_KEY);',
  'localStorage.removeItem(TOKEN_KEY);\n    localStorage.removeItem(LEGACY_TOKEN_KEY);'),
 ('payload.sub ?? "ARTEX"','payload.sub ?? "Kexra"')])
upd("web/src/proxy.ts",[
 ('request.cookies.get("artex_token")?.value',
  '(request.cookies.get("kexra_token")?.value ?? request.cookies.get("artex_token")?.value)')])
upd("web/src/lib/api.ts",[
 ('return localStorage.getItem("artex_token");',
  'return localStorage.getItem("kexra_token") ?? localStorage.getItem("artex_token");'),
 ('localStorage.removeItem("artex_token");',
  'localStorage.removeItem("kexra_token");\n      localStorage.removeItem("artex_token");'),
 ('document.cookie = "artex_token=; path=/; max-age=0";',
  'document.cookie = "kexra_token=; path=/; max-age=0";\n      document.cookie = "artex_token=; path=/; max-age=0";')])
for f in ["web/src/app/(main)/system/logs/page.tsx","web/src/lib/mock/data.ts"]:
 upd(f,fn=lambda x:x.replace("ARTEX","Kexra"))
for f in ["web/src/app/(main)/system/notify/page.tsx",
          "web/src/app/(main)/system/notify/_components/channel-fields.ts"]:
 upd(f,fn=lambda x:x.replace("artex.example.com","kexra.example.com").replace("artex@example.com","kexra@example.com"))
logo="""<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128" role="img">
<title>Kexra 科巡</title><defs><linearGradient id="g" x2="1" y2="1"><stop stop-color="#4ADEB0"/><stop offset="1" stop-color="#65B7FF"/></linearGradient></defs>
<rect x="4" y="4" width="120" height="120" rx="27" fill="#0D1726"/>
<path d="M36 30v68" stroke="#E9FFF6" stroke-width="13" stroke-linecap="round"/>
<path d="M90 31 51 64 90 97" fill="none" stroke="url(#g)" stroke-width="14" stroke-linejoin="round" stroke-linecap="round"/>
<circle cx="94" cy="26" r="6" fill="#65B7FF"/></svg>
"""
for f in ["web/public/logo.svg","web/src/app/icon.svg"]:
 p=root/f;p.write_text(logo,encoding="utf8");changed.append(f)
for f in ["web/public/logo.png","web/src/app/icon.png","web/src/app/favicon.ico"]:
 p=root/f
 if p.exists():p.unlink();changed.append("removed "+f)
print("Kexra migration completed",len(changed),"files / paths")
for f in changed[:85]:print(" -",f)
