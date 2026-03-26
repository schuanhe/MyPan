# ─── 阶段1：构建 Flutter Web ────────────────────────────────
FROM ghcr.io/cirruslabs/flutter:3.41.5 AS builder

WORKDIR /app

# 先复制依赖文件（利用缓存）
COPY pubspec.yaml pubspec.lock ./
RUN flutter pub get

# 复制全部源码并构建 Web Release 版
COPY . .
RUN flutter build web --release

# ─── 阶段2：Nginx 静态文件服务 ──────────────────────────────
# nginx:alpine 约 8MB，极致轻量
FROM nginx:alpine

# 拷贝构建产物
COPY --from=builder /app/build/web /usr/share/nginx/html

# 拷贝自定义 nginx 配置（含反代后端 + SPA 路由）
COPY nginx.conf /etc/nginx/nginx.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
