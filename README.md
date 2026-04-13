# MeloVault

基于网易云音乐 API 的自托管音乐播放器，支持在线播放、搜索、歌单、收藏管理、歌词显示、下载等功能。

## 功能特性

- 在线搜索和播放网易云音乐
- 歌词实时同步显示
- 歌单 / 专辑浏览
- 用户收藏管理
- 音乐下载（支持多音质）
- 用户系统（邮箱注册 / LinuxDo OAuth）
- 管理后台（用户管理、网易云配置等）
- Docker 一键部署
- 响应式 UI，支持移动端

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.22+, `net/http` |
| 前端 | Vue 3, Pinia, Element Plus, Vite |
| 数据库 | PostgreSQL 16 |
| 认证 | JWT + LinuxDo OAuth2 |
| 部署 | Docker, docker-compose |

## 快速开始

### Docker 部署（推荐）

```bash
# 一键部署（交互式配置）
bash deploy.sh
```

或者手动部署：

```bash
# 1. 克隆项目
git clone https://github.com/h0sh1-no/MeloVault.git
cd MeloVault

# 2. 复制并编辑配置
cp .env.example .env
# 编辑 .env，至少设置 DB_PASSWORD 和 JWT_SECRET

# 3. 启动
docker compose up -d
```

### 开发环境

**后端**

```bash
# 配置环境变量
cp .env.example .env
# 编辑 .env 填入配置

# 安装依赖 & 启动
go mod tidy
go run ./cmd/server
```

**前端**

```bash
cd web
npm install
npm run dev
# 访问 http://localhost:3000
```

## 环境变量

参考 [`.env.example`](.env.example) 查看完整配置说明。

| 变量 | 说明 | 默认值 |
|---|---|---|
| `PORT` | 后端监听端口 | `5000` |
| `DB_HOST` | PostgreSQL 主机 | `localhost` |
| `DB_PASSWORD` | 数据库密码 | **必填** |
| `JWT_SECRET` | JWT 签名密钥（`openssl rand -base64 32`） | **必填** |
| `LINUXDO_CLIENT_ID` | LinuxDo OAuth Client ID | 可选 |
| `LINUXDO_CLIENT_SECRET` | LinuxDo OAuth Secret | 可选 |
| `SMTP_HOST` | SMTP 服务器（邮件验证码） | 可选 |
| `FRONTEND_URL` | 前端地址（OAuth 回调） | `http://localhost:5000` |

## Cookie 配置

将网易云音乐的 Cookie 写入项目根目录的 `cookie.txt`，格式为标准 HTTP Cookie 字符串（`key=value; key2=value2`）。

也可以在管理后台中通过网易云二维码登录自动配置。

## 项目结构

```
├── cmd/server/          # 程序入口
├── internal/            # Go 后端核心逻辑
│   ├── auth/            # 认证模块
│   ├── config/          # 配置加载
│   ├── database/        # 数据库连接
│   ├── netease/         # 网易云 API 客户端
│   ├── server/          # HTTP 路由和处理器
│   └── ...
├── web/                 # Vue 3 前端
│   ├── src/
│   │   ├── components/  # 通用组件
│   │   ├── views/       # 页面视图
│   │   ├── stores/      # Pinia 状态管理
│   │   └── api/         # API 封装
│   └── ...
├── deploy.sh            # 一键部署脚本
├── Dockerfile           # 多阶段构建
├── docker-compose.yml   # 编排文件
└── .env.example         # 环境变量示例
```

## 图标版权

- 界面图标：[Element Plus Icons](https://github.com/element-plus/element-plus-icons)（MIT）
- 播放器图标：[Lucide](https://lucide.dev/)（ISC）

## 许可证

[MIT License](LICENSE)
