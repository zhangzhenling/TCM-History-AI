#!/bin/bash
set -e

# ============================================================
# TCM-History-AI 一键部署脚本 (Linux)
# 支持: 一键打包、启动、停止、重启、查看状态
# 用法:
#   ./deploy-linux.sh build            # 仅打包（后端+前端）
#   ./deploy-linux.sh start            # 打包并启动所有服务
#   ./deploy-linux.sh start:dev        # 以开发模式启动（直接 go run）
#   ./deploy-linux.sh stop             # 停止所有服务
#   ./deploy-linux.sh restart          # 重启所有服务
#   ./deploy-linux.sh status           # 查看服务状态
#   ./deploy-linux.sh logs [service]   # 查看日志
#   ./deploy-linux.sh clean            # 清理构建产物
# ============================================================

# ---------- 颜色输出 ----------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[OK]${NC} $1"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $1"; }
error()   { echo -e "${RED}[ERROR]${NC} $1"; }
step()    { echo -e "\n${GREEN}====> $1${NC}"; }

# ---------- 配置 ----------
PROJECT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
BACKEND_DIR="${PROJECT_DIR}/backend"
FRONTEND_DIR="${PROJECT_DIR}/frontend"
BIN_DIR="${BACKEND_DIR}/bin"
PID_DIR="${PROJECT_DIR}/.deploy/pids"
LOG_DIR="${PROJECT_DIR}/.deploy/logs"

# 服务列表
SERVICES=("gateway" "user-service" "history-service" "knowledge-service" "graph-service" "ai-service" "learning-service")

# 端口配置
declare -A SERVICE_PORTS=(
    ["gateway"]="8080"
    ["user-service"]="9001"
    ["history-service"]="9002"
    ["knowledge-service"]="9003"
    ["graph-service"]="9004"
    ["ai-service"]="9005"
    ["learning-service"]="9006"
)

# ---------- 环境检查 ----------
check_dependency() {
    local cmd=$1
    local pkg=$2
    local install_hint=$3

    if ! command -v "$cmd" &> /dev/null; then
        error "$cmd 未安装"
        if [ -n "$install_hint" ]; then
            info "安装提示: $install_hint"
        fi
        return 1
    fi
    success "$cmd 已安装: $($cmd version 2>/dev/null || $cmd --version 2>/dev/null || echo '已安装')"
    return 0
}

check_environment() {
    step "检查运行环境"

    local missing=0

    # Go
    if ! command -v go &> /dev/null; then
        error "Go 未安装，请安装 Go 1.22+"
        info "下载地址: https://go.dev/dl/"
        ((missing++))
    else
        local go_version=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+' | head -1)
        local go_major=$(echo $go_version | cut -d. -f1)
        local go_minor=$(echo $go_version | cut -d. -f2)
        if [ "$go_major" -lt 1 ] || ([ "$go_major" -eq 1 ] && [ "$go_minor" -lt 22 ]); then
            warn "Go 版本过低 (当前: $go_version)，建议 Go 1.22+"
        else
            success "Go $go_version"
        fi
    fi

    # Node.js
    if ! command -v node &> /dev/null; then
        error "Node.js 未安装，请安装 Node.js 20+"
        info "安装: curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && apt install -y nodejs"
        ((missing++))
    else
        local node_version=$(node --version | sed 's/v//' | cut -d. -f1)
        if [ "$node_version" -lt 20 ]; then
            warn "Node.js 版本过低 (当前: $(node --version))，建议 Node.js 20+"
        else
            success "Node.js $(node --version)"
        fi
    fi

    # pnpm
    if ! command -v pnpm &> /dev/null; then
        warn "pnpm 未安装，尝试自动安装..."
        npm install -g pnpm@9 2>/dev/null || {
            error "pnpm 安装失败，请手动安装: npm install -g pnpm@9"
            ((missing++))
        }
    else
        success "pnpm $(pnpm --version)"
    fi

    # Git
    if ! command -v git &> /dev/null; then
        error "Git 未安装"
        ((missing++))
    else
        success "Git $(git --version | grep -oP '[0-9]+\.[0-9]+\.[0-9]+')"
    fi

    # Make (可选，脚本内直接调用 go)
    if command -v make &> /dev/null; then
        success "Make 已安装"
    else
        warn "Make 未安装，将使用 go 命令直接编译"
    fi

    if [ "$missing" -gt 0 ]; then
        error "缺少 $missing 个必要依赖，请先安装后重试"
        exit 1
    fi

    success "环境检查完成"
}

# ---------- 目录准备 ----------
prepare_dirs() {
    mkdir -p "$PID_DIR" "$LOG_DIR"
    info "PID 目录: $PID_DIR"
    info "日志目录: $LOG_DIR"
}

# ---------- 后端编译 ----------
build_backend() {
    step "编译后端服务"

    cd "$BACKEND_DIR"
    mkdir -p "$BIN_DIR"

    # 检查 go.mod
    if [ ! -f "go.mod" ]; then
        error "在 $BACKEND_DIR 未找到 go.mod"
        exit 1
    fi

    # 下载依赖
    info "下载 Go 依赖..."
    go mod download 2>/dev/null || go mod tidy

    # 编译所有服务
    for svc in "${SERVICES[@]}"; do
        local cmd_dir="$BACKEND_DIR/$svc/cmd/$svc"
        if [ ! -d "$cmd_dir" ]; then
            warn "跳过 $svc (cmd 目录不存在)"
            continue
        fi

        info "编译 $svc ..."
        if go build -o "$BIN_DIR/$svc" "./$svc/cmd/$svc"; then
            success "$svc 编译成功"
        else
            error "$svc 编译失败"
            exit 1
        fi
    done

    success "后端编译完成，产物位于 $BIN_DIR/"
    ls -la "$BIN_DIR/"
}

# ---------- 前端构建 ----------
build_frontend() {
    step "构建前端"

    cd "$FRONTEND_DIR"

    if [ ! -f "package.json" ]; then
        error "在 $FRONTEND_DIR 未找到 package.json"
        exit 1
    fi

    # 安装依赖
    if [ ! -d "node_modules" ]; then
        info "安装前端依赖 (pnpm install)..."
        pnpm install || {
            error "pnpm install 失败"
            exit 1
        }
    else
        info "前端依赖已安装，跳过 pnpm install"
    fi

    # 构建
    info "构建前端 (pnpm build)..."
    pnpm build || {
        error "前端构建失败"
        exit 1
    }

    success "前端构建完成"
}

# ---------- 打包（后端+前端） ----------
do_build() {
    step "========== 开始打包 =========="
    build_backend
    build_frontend
    step "========== 打包完成 =========="
    success "所有产物已生成！"
    info "后端二进制: $BIN_DIR/"
    info "前端产物: $FRONTEND_DIR/apps/learner/dist/"
    info "管理端产物: $FRONTEND_DIR/apps/admin/dist/"
}

# ---------- 启动单个服务 ----------
start_service() {
    local svc=$1
    local mode=${2:-prod}
    local cmd_dir="$BACKEND_DIR/$svc/cmd/$svc"
    local binary="$BIN_DIR/$svc"
    local pid_file="$PID_DIR/$svc.pid"
    local log_file="$LOG_DIR/$svc.log"

    # 检查是否已在运行
    if [ -f "$pid_file" ] && kill -0 "$(cat "$pid_file")" 2>/dev/null; then
        warn "$svc 已在运行 (PID: $(cat "$pid_file"))"
        return 0
    fi

    if [ "$mode" == "dev" ]; then
        # 开发模式：使用 go run
        info "以开发模式启动 $svc (go run)..."
        cd "$BACKEND_DIR"
        nohup go run "./$svc/cmd/$svc" > "$log_file" 2>&1 &
        local pid=$!
        echo "$pid" > "$pid_file"
        sleep 2
        if kill -0 "$pid" 2>/dev/null; then
            success "$svc 启动成功 (PID: $pid, 端口: ${SERVICE_PORTS[$svc]})"
        else
            error "$svc 启动失败，查看日志: $log_file"
            tail -20 "$log_file"
            rm -f "$pid_file"
            return 1
        fi
    else
        # 生产模式：使用编译好的二进制
        if [ ! -f "$binary" ]; then
            error "$binary 不存在，请先执行 build"
            return 1
        fi

        info "启动 $svc..."
        cd "$BACKEND_DIR"
        nohup "$binary" > "$log_file" 2>&1 &
        local pid=$!
        echo "$pid" > "$pid_file"
        sleep 2
        if kill -0 "$pid" 2>/dev/null; then
            success "$svc 启动成功 (PID: $pid, 端口: ${SERVICE_PORTS[$svc]})"
        else
            error "$svc 启动失败，查看日志: $log_file"
            tail -20 "$log_file"
            rm -f "$pid_file"
            return 1
        fi
    fi
}

# ---------- 启动所有服务 ----------
do_start() {
    local mode=${1:-prod}

    step "========== 启动所有服务 (模式: $mode) =========="

    # 先打包
    if [ "$mode" == "prod" ] && [ ! -f "$BIN_DIR/gateway" ]; then
        warn "未找到编译产物，先执行打包..."
        do_build
    elif [ "$mode" == "prod" ]; then
        # 检查所有二进制是否存在
        local need_build=false
        for svc in "${SERVICES[@]}"; do
            if [ ! -f "$BIN_DIR/$svc" ]; then
                warn "缺少 $svc 二进制，需要重新打包"
                need_build=true
                break
            fi
        done
        if [ "$need_build" = true ]; then
            do_build
        fi
    fi

    # 按依赖顺序启动
    local startup_order=("gateway" "user-service" "history-service" "graph-service" "knowledge-service" "ai-service" "learning-service")
    for svc in "${startup_order[@]}"; do
        start_service "$svc" "$mode"
        sleep 1
    done

    step "========== 所有服务已启动 =========="
    echo ""
    info "服务访问地址:"
    echo -e "  ${BLUE}前端:${NC}    http://localhost (需配合 Nginx)"
    echo -e "  ${BLUE}Gateway:${NC} http://localhost:${SERVICE_PORTS['gateway']}"
    echo -e "  ${BLUE}API:${NC}     http://localhost:${SERVICE_PORTS['gateway']}/api/v1"
    echo ""
    info "查看状态: $0 status"
    info "查看日志: $0 logs <service>"
    info "停止服务: $0 stop"
}

# ---------- 停止单个服务 ----------
stop_service() {
    local svc=$1
    local pid_file="$PID_DIR/$svc.pid"

    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            info "停止 $svc (PID: $pid)..."
            kill "$pid" 2>/dev/null
            sleep 1
            # 强制终止
            kill -9 "$pid" 2>/dev/null || true
            rm -f "$pid_file"
            success "$svc 已停止"
        else
            warn "$svc 未在运行 (旧 PID: $pid)"
            rm -f "$pid_file"
        fi
    else
        warn "$svc 未运行"
    fi
}

# ---------- 停止所有服务 ----------
do_stop() {
    step "========== 停止所有服务 =========="
    for svc in "${SERVICES[@]}"; do
        stop_service "$svc"
    done
    success "所有服务已停止"
}

# ---------- 重启所有服务 ----------
do_restart() {
    local mode=${1:-prod}
    do_stop
    sleep 2
    do_start "$mode"
}

# ---------- 查看状态 ----------
do_status() {
    step "========== 服务状态 =========="

    local all_running=true
    printf "%-20s %-10s %-10s %s\n" "服务" "状态" "PID" "端口"
    echo "------------------------------------------------------------"

    for svc in "${SERVICES[@]}"; do
        local pid_file="$PID_DIR/$svc.pid"
        local status="${RED}已停止${NC}"
        local pid="-"

        if [ -f "$pid_file" ]; then
            pid=$(cat "$pid_file")
            if kill -0 "$pid" 2>/dev/null; then
                status="${GREEN}运行中${NC}"
            else
                status="${RED}已停止${NC}"
                pid="-"
                rm -f "$pid_file"
                all_running=false
            fi
        else
            all_running=false
        fi

        printf "%-20s " "$svc"
        echo -e "$status"
        printf "%-10s %s\n" "$pid" "${SERVICE_PORTS[$svc]}"
    done

    echo ""
    if [ "$all_running" = true ]; then
        success "所有服务正常运行"
    else
        warn "部分服务未运行，可使用 '$0 start' 启动"
    fi
}

# ---------- 查看日志 ----------
do_logs() {
    local svc=$1
    local lines=${2:-50}

    if [ -n "$svc" ]; then
        local log_file="$LOG_DIR/$svc.log"
        if [ -f "$log_file" ]; then
            step "========== $svc 日志 (最近 $lines 行) =========="
            tail -n "$lines" "$log_file"
        else
            error "日志文件不存在: $log_file"
        fi
    else
        # 查看所有服务的最新日志
        for svc in "${SERVICES[@]}"; do
            local log_file="$LOG_DIR/$svc.log"
            if [ -f "$log_file" ] && [ -s "$log_file" ]; then
                echo ""
                step "========== $svc 日志 =========="
                tail -n 10 "$log_file"
            fi
        done
    fi
}

# ---------- 清理构建产物 ----------
do_clean() {
    step "========== 清理构建产物 =========="

    # 停止所有服务
    do_stop

    # 清理后端产物
    if [ -d "$BIN_DIR" ]; then
        rm -rf "$BIN_DIR"
        info "已删除 $BIN_DIR/"
    fi

    # 清理 PID 和日志
    if [ -d "$PID_DIR" ]; then
        rm -rf "$PID_DIR"
        info "已删除 $PID_DIR/"
    fi
    if [ -d "$LOG_DIR" ]; then
        rm -rf "$LOG_DIR"
        info "已删除 $LOG_DIR/"
    fi

    # 清理前端产物
    if [ -d "$FRONTEND_DIR/apps/learner/dist" ]; then
        rm -rf "$FRONTEND_DIR/apps/learner/dist"
        info "已删除前端构建产物"
    fi
    if [ -d "$FRONTEND_DIR/apps/admin/dist" ]; then
        rm -rf "$FRONTEND_DIR/apps/admin/dist"
    fi

    success "清理完成"
}

# ---------- 启动前端开发服务器 ----------
do_frontend_dev() {
    step "========== 启动前端开发服务器 =========="
    cd "$FRONTEND_DIR"
    if [ ! -d "node_modules" ]; then
        info "安装前端依赖..."
        pnpm install
    fi
    info "启动前端开发服务器 (学习端)..."
    pnpm dev
}

# ---------- 主入口 ----------
main() {
    local action=${1:-help}
    local arg1=${2:-}
    local arg2=${3:-}

    case "$action" in
        build)
            check_environment
            prepare_dirs
            do_build
            ;;

        start)
            check_environment
            prepare_dirs
            do_start "prod"
            ;;

        "start:dev")
            check_environment
            prepare_dirs
            do_start "dev"
            ;;

        stop)
            prepare_dirs
            do_stop
            ;;

        restart)
            check_environment
            prepare_dirs
            local mode=${arg1:-prod}
            do_restart "$mode"
            ;;

        status)
            prepare_dirs
            do_status
            ;;

        logs)
            prepare_dirs
            do_logs "$arg1" "$arg2"
            ;;

        clean)
            do_clean
            ;;

        "dev:frontend")
            check_environment
            do_frontend_dev
            ;;

        *)
            echo "TCM-History-AI 一键部署脚本 (Linux)"
            echo ""
            echo "用法: $0 <命令> [参数]"
            echo ""
            echo "命令:"
            echo "  build            仅打包（编译后端 + 构建前端）"
            echo "  start            打包并以生产模式启动所有服务"
            echo "  start:dev        以开发模式启动所有服务（go run）"
            echo "  stop             停止所有服务"
            echo "  restart [mode]   重启所有服务 (mode: prod/dev, 默认 prod)"
            echo "  status           查看服务运行状态"
            echo "  logs [service]   查看日志 (可选服务名)"
            echo "  clean            清理构建产物和日志"
            echo "  dev:frontend     启动前端开发服务器"
            echo ""
            echo "服务列表:"
            echo "  ${SERVICES[*]}"
            echo ""
            echo "示例:"
            echo "  $0 build              # 打包"
            echo "  $0 start              # 打包并启动"
            echo "  $0 logs gateway       # 查看 gateway 日志"
            echo "  $0 restart dev        # 以开发模式重启"
            ;;
    esac
}

main "$@"
