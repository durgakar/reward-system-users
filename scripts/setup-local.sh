#!/usr/bin/env bash
set -euo pipefail

REPO_URL="https://github.com/durgakar/reward-system-users.git"
TARGET="${1:-$HOME/reward-system-users}"

if [ -d "$TARGET/.git" ]; then
  echo "Already cloned at $TARGET — pulling latest..."
  git -C "$TARGET" pull --ff-only
else
  echo "Cloning into $TARGET ..."
  git clone "$REPO_URL" "$TARGET"
fi

echo ""
echo "Done. Open this folder in Cursor:"
echo "  $TARGET"
echo ""
echo "Then run:"
echo "  cd \"$TARGET\""
echo "  go mod tidy"
echo "  make dry-run"
