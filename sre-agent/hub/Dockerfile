FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /claritty-hub .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /claritty-hub /usr/local/bin/claritty-hub
COPY dashboard/ ./dashboard/
EXPOSE 8822
ENTRYPOINT ["/usr/local/bin/claritty-hub"]
