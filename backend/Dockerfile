# Stage 1: Build the binary
FROM golang:1.24.3 AS builder

WORKDIR /app
COPY . .

# Disable CGO to make a fully static binary
ENV CGO_ENABLED=0

# Download dependencies
RUN go mod tidy

# Build the backend binary
RUN go build -o claritty-backend main.go

# Stage 2: Use scratch (minimal) or distroless image
FROM gcr.io/distroless/static

# Copy compiled binary from builder
COPY --from=builder /app/claritty-backend /usr/local/bin/claritty-backend

# Expose port (match your Go server port)
EXPOSE 8088

# Run the binary
ENTRYPOINT ["/usr/local/bin/claritty-backend"]
