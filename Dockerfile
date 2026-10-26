# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git and build essentials if needed
RUN apk add --no-cache git

# Copy dependency definition
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build application binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/server ./cmd/web

# Final stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates and tzdata
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/bin/server /app/server

EXPOSE 3000

CMD ["/app/server"]
