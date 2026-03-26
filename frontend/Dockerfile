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
# 复制 Nginx 模板
COPY nginx.conf.template /etc/nginx/templates/nginx.conf.template
# 告知 Nginx 启动脚本将模板替换后输出到 /etc/nginx/nginx.conf
ENV NGINX_ENVSUBST_OUTPUT_DIR=/etc/nginx
# 默认后端地址（宿主机 localhost:8080 或容器网桥 backend:8080）
ENV BACKEND_URL=http://mypan-backend:8080/api/

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]