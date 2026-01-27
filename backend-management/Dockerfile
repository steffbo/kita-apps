# Frontend build stage
FROM oven/bun:1 AS frontend-builder

WORKDIR /frontend

# Copy openapi specs (needed for type generation)
COPY openapi /openapi

# Copy frontend workspace
COPY frontend/package.json frontend/bun.lock* ./
COPY frontend/apps ./apps
COPY frontend/packages ./packages

# Install dependencies
RUN bun install --frozen-lockfile

# Generate API types
RUN bun run --filter @kita/shared generate:api || true

# Build dienstplan (plan) frontend - skip vue-tsc, use vite directly
ARG VITE_API_URL=/api
ARG VITE_BASE_PATH=/plan
ENV VITE_API_URL=$VITE_API_URL
ENV VITE_BASE_PATH=$VITE_BASE_PATH
RUN cd apps/dienstplan && bun run vite build

# Build zeiterfassung (zeit) frontend
ARG VITE_BASE_PATH_ZEIT=/zeit
ENV VITE_BASE_PATH=$VITE_BASE_PATH_ZEIT
RUN cd apps/zeiterfassung && bun run vite build

# Go build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files from backend-management
COPY backend-management/go.mod backend-management/go.sum ./

# Download dependencies
RUN go mod download

# Copy backend source code
COPY backend-management/ .

# Copy frontend builds to embed directories
COPY --from=frontend-builder /frontend/apps/dienstplan/dist ./internal/frontend/plan
COPY --from=frontend-builder /frontend/apps/zeiterfassung/dist ./internal/frontend/zeit

# Build the application with embedded frontends
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
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# Run the application
CMD ["./server"]
