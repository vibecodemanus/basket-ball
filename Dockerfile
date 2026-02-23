# ============================================================
# Stage 1: Build client (TypeScript → minified JS bundle)
# ============================================================
FROM node:22-alpine AS client-builder

WORKDIR /app/client
COPY client/package.json client/package-lock.json* ./
RUN npm ci --ignore-scripts
COPY client/ ./
RUN node esbuild.config.mjs --production

# ============================================================
# Stage 2: Build Go server binary
# ============================================================
FROM golang:1.25-alpine AS server-builder

WORKDIR /app/server
COPY server/go.mod server/go.sum* ./
RUN go mod download
COPY server/ ./

# Static build — no CGO, single binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/game-server ./cmd/server

# ============================================================
# Stage 3: Minimal runtime image
# ============================================================
FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary from server builder
COPY --from=server-builder /app/game-server .

# Copy static assets from client builder
COPY --from=client-builder /app/server/static ./static

ENV PORT=8080
ENV STATIC_DIR=./static

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=3s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1

CMD ["./game-server"]
