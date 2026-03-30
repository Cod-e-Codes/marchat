#!/bin/sh
set -e

# When running as root (default), fix ownership of app dirs and common volume
# mount points so the marchat user can write SQLite and config, then drop privileges.
if [ "$(id -u)" = 0 ]; then
	mkdir -p /marchat/server /marchat/server/config /marchat/server/db \
		/marchat/server/data /marchat/server/plugins /data
	chown -R marchat:marchat /marchat /data 2>/dev/null || true
	exec su-exec marchat "$0" "$@"
fi

mkdir -p /marchat/server
mkdir -p /marchat/server/config
mkdir -p /marchat/server/db
mkdir -p /marchat/server/data
mkdir -p /marchat/server/plugins

exec ./marchat-server "$@"
