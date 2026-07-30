# syntax=docker/dockerfile:1.6
#
# Build context: the backend module root (i.e. /workspace/backend).
#   docker build -f ai-service/Dockerfile -t tcm-history-ai/ai-service:latest .

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

# Build a statically-linked binary for the ai-service.
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" \
    -o /out/ai-service \
    ./ai-service/cmd/ai-service

# ---- runtime stage ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata wget && \
    update-ca-certificates && \
    addgroup -S app && adduser -S -G app app

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /out/ai-service /app/ai-service
COPY ai-service/migrations /app/migrations
COPY ai-service/internal/conf/config.dev.yaml /app/config.yaml

USER app

EXPOSE 8086

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8086/health || exit 1

ENTRYPOINT ["/app/ai-service"]
CMD ["-config", "/app/config.yaml"]
