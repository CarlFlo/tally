#!/bin/sh
set -eu

case "${PUID:-1000}" in
  ''|*[!0-9]*) echo "PUID must be a numeric user id" >&2; exit 64 ;;
esac
case "${PGID:-1000}" in
  ''|*[!0-9]*) echo "PGID must be a numeric group id" >&2; exit 64 ;;
esac

PUID="${PUID:-1000}"
PGID="${PGID:-1000}"

group_name="tally"
user_name="tally"
if ! getent group "$PGID" >/dev/null 2>&1; then
  addgroup -g "$PGID" "$group_name"
else
  group_name="$(getent group "$PGID" | cut -d: -f1)"
fi
if ! getent passwd "$PUID" >/dev/null 2>&1; then
  adduser -D -H -u "$PUID" -G "$group_name" "$user_name"
else
  user_name="$(getent passwd "$PUID" | cut -d: -f1)"
fi

chown "$PUID:$PGID" /config

# Resolve the image's public command before dropping privileges. This keeps
# CMD ["tally"] conventional without relying on su-exec to search PATH.
if [ "$#" -gt 0 ] && [ "$1" = "tally" ]; then
  shift
  set -- /usr/local/libexec/tally "$@"
fi

exec su-exec "$PUID:$PGID" "$@"
