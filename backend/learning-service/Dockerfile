# syntax=docker/dockerfile:1.6
#
# Build context: the backend module root (i.e. /workspace/backend).
#   docker build -f learning-service/Dockerfile -t tcm-history-ai/learning-service:latest .

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

# Build a statically-linked binary for the learning-service.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /out/learning-service \
    ./learning-service/cmd/learning-service

# ---- runtime stage ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata wget && \
    update-ca-certificates && \
    addgroup -S app && adduser -S -G app app

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /out/learning-service /app/learning-service
COPY learning-service/migrations /app/migrations
COPY learning-service/internal/conf/config.dev.yaml /app/config.yaml

USER app

EXPOSE 8087

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8087/health || exit 1

ENTRYPOINT ["/app/learning-service"]
CMD ["-config", "/app/config.yaml"]
