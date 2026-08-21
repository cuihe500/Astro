# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY cmd ./cmd
COPY docs ./docs
COPY internal ./internal
COPY pkg ./pkg

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/astro ./cmd/server

FROM alpine:3.22

ARG VCS_REF=unknown
ARG SOURCE_URL=https://github.com/cuihe500/astro

LABEL org.opencontainers.image.title="Astro API" \
      org.opencontainers.image.description="Astro 容器即服务平台 API" \
      org.opencontainers.image.source=$SOURCE_URL \
      org.opencontainers.image.revision=$VCS_REF

RUN apk add --no-cache ca-certificates su-exec tzdata \
    && addgroup -S -g 10001 astro \
    && adduser -S -D -H -u 10001 -G astro astro \
    && mkdir -p /app/configs /run/astro \
    && chown root:astro /run/astro \
    && chmod 0750 /run/astro

WORKDIR /app

COPY --from=build /out/astro /app/astro
COPY configs/config.yaml /app/configs/config.yaml
COPY --chmod=0555 docker/api-entrypoint.sh /usr/local/bin/api-entrypoint

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=5s --start-period=30s --retries=6 \
    CMD wget -q -T 3 -O /dev/null http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/usr/local/bin/api-entrypoint"]
CMD ["/app/astro"]
