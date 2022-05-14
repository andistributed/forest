FROM alpine
RUN apk update
RUN apk upgrade
ADD ./bin/forest_linux_amd64 /forest
RUN chmod +x /forest

ENTRYPOINT [ "/forest" ]
CMD [ "--dsn", "root:root@tcp(127.0.0.1:3306)/forest?charset=utf8", "--admin-password", "root", "--api-address", ":2856" ]
