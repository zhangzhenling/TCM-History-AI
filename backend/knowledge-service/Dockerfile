# syntax=docker/dockerfile:1.6
#
# Build context: the backend module root (i.e. /workspace/backend).
#   docker build -f knowledge-service/Dockerfile -t tcm-history-ai/knowledge-service:latest .

# ---- builder stage ----
FROM golang:1.25-alpine AS builder

# go.mod 要求 go 1.25.0；GOTOOLCHAIN=auto 允许在镜像内自动获取所需 toolchain。
ENV GOTOOLCHAIN=auto

WORKDIR /src

# Cache module downloads by copying only the manifests first.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the backend module source tree.
COPY . .

# Build a statically-linked binary for the knowledge-service.
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" \
    -o /out/knowledge-service \
    ./knowledge-service/cmd/knowledge-service

# ---- runtime stage ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata wget && \
    update-ca-certificates && \
    addgroup -S app && adduser -S -G app app

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /out/knowledge-service /app/knowledge-service
COPY knowledge-service/migrations /app/migrations
COPY knowledge-service/internal/conf/config.dev.yaml /app/config.yaml

USER app

EXPOSE 8084

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8084/health || exit 1

ENTRYPOINT ["/app/knowledge-service"]
CMD ["-config", "/app/config.yaml"]
