# ---- build ----
# Runs on the build machine's own architecture and cross-compiles: DotSlash is
# pure Go (SQLite included), so no emulation is needed for arm64.
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go icons.json ./
COPY web ./web
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
# timetzdata embeds time zone data so TZ works in the empty final image.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -tags timetzdata -ldflags="-s -w -X main.version=${VERSION}" -o /out/dotslash . \
 && mkdir -p /out/data

# ---- run ----
FROM scratch
LABEL org.opencontainers.image.title="DotSlash" \
      org.opencontainers.image.description="A small, fast, self-hosted start page for links, notes, tasks and a directory — for one person or a household" \
      org.opencontainers.image.source="https://github.com/mardix/dotslash" \
      org.opencontainers.image.licenses="MIT"
COPY --from=build /out/dotslash /dotslash
COPY --from=build --chown=65532:65532 /out/data /data
USER 65532:65532
ENV DS_DATA_DIR=/data
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["/dotslash", "healthcheck"]
ENTRYPOINT ["/dotslash"]
