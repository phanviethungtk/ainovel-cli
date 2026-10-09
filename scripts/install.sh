#!/bin/sh
# Script cài đặt ainovel-cli bằng một lệnh
#
#   curl -fsSL https://raw.githubusercontent.com/phanviethungtk/ainovel-cli/main/scripts/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/phanviethungtk/ainovel-cli/main/scripts/install.sh | sh -s -- v1.2.3
#
# Thư mục cài tuỳ chỉnh: AINOVEL_INSTALL_DIR=~/.local/bin curl -fsSL ... | sh
# Chỉ định phiên bản: AINOVEL_VERSION=v1.2.3 curl -fsSL ... | sh
set -e

REPO="phanviethungtk/ainovel-cli"
BIN="ainovel-cli"
DEST="${AINOVEL_INSTALL_DIR:-/usr/local/bin}"
VERSION="${AINOVEL_VERSION:-${1:-latest}}"

for cmd in curl tar; do
	command -v "$cmd" >/dev/null 2>&1 || { echo "Cần $cmd, vui lòng cài đặt rồi thử lại"; exit 1; }
done

case "$(uname -s)" in
	Darwin) OS="Darwin" ;;
	Linux)  OS="Linux" ;;
	*) echo "Hệ điều hành không được hỗ trợ: $(uname -s); Windows vui lòng tải thủ công tại https://github.com/$REPO/releases"; exit 1 ;;
esac

case "$(uname -m)" in
	x86_64|amd64)  ARCH="x86_64" ;;
	arm64|aarch64) ARCH="arm64" ;;
	*) echo "Kiến trúc không được hỗ trợ: $(uname -m)"; exit 1 ;;
esac

if [ "$VERSION" = "latest" ] || [ -z "$VERSION" ]; then
	API="https://api.github.com/repos/$REPO/releases/latest"
	echo "Đang tra phiên bản mới nhất..."
else
	case "$VERSION" in
		v*) TAG="$VERSION" ;;
		*) TAG="v$VERSION" ;;
	esac
	API="https://api.github.com/repos/$REPO/releases/tags/$TAG"
	echo "Đang tra phiên bản $TAG..."
fi

RELEASE=$(curl -fsSL "$API")
TAG=$(printf '%s\n' "$RELEASE" | grep '"tag_name"' | head -1 | cut -d '"' -f 4)
URL=$(printf '%s\n' "$RELEASE" \
	| grep "browser_download_url" \
	| grep "_${OS}_${ARCH}.tar.gz" \
	| head -1 | cut -d '"' -f 4)
[ -n "$URL" ] || { echo "Không tìm thấy gói cài đặt cho ${OS}_${ARCH}, vui lòng tải thủ công tại https://github.com/$REPO/releases"; exit 1; }
SUMS_URL=$(printf '%s\n' "$RELEASE" \
	| grep "browser_download_url" \
	| grep "_checksums.txt" \
	| head -1 | cut -d '"' -f 4)
[ -n "$SUMS_URL" ] || { echo "Release thiếu checksums.txt, không thể xác minh gói cài đặt, đã huỷ"; exit 1; }

if command -v sha256sum >/dev/null 2>&1; then
	SHA256="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
	SHA256="shasum -a 256"
else
	echo "Cần sha256sum hoặc shasum để xác minh gói cài đặt"; exit 1
fi

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

echo "Đang tải $URL"
curl -fsSL -o "$TMP/pkg.tar.gz" "$URL"
curl -fsSL -o "$TMP/checksums.txt" "$SUMS_URL"

# Xác minh SHA-256: không khớp thì huỷ, tuyệt đối không cài file chưa xác minh
ASSET=$(basename "$URL")
WANT=$(awk -v f="$ASSET" '$2 == f || $2 == "*" f { print $1; exit }' "$TMP/checksums.txt")
[ -n "$WANT" ] || { echo "checksums.txt không có dòng cho $ASSET, đã huỷ"; exit 1; }
GOT=$($SHA256 "$TMP/pkg.tar.gz" | cut -d ' ' -f 1)
[ "$WANT" = "$GOT" ] || { echo "Xác minh thất bại: mong đợi $WANT, nhận $GOT, đã huỷ"; exit 1; }
echo "✓ Xác minh SHA-256 thành công"

tar -xzf "$TMP/pkg.tar.gz" -C "$TMP"

echo "Đang cài vào $DEST"
[ -d "$DEST" ] || mkdir -p "$DEST" 2>/dev/null || sudo mkdir -p "$DEST"
if [ -w "$DEST" ]; then
	mv "$TMP/$BIN" "$DEST/$BIN"
else
	echo "Cần quyền quản trị để ghi vào $DEST"
	sudo mv "$TMP/$BIN" "$DEST/$BIN"
fi
chmod +x "$DEST/$BIN"

# File chạy chưa được ký số, macOS Gatekeeper sẽ chặn lần chạy đầu: gỡ cờ cách ly
[ "$OS" = "Darwin" ] && xattr -d com.apple.quarantine "$DEST/$BIN" 2>/dev/null || true

echo "✓ Cài đặt hoàn tất: $DEST/$BIN"
[ -n "$TAG" ] && echo "Phiên bản: $TAG"
command -v "$BIN" >/dev/null 2>&1 || echo "Lưu ý: $DEST chưa nằm trong PATH, hãy thêm vào PATH"
echo "Chạy $BIN để bắt đầu sử dụng"
