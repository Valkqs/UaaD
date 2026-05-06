# UAAD 全栈工程本地运行与联调指南

> **最后更新：** 2026-05-02  
> **关联文档：** [SPRINT3.md](./SPRINT/SPRINT3.md)、[Prometheus_And_Grafna.md](./Prometheus_And_Grafna.md)、[ST_BASELINE.md](./STRESS_TEST/ST_BASELINE.md)

---

## 1. 基础依赖预检查

请确保本机已安装以下工具：

| 工具 | 最低版本 | 用途 | 安装参考 |
|------|---------|------|---------|
| **Docker & Docker Compose** | Docker 24+ / Compose v2 | 拉起 MySQL / Redis / Kafka / Prometheus / Grafana | [docs.docker.com](https://docs.docker.com/get-docker/) |
| **Go** | 1.20+ | 后端编译与运行 | `brew install go` 或 [go.dev/dl](https://go.dev/dl/) |
| **Node.js** | 18+ | 前端运行时 | `brew install node` 或 [nodejs.org](https://nodejs.org/) |
| **pnpm** | 8+ | 前端包管理 | `npm install -g pnpm` |
| **JMeter** | 5.6+ | 压力测试（仅压测需要） | `brew install jmeter` 或 [jmeter.apache.org](https://jmeter.apache.org/) |

**快速验证：**

```bash
docker --version && docker compose version
go version
node --version && pnpm --version
```

---

## 2. 运行态说明

项目支持三种运行态，按需拉起对应服务：

### 2.1 三种运行态概览

| 运行态 | Docker 服务 | 后端 | 前端 | 适用场景 |
|--------|-------------|------|------|---------|
| **开发态** | MySQL + Redis + Kafka | 启动 | 可选 | 后端功能开发与调试、单元测试 |
| **联调态** | MySQL + Redis + Kafka | 启动 | 启动（Mock 关闭） | 前后端联调、闭环测试验证 |
| **演示态 / 压测态** | MySQL + Redis + Kafka + Prometheus + Grafana | 启动 | 启动（Mock 关闭） | 完整监控观测、压力测试、答辩演示 |

### 2.2 各态配置要求

#### 开发态

- 仅拉起核心中间件，无需监控栈
- 前端可选启动；若仅调试后端 API，可使用 curl / Postman
- Mock 可按需启用（`VITE_USE_MOCK=true`），但建议联调时关闭

```bash
docker compose up -d mysql redis kafka
cd backend && go run ./cmd/server
```

#### 联调态

- 中间件与开发态相同
- **前端必须关闭 Mock**：确认 `frontend/.env` 中 `VITE_USE_MOCK=false`（或不设置，默认为 `false`）
- 前后端同时启动，前端通过 `http://localhost:8080` 访问后端 API

```bash
docker compose up -d mysql redis kafka
cd backend && go run ./cmd/server     # 终端 1
cd frontend && pnpm install && pnpm dev  # 终端 2
```

#### 演示态 / 压测态

- **拉起全部 5 个 Docker 服务**（含 Prometheus + Grafana）
- Grafana Dashboard 通过 provisioning 自动加载，无需手动配置
- 执行压测前务必先完成 Seed 数据导入

```bash
docker compose up -d                  # 全部 5 个服务
cd backend && go run ./cmd/server     # 终端 1
cd backend && go run ./scripts/seed   # 终端 1（仅首次或清库后）
cd frontend && pnpm install && pnpm dev  # 终端 2
```

---

## 3. 拉起基础设施容器

### 3.1 一键启动

```bash
# 项目根目录下，拉起全部服务
docker compose up -d

# 验证容器状态（mysql / redis 应为 healthy）
docker compose ps
```

### 3.2 各服务详情

| 服务 | 镜像 | 容器名 | 端口 | 说明 |
|------|------|--------|------|------|
| **MySQL** | `mysql:8.0` | `uaad-mysql` | `3306` | 事务隔离级别 `READ-COMMITTED`，数据卷 `mysql_data` |
| **Redis** | `redis:7-alpine` | `uaad-redis` | `6379` | 数据卷 `redis_data` |
| **Kafka** | `apache/kafka:3.7.0` | `uaad-kafka` | `9092` | KRaft 单节点模式（无需 Zookeeper），数据卷 `kafka_data` |
| **Prometheus** | `prom/prometheus:v2.51.0` | `uaad-prometheus` | `9090` | 从宿主机 `:8080/metrics` 拉取，`scrape_interval=15s` |
| **Grafana** | `grafana/grafana:10.4.0` | `uaad-grafana` | `3000` | 默认账号 `admin` / `admin`，Dashboard 自动加载 |

### 3.3 连接参数

| 参数 | 值 | 环境变量 |
|------|------|---------|
| MySQL 地址 | `localhost:3306` | `DB_HOST` / `DB_PORT` |
| MySQL 用户 / 密码 | `root` / `root` | `DB_USER` / `DB_PASSWORD` |
| MySQL 数据库名 | `uaad` | `DB_NAME` |
| Redis 地址 | `localhost:6379` | `REDIS_HOST` / `REDIS_PORT` |
| Kafka Broker | `localhost:9092` | `KAFKA_BROKER` |

### 3.4 数据清理

```bash
# 彻底销毁所有容器与数据卷（库表、缓存、队列全部清空）
docker compose down -v

# 重新拉起并导入 Seed 数据
docker compose up -d
cd backend && go run ./scripts/seed
```

---

## 4. 启动 Go 后端服务

### 4.1 环境配置

后端配置通过环境变量加载（`backend/internal/config/config.go`），所有参数均有开发友好的默认值。可选创建 `.env` 文件覆盖：

```bash
cd backend
cp .env.example .env  # 可选，默认值即可本地运行
```

**关键环境变量：**

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `APP_ENV` | `development` | 运行环境（`development` / `production`） |
| `PORT` | `8080` | 后端监听端口 |
| `JWT_SECRET` | `uaad-super-secret-key-2026` | JWT 签名密钥 |
| `DB_HOST` / `DB_PORT` | `localhost` / `3306` | MySQL 连接 |
| `DB_USER` / `DB_PASSWORD` | `root` / `root` | MySQL 认证 |
| `DB_NAME` | `uaad` | MySQL 数据库名 |
| `DB_MAX_OPEN_CONNS` | `100` | MySQL 最大连接数 |
| `REDIS_HOST` / `REDIS_PORT` | `localhost` / `6379` | Redis 连接 |
| `KAFKA_BROKER` | `localhost:9092` | Kafka Broker 地址 |
| `KAFKA_TOPIC_ENROLLMENT` | `enrollment_requests` | Kafka 报名队列 Topic |
| `REG_RATE_LIMIT_PER_MIN` | `5` | 注册接口限流（每分钟） |
| `CORS_ALLOWED_ORIGINS` | 空 | 逗号分隔的 CORS 白名单。开发态留空=放开；生产态留空=默认拒绝跨域 |

生产部署建议显式配置 `APP_ENV=production` 与 `CORS_ALLOWED_ORIGINS`，避免跨域策略误放开。

### 4.2 编译与启动

```bash
cd backend
go mod tidy
go run ./cmd/server
```

启动成功标志：终端输出 `Server starting on :8080`。
停止服务时使用 `Ctrl+C`（或发送 `SIGTERM`）可触发优雅关闭：HTTP 先停止接入，再依次释放 Kafka / Redis / MySQL 连接。

**健康检查：**

```bash
curl http://localhost:8080/api/v1/health
# 预期：HTTP 200
```

**请求追踪（Request ID）：**

- 后端会为每个请求返回 `X-Request-ID` 响应头。
- 若客户端已传入 `X-Request-ID`，后端会沿用该值，便于前后端与网关日志串联排障。

### 4.3 Seed 数据导入

**首次启动或清库后必须执行**（后端会自动 AutoMigrate 建表，但需要 Seed 导入初始数据）：

```bash
cd backend
go run ./scripts/seed
```

Seed 内容：

| 数据 | 数量 | 说明 |
|------|------|------|
| 用户 | 5 个 | 含 3 个 `CUSTOMER`（`13800000001`~`13800000003`） + 1 个 `MERCHANT`（`13800000004`） + 1 个 `ADMIN` |
| 活动 | 20 个 | 多种状态（DRAFT / PUBLISHED），含不同容量 |

**测试账号：**

| 角色 | 手机号 | 密码 | 用途 |
|------|--------|------|------|
| CUSTOMER | `13800000001` | `test123456` | C 端报名/浏览 |
| CUSTOMER | `13800000002` | `test123456` | 并发测试备用 |
| MERCHANT | `13800000004` | `test123456` | 商户活动管理 |

---

## 5. 启动前端工程

### 5.1 环境配置

```bash
cd frontend
cp .env.example .env  # 可选
```

**关键环境变量：**

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `VITE_USE_MOCK` | `false` | `true` 启用 MSW Mock（仅离线 UI 调试）；**联调/演示时必须为 `false`** |

### 5.2 安装依赖与启动

```bash
cd frontend
pnpm install
pnpm dev
```

启动成功标志：终端输出 `Local: http://localhost:5173`。

### 5.3 Mock 策略

| 场景 | `VITE_USE_MOCK` | 说明 |
|------|-----------------|------|
| **本地 UI 开发**（无后端） | `true` | MSW 拦截请求，返回 Mock 数据 |
| **联调 / 验收 / 演示** | `false`（默认） | 所有请求指向真实后端 `http://localhost:8080` |

> 联调与验收阶段**必须关闭 Mock**。若因 Mock 误用导致验收结果失真，结果不予采纳。

### 5.4 常用命令

```bash
pnpm dev        # 开发服务器
pnpm build      # 生产构建（tsc + vite build）
pnpm preview    # 预览生产构建产物
pnpm lint       # ESLint 检查
pnpm test       # Vitest 单元测试
```

---

## 6. Prometheus 与 Grafana（监控栈）

### 6.1 启动

随 `docker compose up -d` 一起启动，无需额外操作。

| 服务 | 地址 | 默认账号 |
|------|------|---------|
| Prometheus | `http://localhost:9090` | 无需登录 |
| Grafana | `http://localhost:3000` | `admin` / `admin` |

### 6.2 数据源

**已通过 provisioning 自动配置**（`infra/grafana/provisioning/datasources/prometheus.yaml`）：

- 数据源 UID：`prometheus`
- URL：`http://prometheus:9090`（Docker 内网服务名）
- `timeInterval: 15s`，与 Prometheus `scrape_interval` 对齐

无需手动添加数据源。

### 6.3 Dashboard

**已通过 provisioning 自动加载**（`infra/grafana/provisioning/dashboards/uaad-sprint3.json`）：

- Dashboard 名称：**UAAD / Sprint3 / Enrollment & Worker**
- UID：`uaad-sprint3`
- 直达链接：`http://localhost:3000/d/uaad-sprint3`

**面板布局（5 行 12 面板）：**

| Row | 面板 | 观测要点 |
|-----|------|---------|
| **Overview Stats** | 总请求量、5xx 错误数、平均延迟、Worker 消息数 | 即使无流量也显示 0（非 N/A） |
| **HTTP 吞吐** | 全站 + 按路径 RPS、HTTP 状态码分布 | 压测时与 JMeter Ramp 同步观察 |
| **HTTP 时延** | 全站 P50/P95/P99、按 Path 的 P95 | 压测时关注尾延迟 |
| **Worker & Kafka** | Worker success/failure 吞吐、Worker 耗时 P95 + Kafka Lag | 异步链路健康度 |
| **Go Runtime** | Goroutines、内存（RSS + Heap）、GC Duration | 长期运行稳定性 |

### 6.4 后端指标端点

后端暴露 `/metrics`（`http://localhost:8080/metrics`），Prometheus 每 15 秒自动拉取。

```bash
# 验证指标端点
curl -s http://localhost:8080/metrics | head -20

# 查看 UAAD 自定义指标
curl -s http://localhost:8080/metrics | grep -E "^(http_requests|worker_)"
```

详细指标说明见 [Prometheus_And_Grafna.md](./Prometheus_And_Grafna.md)。

### 6.5 Prometheus 抓取配置

位于 `infra/prometheus.yml`，当前配置：

- Job：`uaad-backend`
- Target：`host.docker.internal:8080`（Docker 宿主机回环）
- 路径：`/metrics`
- 抓取间隔：`15s`

---

## 7. 压力测试

### 7.1 前置条件

| 检查项 | 验证命令 | 预期 |
|--------|----------|------|
| Docker 全栈就绪 | `docker compose ps` | mysql / redis 为 `healthy` |
| 后端已启动 | `curl http://localhost:8080/api/v1/health` | HTTP 200 |
| Seed 已导入 | MySQL 中存在用户和活动 | 有 CUSTOMER / MERCHANT 账号 |
| JMeter 已安装 | `jmeter --version` | 5.6+ |

### 7.2 一键执行

```bash
cd backend/tests/jmeter
bash run-jmeter-report.sh
```

脚本自动完成：商户登录 → 创建活动 → 发布（Redis 库存预热）→ 生成用户 CSV → 运行 JMeter → 输出 HTML 报告。

### 7.3 切换并发规模

编辑 `enrollment-load.jmx`，启用目标线程组并禁用其余（同时只启用一个）：

| 线程组 | 线程数 | Ramp-up | 默认状态 |
|--------|--------|---------|---------|
| 峰值 1000 并发 | 1000 | 30s | **启用** |
| Sprint3 大规模 3000 并发 | 3000 | 60s | 禁用 |
| Sprint3 冲刺目标 5000 并发 | 5000 | 90s | 禁用 |

生成 CSV 时 `-count` 须与线程数一致：

```bash
cd backend
go run ./scripts/gen_jmeter_data -count 3000  # 3000 并发时
```

> 5000 并发因 macOS 线程限制需双进程方案，详见 [ST_BASELINE.md §4.5](./STRESS_TEST/ST_BASELINE.md)。

### 7.4 压测报告

- 基线与执行步骤：[ST_BASELINE.md](./STRESS_TEST/ST_BASELINE.md)
- 完整测试报告：[ST_REPORT.md](./STRESS_TEST/ST_REPORT.md)
- 闭环验收清单：[SPRINT3_CHECKLIST.md](./STRESS_TEST/SPRINT3_CHECKLIST.md)

---

## 8. 完整启动流程速查

### 8.1 从零拉起（首次 / 清库后）

```bash
# 1. 拉起 Docker 全栈
docker compose up -d
docker compose ps  # 确认 healthy

# 2. 启动后端
cd backend
go mod tidy
go run ./cmd/server

# 3. 导入 Seed 数据（另开终端）
cd backend
go run ./scripts/seed

# 4. 启动前端（另开终端）
cd frontend
pnpm install
pnpm dev

# 5. 验证
# 浏览器访问 http://localhost:5173 → 首页正常加载
# curl http://localhost:8080/api/v1/health → HTTP 200
# Grafana: http://localhost:3000 → admin/admin → Dashboard 已存在
```

### 8.2 日常开发（已有数据卷）

```bash
docker compose up -d
cd backend && go run ./cmd/server     # 终端 1
cd frontend && pnpm dev               # 终端 2（联调时）
```

---

## 9. 常见问题排查

| 问题 | 排查步骤 |
|------|---------|
| **后端启动报错（端口占用）** | `lsof -i :8080` 查找占用进程并 `kill`；或修改 `.env` 中 `PORT` |
| **前端请求 401 / CORS** | 确认已登录并获取有效 token；验证 `frontend/src/api/axios.ts` 中 BaseURL 指向 `localhost:8080` |
| **MySQL 连接失败** | 确认 Docker 容器 healthy：`docker compose ps`；检查端口 3306 未被其他 MySQL 占用 |
| **Kafka 消费无反应** | 查看后端日志是否有 `kafka connect` 相关报错；`docker logs uaad-kafka` 查看 Kafka 启动日志 |
| **Prometheus 无数据** | 确认后端已启动：`curl http://localhost:8080/metrics`；检查 `docker logs uaad-prometheus` |
| **Grafana Dashboard 为空** | 验证 Prometheus 数据源连通：Grafana → Data sources → Prometheus → Save & test |
| **JMeter 线程数不足** | macOS 限制 `kern.num_taskthreads=4096`；5000 并发需用双进程方案，见 [ST_BASELINE.md §4.5](./STRESS_TEST/ST_BASELINE.md) |
| **Seed 报错（用户已存在）** | Seed 为幂等操作，重复执行不会报错；若数据异常可 `docker compose down -v` 重建 |
| **前端数据与后端不符** | 确认 `VITE_USE_MOCK=false`；DevTools Network 检查请求是否指向 `localhost:8080`，无 `[MSW]` 标记 |
| **注册限流（gen_jmeter_data 慢）** | 临时设置 `REG_RATE_LIMIT_PER_MIN=120` 和 `REG_RATE_LIMIT_BURST=40`，生成完恢复 |
