FROM docker.m.daocloud.io/library/golang:1.22-alpine AS builder

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/app ./cmd/api

# Runtime stage
FROM docker.m.daocloud.io/library/alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 nonroot

COPY --from=builder /out/app /app
COPY --from=builder /src/migrations /migrations
COPY --from=builder /src/seed /seed

USER nonroot

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app", "health"]

EXPOSE 8140

ENTRYPOINT ["/app"]
