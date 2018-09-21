#!/bin/bash

HOSTNAME="${COLLECTD_HOSTNAME:-`hostname -s`}"
SQUID_PORT=29979

echo -n "squid,host=$HOSTNAME "
(
    squidclient -p "$SQUID_PORT" cache_object://localhost/counters \
    | awk -F ' = ' \
    '/requests|^(server|client)/ \
    { print $1"="$2 }' \

    squidclient -p "$SQUID_PORT" cache_object://localhost/ipcache \
    | awk -F ':' -v HOSTNAME=$HOSTNAME -v INTERVAL=$INTERVAL \
    '/IPcache (Requests|Hits|Misses)/ \
    { gsub(/ /, "", $1); gsub(/ /, "", $2);
      print $1"="$2 }'
) | paste -sd "," -
echo
