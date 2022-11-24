FROM docker.io/library/alpine:3.17 as runtime

RUN \
  apk add --update --no-cache \
    bash \
    curl \
    ca-certificates \
    tzdata

ENTRYPOINT ["exoscale-metrics-collector"]
COPY exoscale-metrics-collector /usr/bin/

USER 65536:0
