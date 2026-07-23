#!/bin/sh
set -e

TYPE="${NEXTUNNEL_TYPE:-server}"

case "${TYPE}" in
	client)
		BIN=/app/bin/nextunnel-client
		;;
	server)
		BIN=/app/bin/nextunnel-server
		;;
	*)
		echo "Invalid NEXTUNNEL_TYPE: '${TYPE}' (expected 'client' or 'server')" >&2
		exit 1
		;;
esac

if [ "$#" -eq 0 ]; then
	set -- --config "/app/conf/nextunnel-${TYPE}.toml"
fi

exec "${BIN}" "$@"
