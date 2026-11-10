#!/usr/bin/env bash
# Kexra 更新脚本，支持 Docker 重建和本地重新编译。
# Kexra 启动时执行幂等的数据库结构迁移。
set -euo pipefail
cd "$(cd "$(dirname "$0")" && pwd)"

info(){ printf '\033[36m[*]\033[0m %s\n' "$*"; }
ok(){   printf '\033[32m[+]\033[0m %s\n' "$*"; }
warn(){ printf '\033[33m[!]\033[0m %s\n' "$*"; }
die(){  printf '\033[31m[x]\033[0m %s\n' "$*" >&2; exit 1; }
ask(){  local p="$1" d="${2:-}" a; read -rp "$p${d:+ [$d]}: " a; echo "${a:-$d}"; }

# 按提示同步仓库代码。
sync_repo(){
  [ -d .git ] && command -v git >/dev/null 2>&1 || { warn "当前目录不是 Git 工作副本或找不到 Git，跳过 git pull"; return; }
  [ "$(ask '是否获取最新代码（git pull --ff-only）？(y/n)' y)" = y ] || return
  if ! git pull --ff-only; then
    warn "git pull 无法快进，可能存在本地修改或分支分叉。本次继续使用当前代码"
  fi
}

# Docker 更新。
update_docker(){
  command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1 \
    || die "找不到 Docker 或 Docker Compose。请先运行 ./install.sh 部署"
  [ -f .env ] || die "找不到 .env。请先用 ./install.sh 完成首次部署"

  local image
  image="$(grep -m1 '^KEXRA_IMAGE=' .env | cut -d= -f2- || true)"
  image="${image:-kexra:local}"
  if [ "$image" = "kexra:local" ]; then
    docker image inspect "$image" >/dev/null 2>&1 ||
      die "本地 Kexra 镜像不存在。请先重新构建，不能从旧项目拉取二进制。"
  else
    docker compose pull kexra || die "拉取指定的 Kexra 镜像失败"
  fi
  info "正在重建并启动 Kexra，数据库结构会在启动时迁移…"
  docker compose up -d kexra
  ok "更新完成 → http://localhost:8787"
  info "查看日志：docker compose logs -f kexra"
  info "清理旧镜像（可选）：docker image prune -f"
}

# 本地编译更新。
update_local(){
  command -v go >/dev/null 2>&1 || die "找不到 Go（>=1.26）：https://go.dev/dl/"
  [ -f config.json ] || warn "找不到 config.json。首次部署请使用 ./install.sh"
  ok "Go: $(go version)"

  if command -v npm >/dev/null 2>&1; then
    info "正在重新构建前端静态文件…"
    ( cd web && npm ci && npm run build:static )
    rm -rf server/webui/dist && cp -r web/out server/webui/dist
    info "正在重新编译内嵌前端的单文件程序…"
    CGO_ENABLED=0 go build -tags embedui -trimpath -o kexra ./cmd/kexra
  else
    warn "找不到 npm，将仅编译后端。可用 npm run dev 单独启动前端"
    CGO_ENABLED=0 go build -o kexra ./cmd/kexra
  fi
  ok "编译完成 → ./kexra"
  warn "完成构建后请手动重启正在运行的 Kexra 进程，重启前备份数据库。"
}

echo "=============================="
echo "  Kexra 更新"
echo "  1) Docker 更新（使用指定镜像重建）"
echo "  2) 本地更新（Go 重新编译）"
echo "=============================="
case "$(ask '请选择' 1)" in
  1) sync_repo; update_docker ;;
  2) sync_repo; update_local ;;
  *) die "选项无效" ;;
esac
