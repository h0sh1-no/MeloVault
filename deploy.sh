#!/bin/bash
# MeloVault 一键部署脚本
# 适用于 Ubuntu / Debian 系统
# 用法: bash deploy.sh [deploy|upgrade|uninstall|status]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="${MELOVAULT_DIR:-/opt/melovault}"
IMAGE="${MELOVAULT_IMAGE:-melovault:latest}"
PROJECT_NAME="melovault"

# ── 颜色 ────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

# ── 日志函数 ────────────────────────────────────────────────────────
log_info()   { echo -e "${GREEN}[✓]${NC} $1"; }
log_warn()   { echo -e "${YELLOW}[!]${NC} $1"; }
log_error()  { echo -e "${RED}[✗]${NC} $1"; }
log_prompt() { echo -e "${BLUE}[?]${NC} $1"; }
log_step()   { echo -e "${CYAN}[→]${NC} ${BOLD}$1${NC}"; }

banner() {
    echo -e "${CYAN}"
    echo '  __  __      _    __     __         _ _   '
    echo ' |  \/  | ___| | ___\ \   / /_ _ _   _| | |_ '
    echo ' | |\/| |/ _ \ |/ _ \\ \ / / _` | | | | | __|'
    echo ' | |  | |  __/ | (_) |\ V / (_| | |_| | | |_ '
    echo ' |_|  |_|\___|_|\___/  \_/ \__,_|\__,_|_|\__|'
    echo -e "${NC}"
    echo -e " ${DIM}Music streaming · Self-hosted · Docker${NC}"
    echo ""
}

# ── 工具函数 ────────────────────────────────────────────────────────
check_root() {
    if [ "$(id -u)" -ne 0 ]; then
        log_error "请以 root 权限运行此脚本"
        echo "  sudo bash deploy.sh"
        exit 1
    fi
}

check_command() {
    command -v "$1" &>/dev/null
}

generate_secret() {
    openssl rand -hex 32 2>/dev/null || head -c 64 /dev/urandom | od -An -tx1 | tr -d ' \n' | head -c 64
}

generate_password() {
    openssl rand -base64 18 2>/dev/null | tr -d '=/+' | head -c 24 || head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n' | head -c 24
}

read_with_default() {
    local prompt="$1"
    local default="$2"
    local value
    read -rp "  $prompt [$default]: " value
    value="${value//$'\r'/}"
    echo "${value:-$default}"
}

read_password_input() {
    local prompt="$1"
    local min_len="${2:-6}"
    local value
    while true; do
        read -rp "  $prompt: " value
        value="${value//$'\r'/}"
        if [ -z "$value" ]; then
            log_error "不能为空"
            continue
        fi
        if [ "${#value}" -lt "$min_len" ]; then
            log_error "长度至少 ${min_len} 位"
            continue
        fi
        echo "$value"
        return
    done
}

get_server_ip() {
    hostname -I 2>/dev/null | awk '{print $1}' || curl -s4 ifconfig.me 2>/dev/null || echo "YOUR_SERVER_IP"
}

# ── Docker 安装 ─────────────────────────────────────────────────────
install_docker() {
    if check_command docker; then
        log_info "Docker 已安装: $(docker --version | head -1)"
        return 0
    fi

    log_step "安装 Docker..."
    apt-get update -qq
    apt-get install -y -qq ca-certificates curl gnupg lsb-release >/dev/null

    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg 2>/dev/null
    chmod a+r /etc/apt/keyrings/docker.gpg

    echo \
      "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
      $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list >/dev/null

    apt-get update -qq
    apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-compose-plugin >/dev/null

    systemctl enable docker --now
    log_info "Docker 安装完成"
}

detect_compose() {
    if docker compose version &>/dev/null; then
        COMPOSE="docker compose"
    elif check_command docker-compose; then
        COMPOSE="docker-compose"
    else
        log_error "docker compose 不可用"
        exit 1
    fi
}

# ── 部署 ────────────────────────────────────────────────────────────
deploy() {
    banner
    log_step "开始部署 MeloVault"
    echo ""

    check_root
    install_docker
    detect_compose

    if ! docker info &>/dev/null; then
        log_error "Docker 未运行，请先启动: systemctl start docker"
        exit 1
    fi

    # 创建安装目录
    mkdir -p "$INSTALL_DIR"
    cd "$INSTALL_DIR"

    # 检查已有配置
    if [ -f .env ]; then
        log_warn "检测到已有配置 ($INSTALL_DIR/.env)"
        read -rp "  覆盖现有配置？(y/N): " overwrite
        if [[ ! "$overwrite" =~ ^[Yy]$ ]]; then
            log_info "保留现有配置，直接启动服务"
            start_services
            return
        fi
        cp .env ".env.bak.$(date +%Y%m%d_%H%M%S)"
        log_info "已备份旧配置"
    fi

    echo ""
    log_step "配置部署参数"
    echo ""

    # ── 端口配置 ──
    echo -e "  ${BOLD}端口配置${NC} ${DIM}(直接回车使用默认值)${NC}"
    APP_PORT=$(read_with_default "服务端口" "5000")
    PG_PORT=$(read_with_default "PostgreSQL 端口 (仅本机)" "5432")
    echo ""

    # ── 数据库配置 ──
    echo -e "  ${BOLD}数据库配置${NC}"
    DB_USER=$(read_with_default "数据库用户名" "postgres")
    DB_PASSWORD=$(generate_password)
    log_info "数据库密码已自动生成"
    DB_NAME=$(read_with_default "数据库名" "melovault")
    echo ""

    # ── JWT 密钥 ──
    JWT_SECRET=$(generate_secret)
    log_info "JWT 密钥已自动生成"
    echo ""

    # ── 网易云 Cookie ──
    echo -e "  ${BOLD}网易云 Cookie${NC} ${DIM}(可选，后续可在管理后台配置)${NC}"
    log_prompt "粘贴网易云 Cookie（留空跳过）:"
    read -r NETEASE_COOKIE
    NETEASE_COOKIE="${NETEASE_COOKIE//$'\r'/}"
    echo ""

    # ── LinuxDo OAuth (可选) ──
    echo -e "  ${BOLD}LinuxDo OAuth${NC} ${DIM}(可选，留空跳过)${NC}"
    read -rp "  Client ID: " LINUXDO_CLIENT_ID
    LINUXDO_CLIENT_ID="${LINUXDO_CLIENT_ID//$'\r'/}"
    LINUXDO_CLIENT_SECRET=""
    LINUXDO_REDIRECT_URI=""
    if [ -n "$LINUXDO_CLIENT_ID" ]; then
        read -rp "  Client Secret: " LINUXDO_CLIENT_SECRET
        LINUXDO_CLIENT_SECRET="${LINUXDO_CLIENT_SECRET//$'\r'/}"
        local default_redirect="http://$(get_server_ip):${APP_PORT}/api/auth/linuxdo/callback"
        LINUXDO_REDIRECT_URI=$(read_with_default "Redirect URI" "$default_redirect")
    fi
    echo ""

    # ── SMTP (可选) ──
    echo -e "  ${BOLD}SMTP 邮件${NC} ${DIM}(可选，留空跳过)${NC}"
    read -rp "  SMTP Host: " SMTP_HOST
    SMTP_HOST="${SMTP_HOST//$'\r'/}"
    SMTP_PORT=""
    SMTP_USER=""
    SMTP_PASSWORD=""
    if [ -n "$SMTP_HOST" ]; then
        SMTP_PORT=$(read_with_default "SMTP Port" "587")
        read -rp "  SMTP User: " SMTP_USER
        SMTP_USER="${SMTP_USER//$'\r'/}"
        read -rp "  SMTP Password: " SMTP_PASSWORD
        SMTP_PASSWORD="${SMTP_PASSWORD//$'\r'/}"
    fi
    echo ""

    # ── 写入 .env ──
    log_step "生成配置文件..."

    cat > .env <<ENVEOF
# MeloVault 环境配置 (自动生成于 $(date '+%Y-%m-%d %H:%M:%S'))

# ── 服务 ──
PORT=${APP_PORT}
HOST=0.0.0.0
STATIC_DIR=/app/web
DOWNLOADS_DIR=/app/downloads
COOKIE_FILE=/app/cookie.txt

# ── 数据库 ──
DB_HOST=postgres
DB_PORT=5432
DB_USER=${DB_USER}
DB_PASSWORD=${DB_PASSWORD}
DB_NAME=${DB_NAME}
DB_SSLMODE=disable

# ── 认证 ──
JWT_SECRET=${JWT_SECRET}

# ── LinuxDo OAuth (可选) ──
LINUXDO_CLIENT_ID=${LINUXDO_CLIENT_ID}
LINUXDO_CLIENT_SECRET=${LINUXDO_CLIENT_SECRET}
LINUXDO_REDIRECT_URI=${LINUXDO_REDIRECT_URI}

# ── SMTP (可选) ──
SMTP_HOST=${SMTP_HOST}
SMTP_PORT=${SMTP_PORT}
SMTP_USER=${SMTP_USER}
SMTP_PASSWORD=${SMTP_PASSWORD}

# ── Docker Compose 用 ──
POSTGRES_USER=${DB_USER}
POSTGRES_PASSWORD=${DB_PASSWORD}
POSTGRES_DB=${DB_NAME}
PG_PORT=${PG_PORT}
ENVEOF

    chmod 600 .env
    log_info "配置文件已写入 $INSTALL_DIR/.env"

    # ── 写入 cookie.txt ──
    if [ -n "$NETEASE_COOKIE" ]; then
        echo "$NETEASE_COOKIE" > cookie.txt
    else
        touch cookie.txt
    fi
    chmod 600 cookie.txt

    # ── 生成 docker-compose.yml ──
    cat > docker-compose.yml <<'COMPOSEEOF'
services:
  postgres:
    image: postgres:16-alpine
    container_name: melovault-db
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-postgres}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB:-melovault}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "127.0.0.1:${PG_PORT:-5432}:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-postgres} -d ${POSTGRES_DB:-melovault}"]
      interval: 5s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  app:
    image: ${IMAGE}
    container_name: melovault
    env_file:
      - .env
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - STATIC_DIR=/app/web
    ports:
      - "0.0.0.0:${PORT:-5000}:5000"
    volumes:
      - ./cookie.txt:/app/cookie.txt
      - downloads_data:/app/downloads
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

volumes:
  postgres_data:
  downloads_data:
COMPOSEEOF

    log_info "docker-compose.yml 已生成"
    echo ""

    start_services
}

start_services() {
    cd "$INSTALL_DIR"
    detect_compose

    log_step "拉取最新镜像..."
    $COMPOSE pull

    log_step "停止旧容器..."
    $COMPOSE down 2>/dev/null || true

    log_step "启动服务..."
    $COMPOSE up -d

    log_step "等待服务就绪..."
    local retries=0
    while [ $retries -lt 30 ]; do
        if $COMPOSE exec -T postgres pg_isready -U "${DB_USER:-postgres}" &>/dev/null; then
            break
        fi
        retries=$((retries + 1))
        sleep 2
    done

    sleep 3

    # 检查状态
    local failed
    failed=$($COMPOSE ps --services --filter "status=exited" 2>/dev/null || true)
    if [ -n "$failed" ]; then
        log_error "以下服务启动失败:"
        echo "  $failed"
        echo ""
        log_info "查看日志: cd $INSTALL_DIR && $COMPOSE logs"
        exit 1
    fi

    print_success
}

print_success() {
    local server_ip
    server_ip=$(get_server_ip)
    local app_port
    app_port=$(grep "^PORT=" "$INSTALL_DIR/.env" 2>/dev/null | cut -d'=' -f2 || echo "5000")
    local pg_port
    pg_port=$(grep "^PG_PORT=" "$INSTALL_DIR/.env" 2>/dev/null | cut -d'=' -f2 || echo "5432")

    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}  MeloVault 部署成功！${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "  ${BOLD}访问地址${NC}"
    echo -e "    本地: http://localhost:${app_port}"
    echo -e "    外部: http://${server_ip}:${app_port}"
    echo ""
    echo -e "  ${BOLD}首次使用${NC}"
    echo -e "    打开上方地址，按页面引导完成初始化设置"
    echo -e "    (创建超级管理员账户、配置网易云等)"
    echo ""
    echo -e "  ${BOLD}数据库${NC}"
    echo -e "    PostgreSQL: 127.0.0.1:${pg_port}"
    echo ""
    echo -e "  ${BOLD}文件位置${NC}"
    echo -e "    安装目录: ${INSTALL_DIR}"
    echo -e "    配置文件: ${INSTALL_DIR}/.env"
    echo -e "    Cookie:   ${INSTALL_DIR}/cookie.txt"
    echo ""
    echo -e "  ${BOLD}常用命令${NC}"
    echo -e "    查看状态: cd ${INSTALL_DIR} && docker compose ps"
    echo -e "    查看日志: cd ${INSTALL_DIR} && docker compose logs -f"
    echo -e "    重启服务: cd ${INSTALL_DIR} && docker compose restart"
    echo -e "    停止服务: cd ${INSTALL_DIR} && docker compose down"
    echo -e "    更新升级: bash deploy.sh upgrade"
    echo ""
    echo -e "  ${DIM}提示: 如需 HTTPS，请自行配置 Nginx 反向代理${NC}"
    echo ""
}

# ── 升级 ────────────────────────────────────────────────────────────
upgrade() {
    banner
    log_step "升级 MeloVault"
    echo ""

    check_root

    if [ ! -d "$INSTALL_DIR" ] || [ ! -f "$INSTALL_DIR/docker-compose.yml" ]; then
        log_warn "未检测到已有安装，将进入部署流程"
        deploy
        return
    fi

    cd "$INSTALL_DIR"
    detect_compose

    cp .env ".env.bak.upgrade.$(date +%Y%m%d_%H%M%S)"
    log_info "已备份配置"

    log_step "拉取最新镜像..."
    $COMPOSE pull app

    log_step "重启应用容器 (数据库不受影响)..."
    $COMPOSE up -d --no-deps app

    sleep 3

    local failed
    failed=$($COMPOSE ps --services --filter "status=exited" 2>/dev/null | grep "^app$" || true)
    if [ -n "$failed" ]; then
        log_error "应用启动失败，查看日志:"
        $COMPOSE logs --tail=50 app
        exit 1
    fi

    echo ""
    log_info "升级完成！"
    echo -e "  查看状态: cd ${INSTALL_DIR} && $COMPOSE ps"
    echo -e "  查看日志: cd ${INSTALL_DIR} && $COMPOSE logs -f app"
    echo ""
}

# ── 卸载 ────────────────────────────────────────────────────────────
uninstall() {
    banner
    log_warn "卸载 MeloVault"
    echo ""

    check_root

    if [ ! -d "$INSTALL_DIR" ]; then
        log_error "未找到安装目录: $INSTALL_DIR"
        exit 1
    fi

    cd "$INSTALL_DIR"
    detect_compose

    read -rp "  是否同时删除数据库数据卷？此操作不可恢复！(y/N): " del_volumes
    echo ""

    if [[ "$del_volumes" =~ ^[Yy]$ ]]; then
        log_warn "停止容器并删除数据卷..."
        $COMPOSE down -v --remove-orphans 2>/dev/null || true
    else
        log_info "停止容器，保留数据卷..."
        $COMPOSE down --remove-orphans 2>/dev/null || true
    fi

    read -rp "  是否删除安装目录 (${INSTALL_DIR})？(y/N): " del_dir
    if [[ "$del_dir" =~ ^[Yy]$ ]]; then
        cd /
        rm -rf "$INSTALL_DIR"
        log_info "安装目录已删除"
    else
        log_info "保留安装目录"
    fi

    echo ""
    log_info "卸载完成"
    echo ""
}

# ── 状态查看 ────────────────────────────────────────────────────────
status() {
    banner

    if [ ! -d "$INSTALL_DIR" ] || [ ! -f "$INSTALL_DIR/docker-compose.yml" ]; then
        log_error "MeloVault 未安装"
        exit 1
    fi

    cd "$INSTALL_DIR"
    detect_compose

    echo -e "  ${BOLD}容器状态${NC}"
    echo ""
    $COMPOSE ps
    echo ""

    local app_status
    app_status=$($COMPOSE ps --format '{{.State}}' app 2>/dev/null || echo "unknown")
    if [ "$app_status" = "running" ]; then
        local app_port
        app_port=$(grep "^PORT=" .env 2>/dev/null | cut -d'=' -f2 || echo "5000")
        log_info "服务运行中: http://$(get_server_ip):${app_port}"
    else
        log_warn "服务未运行"
    fi
    echo ""
}

# ── 交互菜单 ────────────────────────────────────────────────────────
show_menu() {
    banner
    echo -e "  ${BOLD}请选择操作:${NC}"
    echo ""
    echo -e "    ${CYAN}1${NC})  一键部署 ${DIM}(首次安装/重装)${NC}"
    echo -e "    ${CYAN}2${NC})  升级更新 ${DIM}(拉取最新镜像，保留数据)${NC}"
    echo -e "    ${CYAN}3${NC})  查看状态 ${DIM}(容器运行情况)${NC}"
    echo -e "    ${CYAN}4${NC})  卸载移除 ${DIM}(停止容器，可选删除数据)${NC}"
    echo -e "    ${CYAN}0${NC})  退出"
    echo ""

    while true; do
        read -rp "  请输入 [0-4]: " choice
        case "${choice//$'\r'/}" in
            1) echo ""; deploy; break ;;
            2) echo ""; upgrade; break ;;
            3) echo ""; status; break ;;
            4) echo ""; uninstall; break ;;
            0) echo ""; log_info "已退出"; exit 0 ;;
            *) log_warn "无效选择" ;;
        esac
    done
}

# ── 入口 ────────────────────────────────────────────────────────────
case "${1:-}" in
    deploy|install|1)       deploy ;;
    upgrade|update|2)       upgrade ;;
    status|3)               status ;;
    uninstall|remove|4)     uninstall ;;
    -h|--help|help)
        banner
        echo "用法: bash deploy.sh [命令]"
        echo ""
        echo "命令:"
        echo "  deploy      一键部署 (首次安装/重装)"
        echo "  upgrade     升级更新 (拉取最新镜像)"
        echo "  status      查看运行状态"
        echo "  uninstall   卸载移除"
        echo ""
        echo "不传参数进入交互菜单。"
        echo ""
        echo "环境变量:"
        echo "  MELOVAULT_DIR  自定义安装目录 (默认 /opt/melovault)"
        echo ""
        ;;
    "")
        show_menu
        ;;
    *)
        log_warn "未知参数: $1"
        echo "运行 bash deploy.sh --help 查看帮助"
        exit 1
        ;;
esac
