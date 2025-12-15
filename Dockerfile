# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies for CGO (needed for HEIC image processing)
RUN apk add --no-cache gcc g++ musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o server cmd/server/main.go

# Final stage
FROM alpine:latest

# Install C++ runtime libraries required for CGO binaries
RUN apk --no-cache add ca-certificates libgcc libstdc++

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/server .

# Copy Firebase credentials (you might want to mount this instead)
# COPY firebase-service-account.json .

EXPOSE 8080

CMD ["./server"]
