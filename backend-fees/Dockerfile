# Frontend build stage
FROM oven/bun:1 AS frontend-builder

WORKDIR /frontend

# Copy frontend workspace
COPY frontend/package.json frontend/bun.lock* ./
COPY frontend/apps ./apps
COPY frontend/packages ./packages

# Install dependencies
RUN bun install --frozen-lockfile

# Build beitraege frontend
ARG VITE_API_URL=/api-fees
ARG VITE_BASE_PATH=/beitraege
ENV VITE_API_URL=$VITE_API_URL
ENV VITE_BASE_PATH=$VITE_BASE_PATH
RUN cd apps/beitraege && bun run build

# Go build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files from backend-fees
COPY backend-fees/go.mod backend-fees/go.sum ./

# Download dependencies
RUN go mod download

# Copy backend source code
COPY backend-fees/ .

# Copy frontend build to embed directory
COPY --from=frontend-builder /frontend/apps/beitraege/dist ./internal/frontend/beitraege

# Build the application with embedded frontend
RUN CGO_ENABLED=0 GOOS=linux go build -tags embed_frontend -a -installsuffix cgo -o server ./cmd/server

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy the binary from builder
COPY --from=builder /app/server .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

# Expose port
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

# Run the application
CMD ["./server"]
