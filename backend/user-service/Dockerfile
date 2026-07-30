# syntax=docker/dockerfile:1.6
#
# Build context: the backend module root (i.e. /workspace/backend).
#   docker build -f user-service/Dockerfile -t tcm-history-ai/user-service:latest .

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

# Build a statically-linked binary for the user-service.
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" \
    -o /out/user-service \
    ./user-service/cmd/user-service

# ---- runtime stage ----
FROM alpine:3.19

# ca-certificates: required for any outbound TLS (JWT/OAuth/OTLP).
# tzdata: required for Asia/Shanghai timestamps in logs and DB.
# wget: required for the container HEALTHCHECK.
RUN apk add --no-cache ca-certificates tzdata wget && \
    update-ca-certificates && \
    addgroup -S app && adduser -S -G app app

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /out/user-service /app/user-service
# Bundle migrations so any init container or job can run them.
COPY user-service/migrations /app/migrations
# Bundle the dev config as a fallback default.
COPY user-service/internal/conf/config.dev.yaml /app/config.yaml

USER app

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8081/health || exit 1

ENTRYPOINT ["/app/user-service"]
CMD ["-config", "/app/config.yaml"]
