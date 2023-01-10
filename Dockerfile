FROM docker.io/library/alpine:3.16 as runtime

RUN \
  apk add --update --no-cache \
    bash \
    curl \
    ca-certificates \
    tzdata

ENTRYPOINT ["metrics-collector"]
COPY metrics-collector /usr/bin/

USER 65536:0
