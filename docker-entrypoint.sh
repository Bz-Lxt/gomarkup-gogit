#!/bin/sh
set -e
mkdir -p /data/repo
chown -R app:app /data/repo 2>/dev/null || true
if command -v su-exec >/dev/null 2>&1; then
  exec su-exec app /app/gogit
fi
exec /app/gogit
