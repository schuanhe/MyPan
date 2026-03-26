# 阶段1：编译
FROM golang:1.24-bookworm AS builder

# CGO 需要 gcc
RUN apt-get update && apt-get install -y gcc libc6-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# 开启 CGO 以支持 SQLite
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o mypan-server .

# 阶段2：运行
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/mypan-server .

VOLUME ["/data"]
EXPOSE 8080
ENV GIN_MODE=release
CMD ["./mypan-server"]