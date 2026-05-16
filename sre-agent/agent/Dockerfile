# Multi-stage Dockerfile — produces a ~10MB final image
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /claritty-agent .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /claritty-agent /usr/local/bin/claritty-agent
EXPOSE 9090
ENTRYPOINT ["/usr/local/bin/claritty-agent"]
