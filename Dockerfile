FROM smartentry/alpine:3.5-0.3.14
RUN sed -i -e 's/dl-.*.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories

MAINTAINER Yifan Gao <docker@yfgao.com>

COPY .docker $ASSETS_DIR

RUN smartentry.sh build
