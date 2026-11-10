#!/bin/sh
set -eu
s6-svstat -u /etc/s6/plugin >/dev/null
s6-svstat -u /etc/s6/gost >/dev/null
nc -z 127.0.0.1 8000
nc -z 127.0.0.1 29979
nc -z 127.0.0.1 29980
stamp=/var/log/gost/accounting.last
test -f "$stamp"
now=$(date +%s)
last=$(head -n 1 "$stamp")
case "$last" in
  *[!0-9]*|'') exit 1 ;;
esac
test $((now - last)) -le 3900
