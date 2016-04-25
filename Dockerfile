FROM centos:7

MAINTAINER Yifan Gao "git@gaoyifan.com"

ENV CACHE_DIR="/etc/docker-squid"

ENV BUILD_SCRIPT="${CACHE_DIR}/build.sh"

COPY assets $CACHE_DIR

COPY entrypoint/entrypoint.sh /sbin/entrypoint.sh

RUN /sbin/entrypoint.sh build

EXPOSE 80/tcp 443/tcp

VOLUME /etc/squid

VOLUME /var/lib/squid

ENTRYPOINT ["/sbin/entrypoint.sh"]

CMD ["/usr/sbin/squid", "-N", "-f", "/etc/squid/squid.conf"]
