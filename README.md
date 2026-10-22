# 🎬 Video Feed System

> 短视频 Feed 流系统 · 后端手搓，前端 AI 写
>
> 实现了 Feed 流、分片上传断点续传、点赞/关注/评论、三级缓存、游标分页、按视频热度冷热分离、SSE 实时通知、Docker Compose 部署。

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white">
  <img src="https://img.shields.io/badge/Gin-Web-00B46F">
  <img src="https://img.shields.io/badge/MySQL-8-4479A1?logo=mysql&logoColor=white">
  <img src="https://img.shields.io/badge/Redis-7-DC382D?logo=redis&logoColor=white">
  <img src="https://img.shields.io/badge/RabbitMQ-3.13-FF6600?logo=rabbitmq&logoColor=white">
  <img src="https://img.shields.io/badge/Vue-3-4FC08D?logo=vue.js&logoColor=white">
</p>

---

## ✨ 功能概览

| 模块 | 能力 |
|:---:|:---|
| 🔐 **账号** | 注册/登录，JWT 双 Token 自动续期 |
| 📤 **上传** | 分片上传 + MD5 校验 + 断点续传，单片最大 200MB |
| 📱 **Feed 流** | 最新 / 热门 / 关注 / 标签 四维度，三级缓存 + 游标分页 |
| ❤️ **互动** | 点赞 / 评论 / 关注，RabbitMQ 异步落库 |
| 🔔 **通知** | SSE 服务端推送，点赞评论实时到达 |
| 🔥 **热榜** | Redis ZSET 热度加权 + 时间衰减 |
| 🛡️ **缓存** | 三级缓存（本地 → Redis → DB），冷热分离 |
| 🐳 **部署** | Docker 一键拉起全部中间件 |

---

## 🛠 技术栈

| 层 | 技术 | 选型理由 |
|:---:|:---|:---|
| 后端 | **Go + Gin + GORM** | 简洁高效，天然高并发，goroutine + channel 实现"通过通信共享内存" |
| 存储 | **MySQL + Redis** | Redis 缓存热数据，解决每次查磁盘慢的问题 |
| 消息队列 | **RabbitMQ** | 异步处理点赞/评论/关注/通知，不阻塞主业务 |
| 前端 | **Vue 3 + Vite** | 组件化开发，热更新快，深色主题沉浸式 Feed |
| 部署 | **Docker** | 拉镜像起容器，告别本地环境配置地狱 |

---

## 🚀 快速启动

### 1️⃣ 启动中间件

```bash
# MySQL
docker run -d --name feedsystem-mysql -p 3306:3306 -e MYSQL_ROOT_PASSWORD=3295 mysql:8

# Redis
docker run -d --name feedsystem-redis -p 6379:6379 redis:7

# RabbitMQ
docker run -d --name feedsystem-rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3-management
```

### 2️⃣ 启动后端

```bash
# 终端 1：API 服务
cd backend
go run ./cmd

# 终端 2：Worker（必须同时跑，否则点赞/评论不落库）
cd backend
go run ./cmd/worker
```

### 3️⃣ 启动前端

```bash
cd frontend
npm install
npm run dev
```

> 🌐 访问 **http://localhost:5173**

---

## 📁 项目结构

```
feedsystem_video/
├── 📂 backend/
│   ├── 📂 cmd/
│   │   ├── main.go              # 🚀 API 入口
│   │   └── worker/              # ⚙️ 异步 Worker 入口
│   ├── 📂 internal/
│   │   ├── 📂 account/         # 👤 账号：注册/登录
│   │   ├── 📂 auth/             # 🔑 JWT 鉴权
│   │   ├── 📂 config/          # ⚙️ 配置加载
│   │   ├── 📂 db/              # 🗄️ MySQL/Redis 初始化
│   │   ├── 📂 feed/            # 📱 Feed 流聚合
│   │   ├── 📂 http/            # 🛣️ 路由注册
│   │   ├── 📂 message/         # 📨 私信
│   │   ├── 📂 middleware/      # 🛡️ 限流 / 缓存
│   │   ├── 📂 observability/   # 📊 日志
│   │   ├── 📂 social/          # 👥 关注/粉丝
│   │   ├── 📂 video/           # 🎬 视频核心
│   │   │   ├── video_*.go      # CRUD
│   │   │   ├── chunk_*.go      # 分片上传/断点续传
│   │   │   ├── like_*.go       # 点赞
│   │   │   ├── comment_*.go    # 评论
│   │   │   ├── tag_entity.go   # 🏷️ 标签
│   │   │   └── popularity_cache.go  # 🔥 热度缓存
│   │   └── 📂 worker/          # 🔧 Worker 业务
│   └── go.mod
├── 📂 frontend/                 # 🖥️ Vue3 前端
├── 📂 test/                     # 🧪 Postman + 测试脚本
├── 📄 notes.md                  # 📝 API 速查笔记
└── 📄 DESIGN.md                 # 📐 设计文档 + 面试话术
```

---

## 📡 接口清单

<details>
<summary>🔐 账号模块</summary>

| 方法 | 路径 | 说明 |
|:---:|:---|:---|
| POST | `/account/register` | 注册 |
| POST | `/account/login` | 登录 → 返回 JWT |
| POST | `/account/findByID` | 查用户信息 |

</details>

<details>
<summary>🎬 视频模块</summary>

| 方法 | 路径 | 说明 |
|:---:|:---|:---|
| POST | `/video/publish` | 发布视频（含标签） |
| POST | `/video/delete` | 删除视频 |
| POST | `/video/getDetail` | 视频详情 |
| POST | `/video/listByAuthorID` | 按作者查视频 |

</details>

<details>
<summary>📤 分片上传</summary>

| 方法 | 路径 | 说明 |
|:---:|:---|:---|
| POST | `/video/chunk/init` | 初始化上传 |
| POST | `/video/chunk/upload` | 上传单片 |
| POST | `/video/chunk/status` | 查询已传分片 |
| POST | `/video/chunk/complete` | 合并分片 |

</details>

<details>
<summary>📱 Feed 流</summary>

| 方法 | 路径 | 说明 |
|:---:|:---|:---|
| POST | `/feed/listLatest` | 最新视频 |
| POST | `/feed/listByPopularity` | 热门视频 |
| POST | `/feed/listByFollowing` | 关注的人 |
| POST | `/feed/listByTag` | 按标签查 |

</details>

<details>
<summary>❤️ 互动</summary>

| 方法 | 路径 | 说明 |
|:---:|:---|:---|
| POST | `/like/like` | 点赞 |
| POST | `/like/unlike` | 取消点赞 |
| POST | `/like/isLiked` | 查是否点赞 |
| POST | `/comment/publish` | 发评论 |
| POST | `/comment/listAll` | 评论列表 |
| POST | `/comment/delete` | 删评论 |
| POST | `/social/follow` | 关注 |
| POST | `/social/unfollow` | 取关 |
| POST | `/social/getAllVloggers` | 我关注的人 |

</details>

---

## 🐛 踩坑记录

| # | 问题 | 根因 | 修复 |
|:---:|:---|:---|:---|
| 1 | 删了视频首页还显示 | 删 DB 忘清 Redis ZSET | 补 `ZRem feed:global_timeline` |
| 2 | 点别人主页显示自己名字 | `findByID` 从 JWT 取 id | 改成 body 优先，fallback JWT |
| 3 | 点赞不落库 | worker 没启动 | API + worker 两个终端同时跑 |
| 4 | 标签搜不到 | 忘写 video_tags 关联表 | 补标签解析写入逻辑 |
| 5 | 两个视频同时出声 | 切视图旧 observer 没断 | toggleView 先 disconnect |

---

## 🔮 后续规划

- [ ] 🎞️ ffmpeg 视频转码（多码率自适应）
- [ ] 🔍 ES 全文搜索
- [ ] 🔌 WebSocket 替代 SSE
- [ ] ⚖️ 多实例 + Nginx 负载均衡
- [ ] 🚀 Dockerfile + CI/CD 自动化部署

