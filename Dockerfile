FROM smartentry/alpine:edge-0.3.11

MAINTAINER Yifan Gao <docker@yfgao.com>

COPY . $ASSETS_DIR

RUN smartentry.sh build
