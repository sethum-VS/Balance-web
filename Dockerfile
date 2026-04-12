# ── Stage 1: Builder ──
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache build-base nodejs npm

WORKDIR /build

# Copy dependency files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

COPY package.json package-lock.json* ./
RUN npm ci

# Copy the rest of the source code
COPY . .

# Generate templ templates
RUN go run github.com/a-h/templ/cmd/templ@latest generate

# Build Tailwind CSS via CLI
RUN npx tailwindcss -i web/src/css/input.css -o web/static/css/styles.css --minify

# Bundle TypeScript with esbuild
RUN go run github.com/evanw/esbuild/cmd/esbuild@latest web/src/ts/main.ts --bundle --outfile=web/static/js/main.js

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/server ./cmd/server


# ── Stage 2: Runner ──
FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Copy static assets (CSS, JS, images)
COPY --from=builder /build/web/static ./web/static

# Expose the application port
EXPOSE 3000

# Use environment variables for configuration
ENV PORT=3000

ENTRYPOINT ["./server"]
