# 阶段1：构建 Flutter Web
# 注意：如果 cirruslabs 镜像未更新，可使用 ghcr.io/cirruslabs/flutter:stable
FROM ghcr.io/cirruslabs/flutter:3.38.0 AS builder

WORKDIR /app
# 修复权限问题（Flutter 镜像常见问题）
USER root
COPY . .
RUN flutter pub get
RUN flutter build web --release

# 阶段2：Nginx
FROM nginx:alpine
COPY --from=builder /app/build/web /usr/share/nginx/html
# 确保你仓库里有这个 nginx.conf
COPY nginx.conf /etc/nginx/nginx.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]