FROM alpine:3.14

RUN apk update && \
    apk add --no-cache iproute2-tc && \
    rm -rf /var/cache/apk/*

ENTRYPOINT [ "/sbin/tc" ]
