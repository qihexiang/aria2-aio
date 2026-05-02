# aria2-aio

单可执行文件的 Aria2 多实例管理器。只需该程序 + aria2c 即可通过 Web 界面管理多个 Aria2 下载实例。

## 功能特性

- **多实例管理**：创建、启动、停止、重启、删除多个独立的 Aria2 实例
- **任务监控**：实时查看下载进度、速度、状态（通过 WebSocket 推送）
- **任务操作**：添加下载、暂停、恢复、移除任务
- **历史记录**：自动记录已完成/出错的任务，支持分页查看
- **删除文件**：移除任务或历史记录时可选择同时删除已下载的文件
- **自动恢复**：重启 aria2-aio 后，之前运行中的实例自动恢复
- **单可执行文件**：前端 SPA 通过 go:embed 内嵌在二进制中

## 前置依赖

- **Go 1.22+**（构建后端）
- **Node.js 18+** 与 **npm**（构建前端）
- **aria2c**（运行时依赖，需在 PATH 中或通过配置指定路径）

## 构建步骤

构建分为两步：构建前端 → 编译 Go 二进制。Vite 构建产物直接输出到 `ui/dist/`（Go embed 的嵌入目录），无需手动复制。

### 快速构建（使用 Makefile）

```bash
make all
```

这会依次执行 `make frontend`、`make build`。

### 手动逐步构建

**1. 构建前端**

```bash
cd frontend
npm install
npm run build
cd ..
```

构建产物直接输出到 `ui/dist/`（Vite 的 `outDir` 已配置为 `../ui/dist`），无需额外复制步骤。

**2. 编译 Go 二进制**

```bash
go build -o aria2-aio ./cmd/aria2-aio/
```

也可使用 `CGO_ENABLED=0` 编译纯静态二进制（SQLite 驱动为纯 Go 实现，无需 C 依赖）：

```bash
CGO_ENABLED=0 go build -o aria2-aio ./cmd/aria2-aio/
```

### 交叉编译

```bash
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o aria2-aio-linux-amd64   ./cmd/aria2-aio/
CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -o aria2-aio-linux-arm64   ./cmd/aria2-aio/
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o aria2-aio-darwin-arm64  ./cmd/aria2-aio/
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o aria2-aio-windows-amd64.exe ./cmd/aria2-aio/
```

## 运行

```bash
./aria2-aio
```

默认监听 `0.0.0.0:8080`，数据目录为 `./data`。浏览器访问 `http://localhost:8080` 即可使用。

### 常用命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--data-dir` | `./data` | 数据目录（含数据库、配置、实例数据） |
| `--config` | `<data-dir>/config.yaml` | 配置文件路径 |
| `--host` | `0.0.0.0` | HTTP 服务监听地址 |
| `--port` | `8080` | HTTP 服务监听端口 |
| `--aria2c` | 自动查找 PATH | aria2c 可执行文件路径 |
| `--dev` | `false` | 开发模式：从文件系统读取前端（支持热更新） |
| `--version` | — | 显示版本信息 |

### 配置文件

首次运行时若配置文件不存在，将使用内置默认值。也可参考 `configs/example.yaml` 手动创建：

```yaml
data_dir: "./data"

server:
  host: "0.0.0.0"
  port: 8080

defaults:
  rpc_port_range:
    start: 6801
    end: 6899
  aria2_options:
    max-concurrent-downloads: "5"
    split: "5"
    continue: "true"
    disk-cache: "32M"
  aria2c_path: ""           # 留空则自动查找 PATH 中的 aria2c

task_tracker:
  poll_interval: 2           # 任务轮询间隔（秒）

log:
  level: "info"
  format: "text"
```

## 开发模式

前端开发时可使用 `--dev` 参数，使 Go 后端从本地文件系统读取前端源码而非内嵌版本，配合 Vite 开发服务器实现热更新：

```bash
# 终端 1：启动 Vite 开发服务器
cd frontend && npm run dev

# 终端 2：启动 Go 后端（开发模式）
./aria2-aio --dev --port 8080
```

浏览器访问 Vite 开发服务器（默认 `http://localhost:5173`），其会将 API 请求代理到 Go 后端。

## 项目结构

```
aria2-aio/
├── cmd/aria2-aio/main.go       # 入口：CLI 参数、配置、依赖注入、服务启动
├── internal/
│   ├── config/                  # 配置加载与默认值
│   ├── instance/                # 实例管理：进程监控、生命周期
│   ├── rpc/                     # Aria2 JSON-RPC 客户端与事件监听
│   ├── task/                    # 任务追踪：轮询、事件处理、历史记录
│   ├── store/                   # SQLite 存储层：Schema、实例/任务 CRUD
│   ├── api/                     # REST API 处理器与 WebSocket 端点
│   ├── ws/                      # WebSocket Hub（浏览器客户端广播）
│   └── web/                     # SPA 处理器（embed/文件系统模式）
├── ui/
│   ├── embed.go                 # go:embed 声明（嵌入 ui/dist）
│   └── dist/                    # 前端构建产物（Vite 直接输出到此，Go embed 嵌入目标）
├── frontend/                    # Vue 3 SPA 源码
│   ├── src/
│   │   ├── views/               # 页面组件
│   │   ├── stores/              # Pinia 状态管理
│   │   ├── api/                 # HTTP 与 WebSocket 客户端
│   │   └── types/               # TypeScript 类型定义
│   └── vite.config.ts
├── configs/example.yaml         # 示例配置文件
├── Makefile
├── go.mod / go.sum
└── README.md
```

## REST API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/instances` | 列出所有实例 |
| POST | `/api/v1/instances` | 创建实例 |
| GET | `/api/v1/instances/{id}` | 实例详情 |
| PUT | `/api/v1/instances/{id}` | 更新实例配置 |
| DELETE | `/api/v1/instances/{id}` | 删除实例 |
| POST | `/api/v1/instances/{id}/start` | 启动实例 |
| POST | `/api/v1/instances/{id}/stop` | 停止实例 |
| POST | `/api/v1/instances/{id}/restart` | 重启实例 |
| GET | `/api/v1/instances/{id}/tasks/active` | 活动下载列表 |
| GET | `/api/v1/instances/{id}/tasks/waiting` | 等待下载列表 |
| GET | `/api/v1/instances/{id}/tasks/stopped` | 已停止下载列表 |
| POST | `/api/v1/instances/{id}/tasks` | 添加下载任务 |
| GET | `/api/v1/instances/{id}/tasks/{gid}` | 任务详情 |
| POST | `/api/v1/instances/{id}/tasks/{gid}/pause` | 暂停任务 |
| POST | `/api/v1/instances/{id}/tasks/{gid}/unpause` | 恢复任务 |
| DELETE | `/api/v1/instances/{id}/tasks/{gid}` | 移除任务（`?delete_files=true` 可同时删除文件） |
| GET | `/api/v1/instances/{id}/history` | 任务历史（分页） |
| GET | `/api/v1/instances/{id}/history/{gid}` | 历史记录详情 |
| DELETE | `/api/v1/instances/{id}/history/{gid}` | 删除历史记录（`?delete_files=true` 可同时删除文件） |
| GET | `/api/v1/stats` | 全局统计 |
| GET | `/api/v1/ws` | WebSocket 连接 |

## 清理

```bash
make clean
```

删除编译产物、前端构建产物和嵌入目录。