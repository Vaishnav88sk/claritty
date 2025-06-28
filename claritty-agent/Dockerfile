FROM golang:1.24.3 AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o claritty-agent main.go

FROM debian:bookworm-slim
COPY --from=builder /app/claritty-agent /usr/local/bin/claritty-agent
ENTRYPOINT ["/usr/local/bin/claritty-agent"]
