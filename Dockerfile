# syntax=docker/dockerfile:1
#
# 실행 전용 이미지(이미지 안에서 컴파일하지 않음): 상용 도구만 설치하고, **미리 컴파일한 Linux 단일 바이너리**를 넣는다.
# 바이너리는 CI 의 binaries job 이 크로스 컴파일하며(순수 Go, QEMU 없음), 목표 아키텍처별로
# 빌드 컨텍스트의 dist/<TARGETARCH>/kexra 에 둔다. 이렇게 하면 다중 아키텍처 빌드에서 arm64 는 apt 계층만
# 에뮬레이션하면 되고, Next/Go 컴파일은 더 이상 에뮬레이션하지 않아 훨씬 빠르다.
#
# 로컬에서 이미지를 수동으로 빌드할 때는 먼저 바이너리를 직접 준비한다:
#   cd web && npm run build:static && cd ..
#   mkdir -p server/webui && cp -r web/out server/webui/dist
#   CGO_ENABLED=0 GOARCH=amd64 go build -tags embedui -o dist/amd64/kexra ./cmd/kexra
#   docker build -t kexra:local .
FROM node:22.23.2-bookworm-slim AS node-runtime

FROM python:3.12-slim-bookworm
ARG TARGETARCH

# 使用明确版本的官方 Node 22 运行时，代替构建期执行远程 NodeSource 安装脚本。
# Docker 会根据构建的目标平台选择相同架构的 Node 层。
COPY --from=node-runtime /usr/local/bin/node /usr/local/bin/node
COPY --from=node-runtime /usr/local/lib/node_modules/npm /usr/local/lib/node_modules/npm
RUN ln -s /usr/local/lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm \
    && ln -s /usr/local/lib/node_modules/npm/bin/npx-cli.js /usr/local/bin/npx

# 随镜像安装授权实验室中可能用到的基础网络诊断工具。
# Node 已从上方精确版本的官方镜像复制，不执行远程 setup 脚本。
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates ripgrep curl wget vim git jq unzip \
      dnsutils iputils-ping netcat-openbsd inetutils-telnet whois nmap \
      libstdc++6 libgcc-s1 \
    && rm -rf /var/lib/apt/lists/* \
    && node --version && npm --version
# 使用独立锁文件同时固定 Playwright 工具及其传递依赖。
# Debian apt、浏览器和基础镜像仍需独立的 SBOM/漏洞审计。
# 浏览器由固定 playwright 包提供，在镜像构建时提前安装，运行时不下载。
COPY docker/browser-tools/package.json docker/browser-tools/package-lock.json /opt/kexra-browser-tools/
ENV KEXRA_BROWSER_TOOLS_DIR=/opt/kexra-browser-tools
ENV PATH="/opt/kexra-browser-tools/node_modules/.bin:${PATH}"
RUN npm ci --prefix /opt/kexra-browser-tools --omit=dev --ignore-scripts \
    && playwright-cli --help \
    && playwright-mcp --help \
    && playwright --version \
    && playwright install --with-deps chromium \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
# Separate, non-browsable JWT key store; docker-compose attaches a named volume.
RUN mkdir -p /app/private && chmod 0700 /app/private
ENV KEXRA_JWT_KEY_DIR=/app/private
# 미리 컴파일한 해당 아키텍처 바이너리(dist/amd64/kexra 또는 dist/arm64/kexra)
COPY dist/${TARGETARCH}/kexra /app/kexra
# 감시 기동 스크립트: 프로세스 종료 후 종료 코드에 따라 다시 띄울지 결정하며, 화면의 원클릭 업데이트가 이 스크립트로 교체를 완료한다.
# 이 스크립트는 동시에 SIGTERM 을 kexra 로 전달하는 일도 맡는다: docker stop 은 신호를 PID 1 에만 보내므로,
# 전달하지 않으면 kexra 가 신호를 받지 못해 우아한 종료를 못 하고 10 초 뒤 SIGKILL 로 강제 종료된다.
COPY start.sh /app/start.sh
RUN chmod +x /app/kexra /app/start.sh
COPY skills/ /app/skills/
# 数据与 JWT 私有密钥分别挂载，避免暴露在可浏览的工作目录中。
VOLUME ["/app/data", "/app/private"]
EXPOSE 8787 8788
ENTRYPOINT ["/app/start.sh"]
CMD ["-addr", ":8787", "-proxy", ":8788"]
