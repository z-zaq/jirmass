# Build Stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy dependency manifests
COPY go.mod ./
RUN go mod download

# Copy source code
COPY main.go bot.go ./

# Compile static binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o jirmass .

# Runtime Stage
FROM alpine:3.20

WORKDIR /app

# Add ca-certificates for outbound HTTPS (e.g. Gemini LLM API, Webhooks)
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy compiled binary and web assets
COPY --from=builder /app/jirmass /app/jirmass
COPY templates/ /app/templates/
COPY static/ /app/static/

RUN chown -R appuser:appgroup /app
USER appuser

ENV PORT=8081 \
    BACKEND_URL=http://backend:8000

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT:-8081}/healthz || exit 1

ENTRYPOINT ["/app/jirmass"]
