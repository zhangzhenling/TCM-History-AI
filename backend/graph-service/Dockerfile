# syntax=docker/dockerfile:1.6
#
# Build context: the backend module root (i.e. /workspace/backend).
#   docker build -f graph-service/Dockerfile -t tcm-history-ai/graph-service:latest .

# ---- builder stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /src

# Cache module downloads by copying only the manifests first.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the backend module source tree.
COPY . .

# Build a statically-linked binary for the graph-service.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /out/graph-service \
    ./graph-service/cmd/graph-service

# ---- runtime stage ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata wget && \
    update-ca-certificates && \
    addgroup -S app && adduser -S -G app app

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /out/graph-service /app/graph-service
COPY graph-service/migrations /app/migrations
COPY graph-service/internal/conf/config.dev.yaml /app/config.yaml

USER app

EXPOSE 8085

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8085/health || exit 1

ENTRYPOINT ["/app/graph-service"]
CMD ["-config", "/app/config.yaml"]
