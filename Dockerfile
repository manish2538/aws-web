# Stage 1: Build Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* frontend/tsconfig.json frontend/vite.config.ts frontend/index.html ./
COPY frontend/src ./src
RUN npm ci --silent 2>/dev/null || npm install && npm run build

# Stage 2: Build Backend
FROM golang:1.22-alpine AS backend-builder
WORKDIR /app
COPY backend/go.mod backend/go.sum* ./
RUN go mod download 2>/dev/null || true
COPY backend ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# Stage 3: Final Image
FROM alpine:3.20
WORKDIR /app

# Install AWS CLI and ca-certificates
RUN apk add --no-cache ca-certificates aws-cli

# Copy built artifacts
COPY --from=backend-builder /server ./server
COPY --from=frontend-builder /app/frontend/dist ./static
COPY backend/command-config.json ./command-config.json

# Create directory for profile storage
RUN mkdir -p /app/data && chmod 755 /app/data

# Environment variables
ENV PORT=8080
ENV STATIC_DIR=/app/static
ENV COMMAND_CONFIG_PATH=/app/command-config.json
ENV PROFILE_STORE_PATH=/app/data/.aws-local-dashboard-profiles.json

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/profiles || exit 1

# Run the server
CMD ["./server"]
