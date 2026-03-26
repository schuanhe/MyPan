# 🗂️ MyPan — 个人私有网盘

> 一个基于 Go + Flutter 打造的轻量级、开源个人网盘系统，支持文件管理、多模式分享和 Docker 一键部署。

[![Build & Push](https://github.com/<YOUR_GITHUB_USERNAME>/MyPan/actions/workflows/build-and-push.yml/badge.svg)](https://github.com/<YOUR_GITHUB_USERNAME>/MyPan/actions/workflows/build-and-push.yml)

---

## ✨ 功能特性

- **文件管理**：上传、下载、删除，支持大文件
- **存储卷**：多卷管理，隔离不同用途的文件空间
- **多模式文件分享**：私有 / 公开 / 登录可见 / 密码访问，自定义短链接
- **JWT 鉴权**：安全的用户认证体系
- **速率限制**：内置请求限流，防止滥用
- **单用户注册策略**：专为个人私有部署设计
- **响应式 Web UI**：Flutter Web 构建，磨砂玻璃风格界面

---

## 🏗️ 技术栈

| 模块 | 技术 |
|------|------|
| 后端 | Go 1.23 · Gin · GORM · SQLite · JWT |
| 前端 | Flutter 3 · Go Router · Provider · Dio |
| 容器 | Docker · Nginx Alpine |
| CI/CD | GitHub Actions |

---

## 📁 目录结构

```
MyPan/
├── backend/                # Go 后端
│   ├── controllers/        # 业务控制器（文件、分享、用户等）
│   ├── db/                 # 数据库初始化与连接
│   ├── middlewares/        # JWT 鉴权等中间件
│   ├── models/             # 数据模型（GORM）
│   ├── routes/             # 路由注册
│   ├── utils/              # 工具函数
│   ├── main.go
│   └── Dockerfile
├── frontend/               # Flutter Web 前端
│   ├── lib/                # Dart 应用代码
│   ├── web/                # Web 入口资源
│   ├── nginx.conf          # Nginx 配置（反代 + SPA 路由）
│   └── Dockerfile
├── docker-compose.yml      # 容器编排
├── .env.example            # 环境变量示例
└── .github/
    └── workflows/
        └── build-and-push.yml  # CI/CD 工作流
```

---

## 🚀 快速开始

### 方式一：Docker 部署（推荐）

**前提条件**：已安装 [Docker](https://docs.docker.com/get-docker/) 和 Docker Compose。

**第一步**：创建环境变量文件

```bash
cp .env.example .env
```

编辑 `.env`，填入你的 Docker Hub 用户名：

```env
DOCKERHUB_USERNAME=your_dockerhub_username
PORT=80
```

**第二步**：启动服务

```bash
docker compose up -d
```

**第三步**：访问应用

打开浏览器访问 `http://localhost`（或你配置的端口）。

**停止服务**

```bash
docker compose down
```

**查看日志**

```bash
# 后端日志
docker compose logs -f backend

# 前端日志
docker compose logs -f frontend
```

**更新到最新版本**

```bash
docker compose pull && docker compose up -d
```

---

### 方式二：本地开发运行

**前提条件**：已安装 Go 1.21+、Flutter 3.x、GCC（SQLite CGO 依赖）。

**启动后端**

```bash
cd backend
go run .
# 服务启动在 http://localhost:8080
```

**启动前端（Web 模式）**

```bash
cd frontend
flutter pub get
flutter run -d chrome
# 或构建静态文件：flutter build web
```

---

## ⚙️ GitHub Actions 配置

项目使用 GitHub Actions 自动化构建流程，推送到 `main/master` 分支或创建 `v*` 标签时自动触发。

**需要在仓库 Settings → Secrets → Actions 中配置以下 Secrets：**

| Secret 名称 | 说明 |
|-------------|------|
| `DOCKERHUB_USERNAME` | 你的 Docker Hub 用户名 |
| `DOCKERHUB_TOKEN` | Docker Hub Access Token（在 [Account Settings](https://hub.docker.com/settings/security) 创建） |

**工作流执行内容：**
1. **后端 Job**：Go 编译 Linux amd64 二进制 → 构建 Docker 镜像 → 推送到 `<你的用户名>/mypan-backend`
2. **前端 Job**：`flutter build web --release` → 构建 Nginx Docker 镜像 → 推送到 `<你的用户名>/mypan-frontend`

---

## 🐳 Docker 镜像说明

| 镜像 | 基础镜像 | 约占空间 | 说明 |
|------|---------|---------|------|
| `mypan-backend` | `debian:bookworm-slim` | ~80 MB | 多阶段构建，仅含运行二进制 |
| `mypan-frontend` | `nginx:alpine` | ~15 MB | 静态文件 + Nginx 反代 |

**数据持久化**：后端使用 Docker Volume `mypan_data` 持久化 SQLite 数据库和上传文件，容器删除重建后数据不丢失。

---

## 🔧 配置说明

### 后端环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `GIN_MODE` | `release` | Gin 运行模式（debug/release） |

> 数据库文件和上传文件默认存储在 `/data` 目录（容器内），通过 Volume 映射到宿主机。

### 端口说明

| 服务 | 容器内端口 | 宿主机端口 |
|------|-----------|-----------|
| 前端（Nginx） | 80 | `${PORT:-80}`（可自定义） |
| 后端（Go API） | 8080 | 不对外暴露，仅内网 |

---

## 📝 License

MIT License — 自由使用、修改和分发。
