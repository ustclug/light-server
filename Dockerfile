# syntax=docker/dockerfile:1
ARG GOST_VERSION=3.3.3

FROM golang:1.27-alpine AS plugin-build
WORKDIR /src
COPY plugin/go.mod plugin/go.sum ./
RUN go mod download
COPY plugin/ ./
RUN go build -trimpath -ldflags="-s -w" -o /out/plugin .

FROM alpine:latest AS gost-download
ARG GOST_VERSION
ARG TARGETARCH
RUN apk add --no-cache ca-certificates curl tar \
 && curl -fsSL "https://github.com/go-gost/gost/releases/download/v${GOST_VERSION}/gost_${GOST_VERSION}_linux_${TARGETARCH}.tar.gz" \
    | tar -xz -C /usr/local/bin gost

FROM alpine:latest
RUN apk add --no-cache ca-certificates s6 tzdata netcat-openbsd \
 && mkdir -p /etc/s6/gost /etc/s6/plugin /var/log/gost /opt
COPY --from=plugin-build /out/plugin /usr/local/bin/plugin
COPY --from=gost-download /usr/local/bin/gost /usr/local/bin/gost
COPY etc/s6/ /etc/s6/
COPY etc/healthcheck.sh /etc/healthcheck.sh
ENV TZ=Asia/Shanghai
EXPOSE 29979 29980
VOLUME ["/opt", "/var/log/gost"]
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 CMD ["/etc/healthcheck.sh"]
CMD ["s6-svscan", "/etc/s6"]
