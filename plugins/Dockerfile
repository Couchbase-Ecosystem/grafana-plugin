from nginx:alpine
RUN mkdir -p /usr/share/nginx/html/plugins
COPY repo /usr/share/nginx/html/plugins/repo/
COPY couchbase-datasource /usr/share/nginx/html/plugins/couchbase-datasource/
COPY nginx.default.conf /etc/nginx/conf.d/default.conf