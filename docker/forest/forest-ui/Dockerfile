FROM alpine
RUN apk update
RUN apk upgrade
ADD forester_linux_amd64 /forester
RUN chmod +x /forester

ENTRYPOINT [ "/forester" ]
CMD [ "--listen", ":8086", "--api", "http://127.0.0.1:2856" ]
