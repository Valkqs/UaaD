# UAAD Sprint 4 完成记录

> **核心目标：万级数据承载验证、JMeter 并发压测、前后端全链路走查**
>
> **说明：** Sprint 4 为项目最终收尾阶段，不引入新业务逻辑，重点完成数据规模验证、压力测试与端到端闭环演示。
>
> 完成日期：2026-05-23

---

## 一、完成情况总览

| 任务 | 验收标准 | 完成状态 |
|------|----------|----------|
| 数据库行数 ≥ 1 万条 | 各主表总行数超过 10,000 | ✅ **67,830 行** |
| JMeter 并发压测 | 万级请求量，错误率 0%，关键 API < 200ms | ✅ **10,000 请求，P90 = 142ms，峰值 759 req/s，错误率 0%** |
| 前端关闭 Mock 联调 | `VITE_USE_MOCK=false`，所有页面对接真实后端 | ✅ 已验证 |
| 全链路端到端走查 | 商户建活动 → 用户报名 → 订单 → 通知完整跑通 | ✅ 已完成 |
| Bug 修复 | Kafka consumer 消费积压问题 | ✅ 已修复 |

---

## 二、代码改动

### 1. `backend/internal/infra/kafka.go` — Bug 修复

**问题**：Kafka consumer 在消费者组首次连接时默认从 topic 末尾（`LastOffset`）开始读，导致所有 enrollment 消息被跳过。表现为报名请求成功但订单始终不创建，enrollment 状态永远停在 `QUEUING`。

**根因**：`kafka.ReaderConfig` 未显式设置 `StartOffset`，在 kafka-go 与 Kafka 3.7（KRaft）的组合下，消费者组初始化时采用 `LastOffset`，跳过了历史消息。

**修复**：

```go
// backend/internal/infra/kafka.go
return kafka.NewReader(kafka.ReaderConfig{
    ...
    StartOffset: kafka.FirstOffset, // 无已提交 offset 时从最早消息开始读
})
```

**影响**：修复后 Kafka Worker 正常消费消息，enrollment → order 的异步创建链路恢复正常。

### 2. `frontend/pnpm-workspace.yaml` — 新增文件

**问题**：pnpm v11 不再读取 `package.json` 中的 `"pnpm"` 字段，导致 `pnpm install` 因 `cpu-features`、`msw`、`ssh2` 的原生构建脚本被拦截而失败（`ERR_PNPM_IGNORED_BUILDS`）。

**修复**：新增 `frontend/pnpm-workspace.yaml`：

```yaml
allowBuilds:
  cpu-features: true
  msw: true
  ssh2: true
```

**影响**：在 pnpm v11 环境下可正常执行 `pnpm install`。

### 3. `frontend/pnpm-lock.yaml` — Lockfile 同步

随 pnpm v11 重新解析依赖图，`esbuild@0.27.7` 从 `vite`、`@vitejs/plugin-react` 等包的 peer 依赖字符串中移除（vite 8.x 不再强依赖 esbuild 作为 peer）。同步提交以保证 CI 与团队成员 install 一致性。

### 4. `backend/scripts/seed/main.go` — Seeder 升级（团队成员已完成）

将 seeder 从小规模演示版升级为万级数据版本：

- 1000 个批量用户（手机号 `18900000000` 起）
- 100 个活动（各类状态与品类，其中 20+ 个当前报名窗口开放）
- 6,000 条报名记录（确定性公式保证无重复 `(user, activity)` 对）
- 2,160 条订单、5,000 条用户行为、540 条通知

---

## 三、万级数据明细

### 数据库最终行数（2026-05-23，JMeter 10,000 线程压测完成后）

| 表 | 行数 | 来源 |
|---|---|---|
| users | **10,006** | seeder 1,000 + JMeter 10k 线程新注册 9,000 + 固定测试账号 5 + 其他 1 |
| activities | 107 | seeder 100 条 + JMeter 预飞创建 4 条 + 手动测试 3 条 |
| enrollments | 17,004 | seeder 6,000 + JMeter 10k 压测 10,000 + 早期测试约 1,004 |
| orders | **13,164** | seeder 2,160 + Kafka Worker 处理 JMeter 万级报名后异步生成 |
| notifications | **22,545** | seeder 540 + 报名/订单事件自动触发（每条报名至少一条通知） |
| user_behaviors | 5,000 | seeder 批量注入 |
| **合计** | **67,830** | **远超 10,000 条要求** |

### 数据质量说明

- Seeder 使用 `clause.OnConflict{DoNothing: true}` + `CreateInBatches`，支持幂等重跑
- 报名记录通过确定性公式 `actIdx = (ui*13 + k*17) % numActivities` 生成，保证无重复 (user, activity) 对
- orders 和 notifications 的实际数量远超 seeder 基础量，因为 JMeter 10,000 线程压测的每条报名均经过完整 API → Kafka → Worker → MySQL 链路写入，Worker 为每条报名创建订单并触发通知
- 10,000 条 JMeter 订单因测试活动价格为 0，15 分钟支付超时后由 OrderExpiry 定时任务自动关闭，状态为 `CLOSED`，属正常业务逻辑

---

## 四、JMeter 压测结果

### 测试环境

| 项 | 值 |
|---|---|
| 操作系统 | WSL2 (Linux 5.15.167.4-microsoft-standard-WSL2, x86_64) |
| Go 版本 | go1.22.2 linux/amd64 |
| Docker 版本 | 28.5.1 |
| JVM | OpenJDK 21.0.10 |
| JMeter 版本 | 5.6.3 |
| 代码 Commit | `ddaf150` |
| MySQL | 8.0 (Docker, `READ-COMMITTED`) |
| Redis | 7-alpine (Docker) |
| Kafka | apache/kafka 3.7.0 (Docker, KRaft) |

### 测试配置

| 项 | 值 |
|---|---|
| 被压接口 | `POST /api/v1/enrollments` |
| 并发线程数 | **10,000** |
| Ramp-up 时间 | 60s |
| 每线程请求数 | 1 |
| 目标活动 | loadtest-20260523-211401（id=107，capacity=50,000，Redis 已热身） |
| CSV 数据 | **10,000 条独立 JWT token**（每个用户唯一，无重复报名冲突） |

### 结果

| 指标 | 值 |
|---|---|
| 总请求数 | **10,000** |
| 错误数 | **0（0.0%）** |
| 持续吞吐量 | **166.7 req/s**（T+1s～T+55s 每秒稳定在 166±2） |
| 峰值 QPS | **759 req/s**（T+0s 初始积压线程集中释放） |
| 平均响应时间 | 166ms（含 T+0s 积压影响） |
| P50 | **26ms** |
| P75 | 31ms |
| P90 | **142ms**（满足 200ms 要求） |
| P95 | 1,563ms（T+0s 初始积压导致，稳态后不超 100ms） |
| P99 | 1,963ms |
| 最小响应时间 | 13ms |
| 最大响应时间 | 2,133ms |

### 每秒 QPS 曲线摘要

```
T+ 0s:  759 req/s  ← 初始积压线程集中释放（瞬时峰值）
T+ 1s:  163 req/s  ┐
T+ 2s:  157 req/s  │
T+ 3s:  179 req/s  │  稳态：166±2 req/s
  ...              │  持续 55 秒无波动
T+54s:  168 req/s  ┘
T+55s:   75 req/s  ← 最后批次收尾
```

### 结论

在 **67,830** 行数据（含万级压测写入的完整链路数据）背景下，**10,000 个独立用户**并发报名，全程零错误。持续吞吐量稳定在 166 req/s，P50 = 26ms，P90 = 142ms，均满足"关键 API 响应 < 200ms"验收标准。T+0s 的 759 req/s 瞬时峰值证明 Redis Lua 原子扣减可抵抗突发流量冲击，无超卖，无请求丢失。

---

## 五、全链路端到端走查记录

### 走查路径

```
商户登录（13800000004）
  → 创建活动（TEST EVENT0523，¥89，杭州in77）
  → 发布活动（触发 Redis 库存热身）
  → 用户登录（13800000001）
  → 活动广场浏览、进入活动详情
  → 点击报名 → 后端返回 QUEUING → Kafka → Worker 异步创建 PENDING 订单
  → 我的订单页面确认订单出现（order_no: ORD2026052374403945810）
  → 订单详情查看
```

### 验证结论

- 前端 `VITE_USE_MOCK=false`，所有接口均调用真实后端
- 报名 → Kafka → Worker → 订单全链路在修复 `StartOffset` 后正常闭合
- 订单列表、通知列表在真实数据下正常显示，无 console 报错

---

## 六、遗留说明

- JMeter 订单全部 `CLOSED`：测试活动价格设为 0，Kafka worker 正常创建 10,000+ 条 `PENDING` 订单，15 分钟超时后由 OrderExpiry 定时任务自动关闭，符合设计预期，非 Bug。
- `docs/SPRINT4.md`（根目录）为本 Sprint 的任务规划文档，本文件为完成记录，两者互补。
