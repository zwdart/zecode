#!/usr/bin/env bash
# =========================================================
# ZeCode 构建/运行脚本
# 用法: ./start.sh <命令> [参数]
#
# 命令:
#   run          本地运行（嵌入模式，读取 config/config.toml）
#   dev          开发模式：强制静态资源从磁盘读，改前端无需重编译
#   build        构建当前平台二进制（不带版本号）
#   build-all    交叉编译全平台 + 打包（带版本号）+ 生成 checksums.txt
#   clean        清理 dist/ 与临时文件
#   tidy         整理 go.mod / go.sum
#   test         运行测试
#   fmt          格式化代码
#   help         显示帮助
#
# 环境变量:
#   ZECODE_CONFIG   指定配置文件路径，默认 config/config.toml
# =========================================================

set -euo pipefail

# ---------- 颜色 ----------
if [[ -t 1 ]]; then
  C_RED=$'\033[31m'; C_GREEN=$'\033[32m'; C_YELLOW=$'\033[33m'
  C_BLUE=$'\033[34m'; C_CYAN=$'\033[36m'; C_MAGENTA=$'\033[35m'; C_RESET=$'\033[0m'
else
  C_RED=""; C_GREEN=""; C_YELLOW=""; C_BLUE=""; C_CYAN=""; C_MAGENTA=""; C_RESET=""
fi

info()  { echo "${C_CYAN}[INFO]${C_RESET}  $*"; }
ok()    { echo "${C_GREEN}[OK]${C_RESET}    $*"; }
warn()  { echo "${C_YELLOW}[WARN]${C_RESET}  $*"; }
err()   { echo "${C_RED}[ERROR]${C_RESET} $*" >&2; }
step()  { echo "${C_MAGENTA}==>${C_RESET} $*"; }

# ---------- 项目信息 ----------
APP_NAME="zecode"
DIST_DIR="dist"
CONFIG_FILE="${ZECODE_CONFIG:-config/config.toml}"
STATIC_DIR="statics"
MAIN_PKG="."
LDFLAGS_BASE="-s -w"

# ---------- 前置检查 ----------
require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    err "未找到命令: $1，请先安装"
    exit 1
  fi
}

check_go() {
  require_cmd go
  local v
  v="$(go version | awk '{print $3}')"
  info "Go 版本: $v"
}

check_embed_src() {
  if [[ ! -d "$STATIC_DIR" ]]; then
    err "目录 $STATIC_DIR 不存在，无法嵌入静态资源"
    exit 1
  fi
  if [[ -z "$(ls -A "$STATIC_DIR" 2>/dev/null)" ]]; then
    err "目录 $STATIC_DIR 为空，embed 会编译失败，请放入 index.html 等文件"
    exit 1
  fi
}

# ---------- 版本信息 ----------
# 从 git describe 取，没有 git 或非仓库时退化为 dev
resolve_version() {
  local v
  if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    v="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
  else
    v="dev"
  fi
  echo "${v#v}"
}

build_time() {
  date -u +"%Y-%m-%dT%H:%M:%SZ"
}

build_ldflags() {
  local ver="$1" bt="$2"
  echo "${LDFLAGS_BASE} -X main.version=${ver} -X main.buildTime=${bt}"
}

# ---------- 命令 ----------

cmd_run() {
  check_go
  check_embed_src
  info "启动服务（嵌入模式，使用 $CONFIG_FILE）..."
  ZECODE_CONFIG="$CONFIG_FILE" go run "$MAIN_PKG"
}

cmd_dev() {
  check_go
  check_embed_src

  if [[ ! -f "$CONFIG_FILE" ]]; then
    warn "未找到 $CONFIG_FILE，将使用内置默认值"
    go run "$MAIN_PKG"
    return
  fi

  local tmp_cfg
  tmp_cfg="$(mktemp -t zecode-dev-XXXX.toml)"
  grep -v '^[[:space:]]*use_embedded_static' "$CONFIG_FILE" > "$tmp_cfg" || true
  echo "use_embedded_static = false" >> "$tmp_cfg"

  info "开发模式：静态资源从 $STATIC_DIR 磁盘读取（配置文件 $tmp_cfg）"
  ZECODE_CONFIG="$tmp_cfg" go run "$MAIN_PKG"

  rm -f "$tmp_cfg"
}

# 构建当前平台（单文件，不带版本号，方便本地跑）
cmd_build() {
  check_go
  check_embed_src
  mkdir -p "$DIST_DIR"

  local goos goarch ext out ver bt ldflags
  goos="$(go env GOOS)"
  goarch="$(go env GOARCH)"
  ext=""
  [[ "$goos" == "windows" ]] && ext=".exe"

  ver="$(resolve_version)"
  bt="$(build_time)"
  ldflags="$(build_ldflags "$ver" "$bt")"

  out="$DIST_DIR/${APP_NAME}${ext}"
  info "构建当前平台: ${goos}/${goarch}  版本=${ver}  ->  $out"
  go build -trimpath -ldflags "$ldflags" -o "$out" "$MAIN_PKG"
  ok "构建完成: $out"
  ls -lh "$out"
}

# 交叉编译 + 打包（带版本号）
cmd_build_all() {
  check_go
  check_embed_src
  mkdir -p "$DIST_DIR"

  local ver bt ldflags
  ver="$(resolve_version)"
  bt="$(build_time)"
  ldflags="$(build_ldflags "$ver" "$bt")"

  step "ZeCode 交叉编译"
  echo "  版本:     $ver"
  echo "  构建时间: $bt"
  echo "  产物目录: $DIST_DIR/"
  echo

  # 目标平台：GOOS/GOARCH
  local targets=(
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
    "windows/arm64"
    "darwin/amd64"
    "darwin/arm64"
  )

  # ---------- 1. 编译 ----------
  step "1/3 编译各平台二进制"
  local goos goarch ext bin
  for t in "${targets[@]}"; do
    goos="${t%%/*}"
    goarch="${t##*/}"
    ext=""
    [[ "$goos" == "windows" ]] && ext=".exe"
    bin="$DIST_DIR/${APP_NAME}-${goos}-${goarch}${ext}"

    info "构建 $goos/$goarch ..."
    GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
      go build -trimpath -ldflags "$ldflags" -o "$bin" "$MAIN_PKG"
    ok " -> $(basename "$bin") ($(du -h "$bin" | cut -f1))"
  done

  # ---------- 2. 打包（带版本号） ----------
  echo
  step "2/3 打包（文件名含版本号 v${ver}）"
  local pkg
  for t in "${targets[@]}"; do
    goos="${t%%/*}"
    goarch="${t##*/}"
    ext=""
    [[ "$goos" == "windows" ]] && ext=".exe"

    bin="${APP_NAME}-${goos}-${goarch}${ext}"
    pkg="${APP_NAME}-${ver}-${goos}-${goarch}"

    pushd "$DIST_DIR" >/dev/null

    # 附带一个简单的 README，用户解压后能看到运行说明
    cat > "${pkg}.README.txt" <<EOF
${APP_NAME} v${ver}
Platform : ${goos}/${goarch}
Built    : ${bt}

运行:
  1. 解压后放在任意目录
  2. 可选: 在同级创建 config/config.toml（不创建则使用内置默认值）
  3. 启动:
       Linux/macOS : chmod +x ${bin} && ./${bin}
       Windows     : ${bin}
  4. 浏览器访问 http://localhost:8080

环境变量:
  ZECODE_CONFIG   指定配置文件路径
EOF

    if [[ "$goos" == "windows" ]]; then
      if command -v zip >/dev/null 2>&1; then
        zip -q "${pkg}.zip" "$bin" "${pkg}.README.txt"
      elif command -v 7z >/dev/null 2>&1; then
        7z a -bso0 -bsp0 "${pkg}.zip" "$bin" "${pkg}.README.txt" >/dev/null
      else
        warn "未找到 zip/7z，跳过 ${pkg}.zip"
      fi
    else
      tar -czf "${pkg}.tar.gz" "$bin" "${pkg}.README.txt"
    fi

    # 清掉中间文件，dist 里只留压缩包
    rm -f "$bin" "${pkg}.README.txt"
    popd >/dev/null

    ok " -> ${pkg}.$([[ "$goos" == "windows" ]] && echo zip || echo tar.gz)"
  done

  # ---------- 3. checksums ----------
  echo
  step "3/3 生成 checksums.txt"
  pushd "$DIST_DIR" >/dev/null
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum ./*.tar.gz ./*.zip 2>/dev/null > checksums.txt || true
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 ./*.tar.gz ./*.zip 2>/dev/null > checksums.txt || true
  else
    warn "未找到 sha256sum/shasum，跳过校验和"
  fi
  [[ -s checksums.txt ]] && cat checksums.txt
  popd >/dev/null

  echo
  ok "全部完成，产物位于 $DIST_DIR/"
  ls -lh "$DIST_DIR"
}

cmd_clean() {
  info "清理 $DIST_DIR 与缓存文件..."
  rm -rf "$DIST_DIR"
  rm -f zecode zecode.exe
  ok "清理完成"
}

cmd_tidy() {
  check_go
  info "整理 go.mod / go.sum ..."
  go mod tidy
  ok "整理完成"
}

cmd_test() {
  check_go
  info "运行测试 ..."
  go test ./... -v
}

cmd_fmt() {
  check_go
  info "格式化代码 ..."
  go fmt ./...
  ok "格式化完成"
}

cmd_help() {
  cat <<EOF
${C_BLUE}ZeCode 构建脚本${C_RESET}

用法: ./start.sh <命令>

命令:
  ${C_GREEN}run${C_RESET}         本地运行（嵌入模式，读取 $CONFIG_FILE）
  ${C_GREEN}dev${C_RESET}         开发模式：强制 use_embedded_static=false，改前端无需重编译
  ${C_GREEN}build${C_RESET}       构建当前平台二进制到 $DIST_DIR/（不带版本号）
  ${C_GREEN}build-all${C_RESET}   交叉编译全平台 + 打包（文件名带版本号）+ 生成 checksums.txt
  ${C_GREEN}tidy${C_RESET}        整理 go.mod / go.sum
  ${C_GREEN}test${C_RESET}        运行测试
  ${C_GREEN}fmt${C_RESET}         格式化代码
  ${C_GREEN}clean${C_RESET}       清理构建产物
  ${C_GREEN}help${C_RESET}        显示本帮助

版本号来源:
  优先取 git describe（如 v1.0.0 → 1.0.0，v1.0.0-3-gabc → v1.0.0-3-gabc）
  无 git 环境时退化为 "dev"

打包产物示例（版本 = 1.0.0）:
  ${DIST_DIR}/zecode-1.0.0-linux-amd64.tar.gz
  ${DIST_DIR}/zecode-1.0.0-linux-arm64.tar.gz
  ${DIST_DIR}/zecode-1.0.0-windows-amd64.zip
  ${DIST_DIR}/zecode-1.0.0-windows-arm64.zip
  ${DIST_DIR}/zecode-1.0.0-darwin-amd64.tar.gz
  ${DIST_DIR}/zecode-1.0.0-darwin-arm64.tar.gz
  ${DIST_DIR}/checksums.txt

示例:
  ./start.sh run           # 本地跑一下
  ./start.sh dev           # 开发调试（前端从磁盘读）
  ./start.sh build         # 本地快速编译
  ./start.sh build-all     # 一次性出全平台包（带版本号）

注意:
  - embed 要求 $STATIC_DIR/ 目录存在且非空
  - 如需指定外部配置: ZECODE_CONFIG=/etc/zecode/config.toml ./start.sh run
  - 版本号取自当前 git tag；打 tag 后重新执行可让产物带正式版本
EOF
}

# ---------- 入口 ----------
main() {
  local cmd="${1:-help}"
  shift || true

  case "$cmd" in
    run)        cmd_run "$@" ;;
    dev)        cmd_dev "$@" ;;
    build)      cmd_build "$@" ;;
    build-all)  cmd_build_all "$@" ;;
    clean)      cmd_clean "$@" ;;
    tidy)       cmd_tidy "$@" ;;
    test)       cmd_test "$@" ;;
    fmt)        cmd_fmt "$@" ;;
    help|-h|--help) cmd_help ;;
    *)
      err "未知命令: $cmd"
      echo
      cmd_help
      exit 1
      ;;
  esac
}

main "$@"