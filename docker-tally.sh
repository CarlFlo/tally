#!/bin/sh
set -eu

if [ "$(id -u)" -eq 0 ] && [ -n "${PUID:-}" ] && [ -n "${PGID:-}" ]; then
	exec su-exec "$PUID:$PGID" /usr/local/libexec/tally "$@"
fi

exec /usr/local/libexec/tally "$@"
