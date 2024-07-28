FROM smartentry/alpine:3.20

LABEL maintainer="Yifan Gao <docker@yfgao.com>"

COPY .docker $ASSETS_DIR
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 CMD /opt/healthcheck.sh

RUN smartentry.sh build
