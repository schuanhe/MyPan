# ─── 阶段1：编译 ────────────────────────────────────────────
FROM golang:1.23-bookworm AS builder

WORKDIR /app

# 复制依赖文件并下载（利用 Docker 层缓存）
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并编译
COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o mypan-server .

# ─── 阶段2：运行时镜像 ───────────────────────────────────────
# 使用 debian-slim（CGO/SQLite 需要 libc），比完整镜像小 ~95%
FROM debian:bookworm-slim

# 安装 ca-certificates（HTTPS 请求所需）
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# 从构建阶段只拷贝二进制
COPY --from=builder /app/mypan-server .

# 数据目录（SQLite 数据库 + 文件存储）
VOLUME ["/data"]

EXPOSE 8080

ENV GIN_MODE=release

CMD ["./mypan-server"]
