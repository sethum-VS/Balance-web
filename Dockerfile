# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache nodejs npm make git

WORKDIR /app

# Install Go tools
RUN go install github.com/a-h/templ/cmd/templ@latest

# Copy dependency files first for caching
COPY go.mod go.sum ./
RUN go mod download

COPY package.json package-lock.json ./
RUN npm install

# Copy source
COPY . .

# Build
RUN templ generate
RUN npm run tailwind
RUN npm run esbuild
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/server ./cmd/server

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/bin/server .
COPY --from=builder /app/web/static ./web/static

EXPOSE 3000

CMD ["./server"]
