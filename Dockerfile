FROM gaoyifan/openresty:light-fat
RUN apk add --no-cache bash tar
ADD https://raw.githubusercontent.com/gaoyifan/smartentry/v0.4.2/smartentry.sh /sbin/smartentry.sh
RUN chmod +x /sbin/smartentry.sh
ENV ASSETS_DIR="/opt/smartentry/HEAD"
ENTRYPOINT ["/sbin/smartentry.sh"]

RUN luarocks install lua-resty-redis-connector
RUN apk add --no-cache openssl
COPY .docker $ASSETS_DIR
COPY nginx.conf $ASSETS_DIR/rootfs/usr/local/openresty/nginx/conf/
WORKDIR /usr/local/openresty/nginx/conf
RUN rm nginx.conf
RUN mkdir -p ssl/
WORKDIR ssl/
RUN openssl req -new -newkey rsa:2048 -days 3650 -nodes -x509 -subj '/CN=light-accelerator-default-certificate' -keyout server.key -out server.crt
COPY lua/ /usr/local/openresty/nginx/lua/
