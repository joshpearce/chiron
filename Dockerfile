# syntax=docker/dockerfile:1.7
FROM golang:1.26-alpine AS build
WORKDIR /src/server-go
COPY server-go/go.mod server-go/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY server-go/ ./
ARG TARGETOS TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -buildvcs=false -ldflags='-s -w' -o /out/chiron-server ./cmd/chiron-server

FROM alpine:3.22
ARG VCS_REF=unknown
LABEL org.opencontainers.image.source="https://github.com/mjbraun/chiron" \
      org.opencontainers.image.revision=$VCS_REF
RUN apk add --no-cache ca-certificates chromium poppler-utils tzdata \
    && addgroup -S -g 10001 chiron \
    && adduser -S -D -H -u 10001 -G chiron chiron \
    && mkdir -p /opt/chiron/content /var/lib/chiron /var/cache/chiron /var/tmp/chiron \
    && chown -R chiron:chiron /var/lib/chiron /var/cache/chiron /var/tmp/chiron
COPY --from=build /out/chiron-server /usr/local/bin/chiron-server
COPY assets/fonts/ /opt/chiron/content/fonts/
COPY corpus/ /opt/chiron/content/corpus/
COPY corpus-v2/ /opt/chiron/content/corpus-v2/
COPY deploy/nas/config.yaml /etc/chiron/config.yaml
USER 10001:10001
EXPOSE 8080
VOLUME ["/var/lib/chiron"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:8080/ping || exit 1
ENTRYPOINT ["/usr/local/bin/chiron-server"]
CMD ["-addr", ":8080", "-config", "/etc/chiron/config.yaml"]
