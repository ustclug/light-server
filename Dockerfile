FROM centos:7

MAINTAINER Yifan Gao "git@gaoyifan.com"

COPY assets /etc/docker-assets

COPY entrypoint/entrypoint.sh /sbin/entrypoint.sh

RUN /sbin/entrypoint.sh build

ENTRYPOINT ["/sbin/entrypoint.sh"]

CMD ["/usr/sbin/squid", "-N", "-f", "/etc/squid/squid.conf"]
