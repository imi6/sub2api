#!/usr/bin/env bash
# Download the latest local/billing build (sub2api-zzs) and swap it in place.
#
# Usage:
#   sudo INSTALL_DIR=/www/wwwroot/sub.zzshu.cc bash update-billing-binary.sh
#
# Environment (all optional):
#   REPO          GitHub repo that runs billing-build.yml  (default: imi6/sub2api)
#   INSTALL_DIR   directory holding sub2api-zzs            (default: current directory)
#   CONTAINER     docker container to restart              (default: sub2api)
#   GITHUB_TOKEN  only needed when the repo is private
set -euo pipefail

REPO="${REPO:-imi6/sub2api}"
INSTALL_DIR="${INSTALL_DIR:-$(pwd)}"
CONTAINER="${CONTAINER:-sub2api}"
BIN="sub2api-zzs"
BASE_URL="https://github.com/${REPO}/releases/download/billing-latest"

# Only linux/amd64 is built by billing-build.yml.
case "$(uname -m)" in
  x86_64 | amd64) ;;
  *) echo "only x86_64 builds are published, this host is $(uname -m)" >&2; exit 1 ;;
esac

AUTH=()
if [ -n "${GITHUB_TOKEN:-}" ]; then
  AUTH=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "Downloading ${BASE_URL}/${BIN}"
curl -fsSL "${AUTH[@]}" -o "$TMP/$BIN" "$BASE_URL/$BIN"
curl -fsSL "${AUTH[@]}" -o "$TMP/$BIN.sha256" "$BASE_URL/$BIN.sha256"
(cd "$TMP" && sha256sum -c "$BIN.sha256")
chmod +x "$TMP/$BIN"

mkdir -p "$INSTALL_DIR"
if [ -f "$INSTALL_DIR/$BIN" ]; then
  cp -p "$INSTALL_DIR/$BIN" "$INSTALL_DIR/$BIN.bak"
fi
# Write beside the target then rename: never overwrite a running binary in place.
cp "$TMP/$BIN" "$INSTALL_DIR/$BIN.new"
mv -f "$INSTALL_DIR/$BIN.new" "$INSTALL_DIR/$BIN"
echo "Installed $INSTALL_DIR/$BIN"

if docker inspect "$CONTAINER" >/dev/null 2>&1; then
  docker restart "$CONTAINER" >/dev/null
  echo "Restarted container $CONTAINER"
else
  echo "Container $CONTAINER not found; restart it manually."
fi
echo "Rollback: mv $INSTALL_DIR/$BIN.bak $INSTALL_DIR/$BIN && docker restart $CONTAINER"
