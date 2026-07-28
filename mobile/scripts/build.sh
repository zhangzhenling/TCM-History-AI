#!/usr/bin/env bash
# 移动端构建脚本
# 用法：
#   ./scripts/build.sh android              # 构建 Android APK (prod release)
#   ./scripts/build.sh android --flavor dev # 构建指定 flavor
#   ./scripts/build.sh ios                  # 构建 iOS IPA
#   ./scripts/build.sh ios --no-codesign    # 构建 iOS 无签名
#   ./scripts/build.sh all                  # 构建全部

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

cd "$PROJECT_DIR"

# 默认值
FLAVOR="prod"
NO_CODESIGN=false
BUILD_TYPE="release"

# 解析参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --flavor)
                FLAVOR="$2"
                shift 2
                ;;
            --no-codesign)
                NO_CODESIGN=true
                shift
                ;;
            --debug)
                BUILD_TYPE="debug"
                shift
                ;;
            --profile)
                BUILD_TYPE="profile"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
}

build_android() {
    echo -e "${CYAN}>>> 构建 Android $BUILD_TYPE ($FLAVOR)${NC}"
    
    local build_args=("apk" "--$BUILD_TYPE")
    
    if [ "$FLAVOR" != "prod" ]; then
        build_args+=("--flavor" "$FLAVOR")
    fi
    
    flutter build "${build_args[@]}"
    
    echo -e "${GREEN}✅ Android 构建完成${NC}"
    
    # 显示产物信息
    local apk_dir="build/app/outputs/flutter-apk"
    if [ -d "$apk_dir" ]; then
        echo -e "${YELLOW}产物目录: $apk_dir${NC}"
        ls -lh "$apk_dir"/*.apk 2>/dev/null || true
    fi
}

build_ios() {
    echo -e "${CYAN}>>> 构建 iOS $BUILD_TYPE${NC}"
    
    if [ "$NO_CODESIGN" = true ]; then
        flutter build ios "--$BUILD_TYPE" --no-codesign
        echo -e "${GREEN}✅ iOS 构建完成（无签名）${NC}"
    else
        flutter build ipa "--$BUILD_TYPE"
        echo -e "${GREEN}✅ iOS IPA 构建完成${NC}"
    fi
    
    # 显示产物信息
    local ios_dir="build/ios"
    if [ -d "$ios_dir" ]; then
        echo -e "${YELLOW}产物目录: $ios_dir${NC}"
        find "$ios_dir" -name "*.ipa" -o -name "Runner.app" -maxdepth 3 2>/dev/null | head -5
    fi
}

build_all() {
    echo -e "${CYAN}>>> 构建全部平台${NC}"
    echo ""
    
    build_android
    echo ""
    build_ios
    echo ""
    echo -e "${GREEN}🎉 全部构建完成${NC}"
}

# 主逻辑
command="${1:-}"
shift || true

case "$command" in
    android)
        parse_args "$@"
        build_android
        ;;
    ios)
        parse_args "$@"
        build_ios
        ;;
    all|both)
        parse_args "$@"
        build_all
        ;;
    help|--help|-h|"")
        echo "移动端构建脚本"
        echo ""
        echo "用法:"
        echo "  ./scripts/build.sh android [选项]    构建 Android"
        echo "  ./scripts/build.sh ios [选项]        构建 iOS"
        echo "  ./scripts/build.sh all [选项]        构建全部"
        echo ""
        echo "选项:"
        echo "  --flavor <dev|staging|prod>   指定 flavor (默认: prod)"
        echo "  --debug                       Debug 构建"
        echo "  --profile                     Profile 构建"
        echo "  --no-codesign                 iOS 无签名构建"
        echo "  --help                        显示帮助"
        ;;
    *)
        echo -e "${RED}未知命令: $command${NC}"
        echo "使用 help 查看帮助"
        exit 1
        ;;
esac
