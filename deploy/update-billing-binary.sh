#!/usr/bin/env bash
# Download the latest local/billing build and swap the binary in place.
#
# Usage:
#   sudo ./update-billing-binary.sh
#
# Environment (all optional):
#   REPO          GitHub repo that runs billing-build.yml   (default: imi6/sub2api)
#   INSTALL_DIR   directory holding sub2api-zzs + resources/ (default: /opt/sub2api-bin)
#   CONTAINER     docker container to restart                (default: sub2api)
#   SERVICE       systemd unit to restart instead of docker  (default: empty)
#   GITHUB_TOKEN  only needed when the repo is private
set -euo pipefail

REPO="${REPO:-imi6/sub2api}"
INSTALL_DIR="${INSTALL_DIR:-/opt/sub2api-bin}"
CONTAINER="${CONTAINER:-sub2api}"
SERVICE="${SERVICE:-}"
TAG="billing-latest"

# Only linux/amd64 is built by billing-build.yml.
case "$(uname -m)" in
  x86_64 | amd64) ;;
  *) echo "only x86_64 builds are published, this host is $(uname -m)" >&2; exit 1 ;;
esac

BIN="sub2api-zzs"
ARCHIVE="${BIN}-linux-amd64.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"
AUTH=()
if [ -n "${GITHUB_TOKEN:-}" ]; then
  AUTH=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "Downloading ${BASE_URL}/${ARCHIVE}"
curl -fsSL "${AUTH[@]}" -o "$TMP/$ARCHIVE" "$BASE_URL/$ARCHIVE"
curl -fsSL "${AUTH[@]}" -o "$TMP/$ARCHIVE.sha256" "$BASE_URL/$ARCHIVE.sha256"
(cd "$TMP" && sha256sum -c "$ARCHIVE.sha256")

mkdir -p "$TMP/pkg" "$INSTALL_DIR"
tar -xzf "$TMP/$ARCHIVE" -C "$TMP/pkg"
chmod +x "$TMP/pkg/$BIN"
echo "New version: $(cat "$TMP/pkg/VERSION")"

# Keep one backup; rename is atomic, a running process keeps the old file.
if [ -f "$INSTALL_DIR/$BIN" ]; then
  cp -p "$INSTALL_DIR/$BIN" "$INSTALL_DIR/$BIN.bak"
fi
cp "$TMP/pkg/$BIN" "$INSTALL_DIR/$BIN.new"
mv -f "$INSTALL_DIR/$BIN.new" "$INSTALL_DIR/$BIN"
cp "$TMP/pkg/VERSION" "$INSTALL_DIR/VERSION"
rm -rf "$INSTALL_DIR/resources.new"
cp -r "$TMP/pkg/resources" "$INSTALL_DIR/resources.new"
rm -rf "$INSTALL_DIR/resources"
mv "$INSTALL_DIR/resources.new" "$INSTALL_DIR/resources"

if [ -n "$SERVICE" ]; then
  systemctl restart "$SERVICE"
  echo "Restarted systemd unit $SERVICE"
elif docker inspect "$CONTAINER" >/dev/null 2>&1; then
  docker restart "$CONTAINER" >/dev/null
  echo "Restarted container $CONTAINER"
else
  echo "Binary updated in $INSTALL_DIR; restart sub2api manually."
fi
echo "Rollback: mv $INSTALL_DIR/$BIN.bak $INSTALL_DIR/$BIN and restart."
