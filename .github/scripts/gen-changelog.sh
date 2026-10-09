#!/bin/sh
#
# Generate AI-summarized release notes from git commits.
# Usage: .github/scripts/gen-changelog.sh [previous_tag]
#
# Requires GEMINI_API_KEY (preferred), ANTHROPIC_API_KEY, or OPENAI_API_KEY.
# Falls back to raw commit list if no API key is set.
#
set -e

PREV_TAG="${1:-$(git describe --tags --abbrev=0 HEAD^ 2>/dev/null || echo "")}"
CURR_TAG="$(git describe --tags --abbrev=0 HEAD 2>/dev/null || echo "HEAD")"

if [ -n "$PREV_TAG" ]; then
    COMMITS=$(git log "${PREV_TAG}..${CURR_TAG}" --pretty=format:"- %s" --no-merges)
    RANGE="${PREV_TAG}..${CURR_TAG}"
else
    COMMITS=$(git log --pretty=format:"- %s" --no-merges -50)
    RANGE="last 50 commits"
fi

if [ -z "$COMMITS" ]; then
    echo "No commits found in range ${RANGE}"
    exit 0
fi

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

cat > "$TMPDIR/prompt.txt" <<PROMPT_EOF
Bạn là người viết ghi chú phát hành cho ainovel-cli, công cụ dòng lệnh viết bằng Go (một engine viết tiểu thuyết bằng AI).
Dựa vào lịch sử commit Git bên dưới, hãy tạo ghi chú phát hành Markdown bằng tiếng Việt, ngắn gọn, rõ ràng, hướng tới người dùng.

Quy tắc:
- Viết bằng tiếng Việt
- Nhóm nội dung theo: Tính năng mới, Sửa lỗi, Tối ưu hiệu năng, Tái cấu trúc, Khác; nhóm nào không có nội dung thì bỏ qua
- Mỗi mục một dòng, ngắn gọn, không ghi commit hash hay tên tác giả
- Bỏ tiền tố conventional commit như feat:, fix:, perf:, refactor:…
- Gộp các commit tương tự hoặc trùng lặp, không liệt kê máy móc từng commit
- Dùng cách diễn đạt hướng người dùng, nêu bật thay đổi và tác động thực tế
- Tập trung vào thay đổi người dùng cảm nhận được, ví dụ quy trình phát hành, đóng gói file chạy, hành vi CLI/TUI, quy trình viết, hỗ trợ model và tài liệu
- Chỉ xuất nội dung Markdown, không có lời mở đầu, giải thích hay tổng kết

Lịch sử commit (${RANGE}):
${COMMITS}
PROMPT_EOF

# Build JSON body with jq (reads from file to handle special chars).
build_body() { jq -Rs "$1" < "$TMPDIR/prompt.txt" > "$TMPDIR/body.json"; }

# Extract text from JSON response (python3 handles control chars reliably).
extract() { python3 -c "import json,sys; d=json.load(open('$TMPDIR/result.json')); print($1)"; }

fallback() {
    echo "## What's Changed"
    echo ""
    echo "$COMMITS"
}

# Try Gemini first, then Anthropic, then OpenAI.
if [ -n "$GEMINI_API_KEY" ]; then
    API_URL="${GEMINI_BASE_URL:-https://generativelanguage.googleapis.com}/v1beta/models/gemini-2.5-flash:generateContent?key=${GEMINI_API_KEY}"
    build_body '{contents: [{parts: [{text: .}]}]}'
    if curl -fsSL "$API_URL" -H "content-type: application/json" -d @"$TMPDIR/body.json" -o "$TMPDIR/result.json"; then
        extract "d['candidates'][0]['content']['parts'][0]['text']"
    else
        fallback
    fi

elif [ -n "$ANTHROPIC_API_KEY" ]; then
    API_URL="${ANTHROPIC_BASE_URL:-https://api.anthropic.com}/v1/messages"
    build_body '{model: "claude-sonnet-4-5-20250514", max_tokens: 1024, messages: [{role: "user", content: .}]}'
    if curl -fsSL "$API_URL" -H "x-api-key: ${ANTHROPIC_API_KEY}" -H "anthropic-version: 2023-06-01" -H "content-type: application/json" -d @"$TMPDIR/body.json" -o "$TMPDIR/result.json"; then
        extract "d['content'][0]['text']"
    else
        fallback
    fi

elif [ -n "$OPENAI_API_KEY" ]; then
    API_URL="${OPENAI_BASE_URL:-https://api.openai.com}/v1/chat/completions"
    build_body '{model: "gpt-4o-mini", messages: [{role: "user", content: .}]}'
    if curl -fsSL "$API_URL" -H "Authorization: Bearer ${OPENAI_API_KEY}" -H "content-type: application/json" -d @"$TMPDIR/body.json" -o "$TMPDIR/result.json"; then
        extract "d['choices'][0]['message']['content']"
    else
        fallback
    fi

else
    fallback
fi
