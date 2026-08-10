#!/usr/bin/env bash
# Regenerate every capture in assets/.
#
# Run from anywhere; it works from the repo root because the tapes refer to
# assets/tapes/bin on PATH and write into assets/.
#
# Needs: vhs (with ttyd + ffmpeg), and gifsicle for the GIF. VHS drives a real
# PTY, so this has to run somewhere with one — on Windows that means WSL, not
# PowerShell.

set -euo pipefail
cd "$(dirname "$0")/../.."

mkdir -p assets/tapes/bin

# VHS records a Linux terminal, so it needs a Linux binary — cross-compiled
# from wherever Go happens to be, since the recording box may not have Go.
if command -v go >/dev/null 2>&1; then
  GOOS=linux GOARCH=amd64 go build -o assets/tapes/bin/justwrite .
  echo "· built assets/tapes/bin/justwrite (linux/amd64)"
fi

if [ ! -x assets/tapes/bin/justwrite ]; then
  echo "no linux binary at assets/tapes/bin/justwrite, and no go to build one" >&2
  echo "build it anywhere with: GOOS=linux GOARCH=amd64 go build -o assets/tapes/bin/justwrite ." >&2
  exit 1
fi

# Exported so `Require` can find it before the tape's own export runs.
export PATH="$PWD/assets/tapes/bin:$PATH"

for tape in assets/tapes/*.tape; do
  echo "· recording $(basename "$tape")"
  vhs "$tape"
done

# Never ship a raw VHS GIF: it writes every frame in full, so a short demo lands
# at 10-20 MB. -O3 is lossless — it rewrites only the pixels that change — and
# usually takes 20-30x off. Wired in here so regenerating never re-bloats.
if command -v gifsicle >/dev/null 2>&1; then
  for gif in assets/*.gif; do
    [ -f "$gif" ] || continue
    before=$(stat -c%s "$gif")
    gifsicle -O3 --batch "$gif"
    after=$(stat -c%s "$gif")
    awk -v b="$before" -v a="$after" -v f="$gif" \
      'BEGIN { printf "· %s: %.1f MB → %.1f MB (lossless -O3)\n", f, b/1048576, a/1048576 }'
  done
else
  echo "· gifsicle not found — the GIF is unoptimized and far bigger than it needs to be" >&2
fi
