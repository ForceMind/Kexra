#!/usr/bin/env bash
# Kexra 安装脚本，支持 Docker 部署和本地编译运行。
set -euo pipefail
cd "$(cd "$(dirname "$0")" && pwd)"

info(){ printf '\033[36m[*]\033[0m %s\n' "$*"; }
ok(){   printf '\033[32m[+]\033[0m %s\n' "$*"; }
warn(){ printf '\033[33m[!]\033[0m %s\n' "$*"; }
die(){  printf '\033[31m[x]\033[0m %s\n' "$*" >&2; exit 1; }
ask(){  local p="$1" d="${2:-}" a; read -rp "$p${d:+ [$d]}: " a; echo "${a:-$d}"; }
rand(){ head -c 18 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 24; }

# 检测 Docker 环境并按提示安装。
ensure_docker(){
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    ok "已检测到 Docker 和 Docker Compose"; return
  fi
  warn "找不到 Docker 或 Docker Compose"
  case "$(uname -s)" in
    Linux)
      if [ "$(ask '是否自动安装 Docker？(y/n)' y)" = y ]; then
        curl -fsSL https://get.docker.com | sh
        sudo usermod -aG docker "$USER" || true
        ok "Docker 安装完成。重新登录后，用户组权限才会生效"
      else
        die "请手动安装 Docker 后重新运行"
      fi ;;
    Darwin) die "请在 macOS 上安装 Docker Desktop：https://www.docker.com/products/docker-desktop/" ;;
    *)      die "请手动安装 Docker 后重新运行" ;;
  esac
}

# Docker 部署。
install_docker(){
  ensure_docker
  if [ ! -f .env ]; then
    cp .env.example .env 2>/dev/null || true
    local pw key
    pw="$(ask 'PostgreSQL 密码（回车随机生成）' "$(rand)")"
    key="$(ask 'ANTHROPIC_API_KEY（可留空，稍后在界面设置）' '')"
    sed -i.bak "s|^POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=${pw}|" .env
    sed -i.bak "s|^ANTHROPIC_API_KEY=.*|ANTHROPIC_API_KEY=${key}|" .env
    rm -f .env.bak
    ok "已生成 .env 并设置 POSTGRES_PASSWORD"
  else
    info "使用已有的 .env 文件"
  fi
  local image
  image="$(grep -m1 '^KEXRA_IMAGE=' .env | cut -d= -f2- || true)"
  image="${image:-kexra:local}"
  if ! docker image inspect "$image" >/dev/null 2>&1; then
    die "尚未找到本地镜像 $image。请先构建 Kexra 二进制并根据 Dockerfile 构建镜像；不会自动拉取原项目镜像。"
  fi
  info "启动 Kexra 容器（镜像：$image）"
  docker compose up -d || die "Docker Compose 启动失败，请检查容器日志"
  ok "启动完成 → http://localhost:8787"
  info "查看日志：docker compose logs -f kexra"
}

# 本地编译运行。
install_local(){
  echo "选择数据库部署方式"
  echo "  1) 连接已有 PostgreSQL"
  echo "  2) 用 Docker 启动 PostgreSQL（需要 Docker）"
  case "$(ask '请选择' 1)" in
    2)
      ensure_docker
      local pw; pw="$(ask 'PostgreSQL 密码（回车随机生成）' "$(rand)")"
      docker run -d --name artex-pg -p 5432:5432 \
        -e POSTGRES_USER=artex -e POSTGRES_PASSWORD="$pw" -e POSTGRES_DB=artex \
        -v artex-pg:/var/lib/postgresql/data postgres:16-alpine
      DB_HOST=127.0.0.1 DB_PORT=5432 DB_USER=artex DB_PASS="$pw" DB_NAME=artex DB_SSL=disable ;;
    *)
      DB_HOST="$(ask '数据库地址' 127.0.0.1)"
      DB_PORT="$(ask '端口' 5432)"
      DB_USER="$(ask '账号' artex)"
      DB_PASS="$(ask '密码' '')"
      DB_NAME="$(ask '数据库名称' artex)"
      DB_SSL="$(ask 'sslmode (disable/require)' disable)" ;;
  esac

  # 生成 config.json。
  cat > config.json <<JSON
{
  "database": {
    "host": "${DB_HOST}",
    "port": ${DB_PORT},
    "user": "${DB_USER}",
    "password": "${DB_PASS}",
    "dbname": "${DB_NAME}",
    "sslmode": "${DB_SSL}"
  }
}
JSON
  ok "已生成 config.json"

  # 检查 Go 环境。
  command -v go >/dev/null 2>&1 || die "找不到 Go。请先安装 Go（>=1.26）：https://go.dev/dl/"
  ok "Go: $(go version)"

  # 生成嵌入二进制所需的前端静态文件。
  if command -v npm >/dev/null 2>&1; then
    info "正在构建前端静态文件…"
    ( cd web && npm ci && npm run build:static )
    rm -rf server/webui/dist && mkdir -p server/webui && cp -r web/out server/webui/dist
    info "正在编译内嵌前端的单文件程序…"
    CGO_ENABLED=0 go build -tags embedui -trimpath -o kexra ./cmd/kexra
  else
    warn "找不到 npm，将仅编译后端。可用 npm run dev 单独启动前端"
    CGO_ENABLED=0 go build -o kexra ./cmd/kexra
  fi
  ok "编译完成 → ./kexra"

  info "正在启动…（按 Ctrl+C 退出）"
  ./kexra
}

echo "=============================="
echo "  Kexra 安装"
echo "  1) Docker 部署"
echo "  2) 本地运行（Go 编译）"
echo "=============================="
case "$(ask '请选择' 1)" in
  1) install_docker ;;
  2) install_local ;;
  *) die "选项无效" ;;
esac
