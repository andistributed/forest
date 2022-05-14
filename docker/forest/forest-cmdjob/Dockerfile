FROM alpine
RUN apk update
RUN apk upgrade
ADD forestcmdjob_linux_amd64 /forestcmdjob
RUN chmod +x /forestcmdjob

ENTRYPOINT [ "/forestcmdjob" ]
