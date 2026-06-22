#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
rm -f novel-dialogue-card.zip
zip -r novel-dialogue-card.zip novel-dialogue-card >/dev/null
echo "created examples/novel-dialogue-card.zip"
