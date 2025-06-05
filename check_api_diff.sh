#!/bin/bash

# API Diff Checker Script
# Usage: ./check_api_diff.sh <old_version> <new_version>
# Example: ./check_api_diff.sh HEAD~1 HEAD

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 帮助信息
show_help() {
    echo "Usage: $0 <old_version> <new_version>"
    echo ""
    echo "Parameters:"
    echo "  old_version    Git reference for old version (e.g., HEAD~1, v1.0.0, commit_hash)"
    echo "  new_version    Git reference for new version (e.g., HEAD, main, commit_hash)"
    echo ""
    echo "Examples:"
    echo "  $0 HEAD~1 HEAD                # Compare last commit with current"
    echo "  $0 v1.0.0 v1.1.0             # Compare two tags"
    echo "  $0 abc123 def456              # Compare two commits"
    echo ""
    echo "This script will:"
    echo "  1. Find all directories containing Go files"
    echo "  2. Generate API files for each directory in both versions"
    echo "  3. Compare APIs and report incompatible changes"
}

# 检查参数
if [ $# -ne 2 ]; then
    echo -e "${RED}Error: Exactly 2 parameters required${NC}"
    show_help
    exit 1
fi

OLD_VERSION="$1"
NEW_VERSION="$2"

# 检查git仓库
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo -e "${RED}Error: Not in a git repository${NC}"
    exit 1
fi

# 检查版本是否存在
if ! git rev-parse --verify "$OLD_VERSION" > /dev/null 2>&1; then
    echo -e "${RED}Error: Old version '$OLD_VERSION' not found${NC}"
    exit 1
fi

if ! git rev-parse --verify "$NEW_VERSION" > /dev/null 2>&1; then
    echo -e "${RED}Error: New version '$NEW_VERSION' not found${NC}"
    exit 1
fi

echo -e "${BLUE}=== API Diff Checker ===${NC}"
echo -e "Old version: ${YELLOW}$OLD_VERSION${NC}"
echo -e "New version: ${YELLOW}$NEW_VERSION${NC}"
echo ""

# 创建临时目录
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# 获取当前分支
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "detached")

# 查找所有包含go文件的目录
find_go_directories() {
    find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | \
    sed 's|/[^/]*\.go$||' | \
    sort -u | \
    while read dir; do
        # 跳过仅包含测试文件的目录
        if ls "$dir"/*.go 2>/dev/null | grep -v "_test.go" > /dev/null; then
            echo "$dir"
        fi
    done
}

echo -e "${BLUE}Finding directories with Go files...${NC}"
GO_DIRS=($(find_go_directories))

if [ ${#GO_DIRS[@]} -eq 0 ]; then
    echo -e "${YELLOW}No directories with Go files found${NC}"
    exit 0
fi

echo -e "Found ${GREEN}${#GO_DIRS[@]}${NC} directories with Go files"
echo ""

# 记录有变化的包
CHANGED_PACKAGES=()
TOTAL_INCOMPATIBLE=0

# 处理每个目录
for dir in "${GO_DIRS[@]}"; do
    # 跳过internal目录、examples目录、test目录和一些特殊目录
    if [[ "$dir" == *"/internal/"* ]] || [[ "$dir" == *"/testdata/"* ]] || [[ "$dir" == *"/vendor/"* ]] || \
       [[ "$dir" == "./examples"* ]] || [[ "$dir" == "./test"* ]] || [[ "$dir" == *"/examples/"* ]] || [[ "$dir" == *"/test/"* ]]; then
        continue
    fi
    
    echo -e "${BLUE}Checking package: ${NC}$dir"
    
    # 生成API文件名
    SAFE_DIR=$(echo "$dir" | tr '/' '_' | tr '.' '_')
    OLD_API="$TEMP_DIR/old_${SAFE_DIR}.api"
    NEW_API="$TEMP_DIR/new_${SAFE_DIR}.api"
    DIFF_FILE="$TEMP_DIR/diff_${SAFE_DIR}.txt"
    
    # 生成旧版本API
    echo -n "  Generating old API... "
    if git checkout "$OLD_VERSION" --quiet 2>/dev/null; then
        if apidiff -w "$OLD_API" "$dir" 2>/dev/null; then
            echo -e "${GREEN}✓${NC}"
        else
            echo -e "${YELLOW}⚠ (skipped - failed to generate)${NC}"
            continue
        fi
    else
        echo -e "${RED}✗ (failed to checkout old version)${NC}"
        continue
    fi
    
    # 生成新版本API
    echo -n "  Generating new API... "
    if git checkout "$NEW_VERSION" --quiet 2>/dev/null; then
        if apidiff -w "$NEW_API" "$dir" 2>/dev/null; then
            echo -e "${GREEN}✓${NC}"
        else
            echo -e "${YELLOW}⚠ (skipped - failed to generate)${NC}"
            continue
        fi
    else
        echo -e "${RED}✗ (failed to checkout new version)${NC}"
        continue
    fi
    
    # 比较API
    echo -n "  Comparing APIs... "
    if apidiff -incompatible "$OLD_API" "$NEW_API" > "$DIFF_FILE" 2>/dev/null; then
        if [ -s "$DIFF_FILE" ]; then
            # 有不兼容变化
            INCOMPATIBLE_COUNT=$(wc -l < "$DIFF_FILE")
            TOTAL_INCOMPATIBLE=$((TOTAL_INCOMPATIBLE + INCOMPATIBLE_COUNT))
            CHANGED_PACKAGES+=("$dir")
            echo -e "${RED}✗ ($INCOMPATIBLE_COUNT incompatible changes)${NC}"
            
            # 显示变化详情
            echo -e "    ${RED}Incompatible changes:${NC}"
            while IFS= read -r line; do
                echo -e "    ${RED}  - $line${NC}"
            done < "$DIFF_FILE"
        else
            echo -e "${GREEN}✓ (no incompatible changes)${NC}"
        fi
    else
        echo -e "${YELLOW}⚠ (comparison failed)${NC}"
    fi
    
    echo ""
done

# 恢复到原来的分支/状态
if [ "$CURRENT_BRANCH" != "detached" ]; then
    git checkout "$CURRENT_BRANCH" --quiet 2>/dev/null || true
else
    git checkout "$NEW_VERSION" --quiet 2>/dev/null || true
fi

# 输出总结
echo -e "${BLUE}=== Summary ===${NC}"
echo -e "Total packages checked: ${GREEN}${#GO_DIRS[@]}${NC}"
echo -e "Packages with incompatible changes: ${RED}${#CHANGED_PACKAGES[@]}${NC}"
echo -e "Total incompatible changes: ${RED}$TOTAL_INCOMPATIBLE${NC}"

if [ ${#CHANGED_PACKAGES[@]} -gt 0 ]; then
    echo ""
    echo -e "${RED}Packages with incompatible API changes:${NC}"
    for pkg in "${CHANGED_PACKAGES[@]}"; do
        echo -e "  ${RED}• $pkg${NC}"
    done
    exit 1
else
    echo -e "${GREEN}No incompatible API changes found!${NC}"
    exit 0
fi 