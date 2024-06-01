FROM alpine:3.16 AS builder
COPY ./squid-preload.c /tmp/squid-preload.c
RUN apk add --no-cache gcc musl-dev && \
    gcc -fPIC -O3 -shared -Wl,-export-dynamic -o /tmp/squid-preload.so /tmp/squid-preload.c -lc

FROM smartentry/alpine:3.16

MAINTAINER Yifan Gao <docker@yfgao.com>

COPY .docker $ASSETS_DIR
COPY --from=builder /tmp/squid-preload.so $ASSETS_DIR/rootfs/usr/lib/squid-preload.so

RUN smartentry.sh build
