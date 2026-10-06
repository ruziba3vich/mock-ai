FROM golang:1.27-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/mock-ai-api ./cmd/api && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/mock-ai-migrate ./cmd/migrate && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/mock-ai-admin ./cmd/admin && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/mock-ai-ai-keys ./cmd/ai-keys

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S mockai && adduser -S -G mockai mockai

WORKDIR /app
COPY --from=builder /out/mock-ai-api /app/mock-ai-api
COPY --from=builder /out/mock-ai-migrate /app/mock-ai-migrate
COPY --from=builder /out/mock-ai-admin /app/mock-ai-admin
COPY --from=builder /out/mock-ai-ai-keys /app/mock-ai-ai-keys

USER mockai
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=6 \
    CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1

ENTRYPOINT ["/app/mock-ai-api"]
