# Multi-stage build: Svelte UI + Go Hub
FROM node:22-alpine AS ui-builder
WORKDIR /app/dashboard
COPY dashboard/package*.json ./
RUN npm ci
COPY dashboard/ ./
RUN npm run build

FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /claritty-hub .

FROM alpine:3.23
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /claritty-hub /usr/local/bin/claritty-hub
COPY --from=ui-builder /app/dashboard/dist/ ./dashboard/dist/
USER 65532:65532
EXPOSE 8822
ENTRYPOINT ["/usr/local/bin/claritty-hub"]
