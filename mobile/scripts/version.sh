#!/usr/bin/env bash
# 移动端版本管理脚本
# 用法：
#   ./scripts/version.sh get               # 查看当前版本
#   ./scripts/version.sh bump patch        # 升级补丁版本 (1.0.0 -> 1.0.1)
#   ./scripts/version.sh bump minor        # 升级次要版本 (1.0.0 -> 1.1.0)
#   ./scripts/version.sh bump major        # 升级主要版本 (1.0.0 -> 2.0.0)
#   ./scripts/version.sh set 1.2.3+10      # 设置指定版本

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
PUBSPEC="$PROJECT_DIR/pubspec.yaml"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

get_version() {
    grep '^version:' "$PUBSPEC" | sed 's/version: //'
}

parse_version() {
    local version="$1"
    VERSION_NAME=$(echo "$version" | cut -d'+' -f1)
    VERSION_CODE=$(echo "$version" | cut -d'+' -f2)
    if [ "$VERSION_NAME" = "$VERSION_CODE" ]; then
        VERSION_CODE="1"
    fi
}

set_version() {
    local new_version="$1"
    parse_version "$new_version"
    local full_version="${VERSION_NAME}+${VERSION_CODE}"
    
    sed -i.bak "s/^version: .*/version: $full_version/" "$PUBSPEC"
    rm -f "${PUBSPEC}.bak"
    
    echo -e "${GREEN}版本已更新为: $full_version${NC}"
}

bump_version() {
    local bump_type="$1"
    local current=$(get_version)
    parse_version "$current"
    
    local major minor patch
    IFS='.' read -r major minor patch <<< "$VERSION_NAME"
    
    case "$bump_type" in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        patch)
            patch=$((patch + 1))
            ;;
        *)
            echo -e "${RED}未知的版本类型: $bump_type${NC}"
            echo "使用: major / minor / patch"
            exit 1
            ;;
    esac
    
    VERSION_CODE=$((VERSION_CODE + 1))
    VERSION_NAME="${major}.${minor}.${patch}"
    
    set_version "${VERSION_NAME}+${VERSION_CODE}"
}

# 主逻辑
case "${1:-}" in
    get|"")
        echo -e "${YELLOW}当前版本: $(get_version)${NC}"
        ;;
    bump)
        bump_type="${2:-patch}"
        current=$(get_version)
        echo -e "当前版本: $current"
        echo -n "确认升级 $bump_type 版本? [y/N] "
        read -r confirm
        if [ "$confirm" = "y" ] || [ "$confirm" = "Y" ]; then
            bump_version "$bump_type"
        else
            echo "已取消"
        fi
        ;;
    set)
        if [ -z "${2:-}" ]; then
            echo -e "${RED}请指定版本号，例如: 1.2.3+10${NC}"
            exit 1
        fi
        set_version "$2"
        ;;
    help|--help|-h)
        echo "移动端版本管理脚本"
        echo ""
        echo "用法:"
        echo "  ./scripts/version.sh get               查看当前版本"
        echo "  ./scripts/version.sh bump patch        升级补丁版本 (默认)"
        echo "  ./scripts/version.sh bump minor        升级次要版本"
        echo "  ./scripts/version.sh bump major        升级主要版本"
        echo "  ./scripts/version.sh set 1.2.3+10      设置指定版本"
        echo "  ./scripts/version.sh help              显示帮助"
        ;;
    *)
        echo -e "${RED}未知命令: $1${NC}"
        echo "使用 help 查看帮助"
        exit 1
        ;;
esac
