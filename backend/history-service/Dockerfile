# syntax=docker/dockerfile:1.6
#
# Build context: the backend module root (i.e. /workspace/backend).
#   docker build -f history-service/Dockerfile -t tcm-history-ai/history-service:latest .

# ---- builder stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /src

# Cache module downloads by copying only the manifests first.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the backend module source tree.
COPY . .

# Build a statically-linked binary for the history-service.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /out/history-service \
    ./history-service/cmd/history-service

# ---- runtime stage ----
FROM alpine:3.19

# ca-certificates: required for any outbound TLS (Meilisearch / MinIO / OTLP).
# tzdata: required for Asia/Shanghai timestamps in logs and DB.
# wget: required for the container HEALTHCHECK.
RUN apk add --no-cache ca-certificates tzdata wget && \
    update-ca-certificates && \
    addgroup -S app && adduser -S -G app app

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /out/history-service /app/history-service
# Bundle migrations so any init container or job can run them.
COPY history-service/migrations /app/migrations
# Bundle the dev config as a fallback default.
COPY history-service/internal/conf/config.dev.yaml /app/config.yaml

USER app

EXPOSE 8082

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8082/health || exit 1

ENTRYPOINT ["/app/history-service"]
CMD ["-config", "/app/config.yaml"]
