# Walkthrough

## 2026-04-13 · Sprint2 乙同学 API 联调迭代记录

### 变更概览
- 新增前端测试基建（Vitest + jsdom）并接入 `pnpm test` 脚本。
- 在全局 HTTP 层实现 401 登出事件派发，解决仅清理 localStorage 导致的同页鉴权态不同步问题。
- 在全局 HTTP 层实现 `HTTP 200 + code=1101` 的业务失败拦截并转为 reject，避免库存不足误判为成功。
- 新增报名 API 封装与类型：
  - `createEnrollment(activityId)` 对接 `POST /enrollments`，支持 `202 + code=1201` 排队受理。
  - `getEnrollmentStatus(enrollmentId)` 对接 `GET /enrollments/:id/status`。
  - `listMyEnrollments(page,pageSize)` 对接 `GET /enrollments` 分页数据。

### Diff 思路
- 先补测试框架，再做拦截器与 endpoint，保证每次变更都有可执行回归。
- 契约映射策略统一在 endpoint 层处理：后端 snake_case 字段映射为前端 camelCase，页面层不承担字段转换。
- 全局错误语义集中在 axios 拦截器，页面调用方只处理成功对象或 reject 对象，避免分散判断逻辑。

### 验证结果
- `pnpm test`：通过（新增 smoke、401、1101、enrollments endpoint 测试）。
- `pnpm build`：通过。
- `pnpm lint`：存在仓库既有错误（Merchant 页面中的 `react-hooks/set-state-in-effect`），与本次改动无直接关联。

## 2026-04-13 · 前端 lint 修复迭代

### 变更概览
- 修复 `MerchantActivities.tsx` 中 `react-hooks/set-state-in-effect`：
  - 去除 effect 内同步 `setBanner`，改为从路由 state 派生 banner。
  - 首次加载改为带取消标记的异步请求，避免 effect 中同步 setState。
  - 发布后刷新列表改为独立 `reloadList()`。
- 修复 `MerchantActivityEdit.tsx` 中 `react-hooks/set-state-in-effect`：
  - 去除 effect 内同步 `setLoading(false)` 分支。
  - 引入 `isActivityIdValid` 派生判定与 invalid-id 早返回视图。
  - 为异步详情加载增加取消标记，卸载时避免写状态。
- 在 `eslint.config.js` 中忽略 `public/mockServiceWorker.js`，消除生成文件的无效注释 warning。

### 验证结果
- `pnpm lint`：通过。
- `pnpm test`：通过。
- `pnpm build`：通过。

## 2026-04-13 · 商户控制台与通知联调记录

### 变更概览
- **商户端**：`MerchantDashboard` / `MerchantActivities` / `MerchantActivityNew` / `MerchantActivityEdit` 对接真实商户活动接口；列表、发布、创建、编辑、详情加载失败时使用页面横幅展示 `getRequestErrorMessage` 文案，不单依赖控制台；创建成功可通过回跳 `location.state.message` 提示；发布走 `publishMerchantActivity` 并刷新列表。
- **MerchantForm**：提交前校验报名开始早于报名结束、报名结束早于活动开始；表单字段与 `toBackendActivityInput` / 后端约定一致。
- **通知 API 客户端**（`api/endpoints/notifications.ts`）：`GET /notifications` 分页（`list`、`total`、`page`、`page_size`）；`GET /notifications/unread-count` 读取 `data.unread_count`；`PUT /notifications/:id/read` 标记已读；将 `created_at`、`is_read` 等蛇形字段映射为 `NotificationItem`。
- **通知页面**（`pages/Notifications.tsx`）：首屏加载 + 「加载更多」分页追加；空态与骨架占位；请求失败横幅；点击条目调用 `markNotificationRead` 并乐观更新已读样式。
- **通知铃铛**（`components/public/NotificationBell.tsx`）：已登录时拉取未读数；依赖路由 `pathname`、定时轮询与 `window` `focus` 刷新，保持角标与列表大致同步。
- **共用错误文案**（`utils/requestErrorMessage.ts`）：供商户与通知等页面统一解析 axios 业务错误与网络错误。


### 验证结果
- 代码层面：商户列表 → 新建/编辑 → 发布、通知列表 → 标记已读、铃铛未读数刷新路径已实现。
- 运行时：需在商户账号与普通用户账号下用 Chrome DevTools 确认核心请求无持续 401/404，并与后端通知写库行为一致（未读数、已读状态）。

## 2026-05-02 · Worker Prometheus 指标（Sprint3 异步链路可观测性）

### 变更概览
- `internal/middleware/metrics.go`：在保留 `http_requests_total` / `http_request_duration_seconds` 的前提下，新增 enrollment Kafka Worker 指标：
  - `worker_messages_processed_total{status}` — 处理成功/失败次数；
  - `worker_message_processing_duration_seconds{status}` — 单条消息处理耗时；
  - `worker_kafka_lag_approx{topic}` — 基于 `kafka-go` `Reader.Stats().Lag` 的滞后近似值。
- 导出 `RecordWorkerMessage(status, durationSec)`、`SetWorkerKafkaLag(topic, lag)` 供 worker 调用，避免在业务包中重复声明 Prometheus 向量。
- `internal/worker/enrollment_worker.go`：`handleMessage` 在 JSON 解析成功后计时，事务失败路径与成功路径分别记录 `failure` / `success`；`Run` 在每读完一条消息后刷新 lag gauge。

### Diff 思路
- 指标统一注册在既有 `MustRegister`，HTTP 与 Worker 共用同一 scrape 端点暴露。
- Lag 在消费循环内更新，成本低且与消息处理节奏一致。

### 验证结果
- `go build ./...`（backend）：通过。

## 2026-05-02 · Sprint3 压测基线固化（JMeter + 文档）

### 变更概览
- **`backend/tests/jmeter/enrollment-load.jmx`**
  - HTTP 断言改为 **`200|202|409|410`**，**`assume_success=false`**，使 **5xx** 正确记为样本失败（修复原先 5xx 仍可能记为通过的问题）。
  - 新增 **`JSONPostProcessor`** 提取 **`$.code` → `resp_code`**。
  - 新增 **`JSR223PostProcessor`（Groovy）**：按 Sprint3 口径写入 **`outcome`**：`QUEUED`（202+1201）、`SOLD_OUT`（200/410+1101）、`CONFLICT`（409）、`FAILURE`（5xx 或非预期组合）。
  - 新增默认 **禁用** 的 **「Sprint3 大规模压测 3000 并发」** 线程组（Ramp 60s）；与 1000 线程组二选一启用；TestPlan 注释说明 `gen_jmeter_data -count` 与线程数对齐。
- **`docs/STRESS_TEST.md`**：重写为 Sprint3 基线文档 — 前置条件、主接口与判定、执行步骤、`outcome` 说明、**基线结果模板（5 项）**、3000/5000 大规模说明、Go stress 测试引用、相关文件索引。
- **`backend/tests/jmeter/ACCEPTANCE_CHECKLIST.md`**：新增 **「七、Sprint 3 压测基线验收」**；§5.1 补充 `run-jmeter-report.sh` 与 P99 / `outcome` 对照。
- **`backend/tests/jmeter/run-jmeter-report.sh`**：新增与 `run-jmeter-report.ps1` 等价的 **bash** 一键报告脚本（时间戳目录、`-e -o` HTML）。

### Diff 思路
- 断言与脚本分层：**状态码** 由 `ResponseAssertion` 兜底；**业务口径** 由 JSON 提取 + Groovy 归类，便于报告外二次统计或人工对 JTL。
- 大规模线程组独立且默认关闭，避免误跑 3000 行 CSV 未生成时全线程失败。

### 验证结果
- 文档与 JMeter XML、shell 脚本为静态交付；**需在本地安装 JMeter 且拉起全栈后** 执行 `bash run-jmeter-report.sh` 做运行时验证。

## 2026-05-02 · 压力测试集成 CI 管线

### 变更概览
- **`.github/workflows/ci.yml`**（修改）：新增 `stress-test` Job，在 `backend` Job 通过后自动执行集成测试与压力测试。
  - 使用 GitHub Actions `services` 拉起 MySQL 8.0（`tmpfs` 加速）、Redis 7-alpine、Kafka 3.7.0（KRaft 单节点）。
  - 通过环境变量 `DB_HOST`/`REDIS_HOST`/`KAFKA_BROKER` 对接后端 `config.Load()` 逻辑。
  - 流程：编译后端 → 启动 server → seed → 执行 `integration` 标签测试（并发抢票 stock=1、报名幂等、库存不足、权限校验、支付流程）→ 执行 `stress` 标签测试（500 goroutine 抢 10 张票零超卖验证、吞吐量基准测试）→ 清理 server 进程。

### Diff 思路
- Job `needs: backend` 确保编译与 vet 通过后再跑重量级集成。
- MySQL 使用 `--tmpfs=/var/lib/mysql` 避免 CI 磁盘瓶颈。
- server 启动后轮询 `/metrics` 端点确认就绪，最多等待 30 秒。
- `if: always()` 确保 server 进程清理不受测试结果影响。

### 验证结果
- YAML 语法检查通过。

## 2026-05-02 · Sprint3 大规模压力测试执行与文档固化

### 变更概览
- **压力测试执行**：依次完成 1000 / 3000 / 5000 三轮并发压测，被压接口为 `POST /api/v1/enrollments`。
  - 1000 并发：100% 成功率，P95=11ms，P99=39ms，0 个 5xx。
  - 3000 并发：100% 成功率，P95=11ms，P99=23ms，0 个 5xx。
  - 5000 并发（实际 4061）：100% 成功率，P95=11ms，P99=26ms，0 个 5xx；JMeter JVM 线程上限为客户端侧瓶颈。
- **数据一致性验证**：三轮压测后 Redis 库存均 >= 0，MySQL 无超卖、无重复成功报名。
- **`docs/STRESS_TEST_REPORT.md`**（新增）：Sprint 3 大规模压力测试正式报告，含环境配置、三轮结果对比、库存一致性验证、5000 并发瓶颈分析与降级方案、Grafana 监控观测指南。
- **`backend/tests/jmeter/enrollment-load.jmx`**（修改）：新增默认禁用的「Sprint3 冲刺目标 5000 并发」线程组（Ramp 90s），三档线程组（1000 / 3000 / 5000）齐备。
- **`docs/STRESS_TEST.md`**（修改）：补充 5000 并发线程组说明、macOS 线程限制已知问题、报告引用链接、能力边界说明（已具备 vs 未覆盖）。
- **`docs/RUN_GUIDE.md`**（重写）：补充三态运行说明（开发态 / 联调态 / 演示态）、Prometheus / Grafana 启动与配置、JMeter 压测执行指南、新增排查建议。
- **`backend/tests/jmeter/ACCEPTANCE_CHECKLIST.md`**（修改）：§7.1 跑前检查项全部勾选，新增 §7.5 Sprint 3 压测基线结果表与 5000 并发瓶颈说明。

### Diff 思路
- 先执行压测获取真实数据，再回填文档和报告，确保所有数字有 JTL / Redis / MySQL 交叉验证。
- JMX 中三个线程组互斥启用，`run-jmeter-report.sh` 自动解析启用的线程组数量生成对应行数的 CSV，无需手工同步。
- 5000 并发未完全达到，如实记录瓶颈原因（客户端 JVM 线程限制）与降级方案，符合 SPRINT3.md 对「若未达到冲刺目标须说明限制条件」的要求。

### 验证结果
- 三轮压测 JTL 与 HTML 报告已生成（`backend/tests/jmeter/out/`，gitignored）。
- Redis / MySQL 一致性检查通过。
- 文档交叉引用链路完整：`SPRINT3.md` → `STRESS_TEST.md` → `STRESS_TEST_REPORT.md` → `ACCEPTANCE_CHECKLIST.md`。

## 2026-05-02 · Sprint3 Grafana 监控栈基础设施代码化

### 变更概览
- **`docker-compose.yaml`**（修改）：Prometheus 新增 `extra_hosts: host.docker.internal:host-gateway` 以拉取宿主机 `:8080/metrics`；Grafana 挂载 `./infra/grafana/provisioning` 目录实现零手工配置启动。
- **`infra/grafana/provisioning/datasources/prometheus.yaml`**（新增）：声明 uid `prometheus` 数据源，`url: http://prometheus:9090`（Docker 内网服务名），设为默认；`timeInterval: 15s` 与 Prometheus `scrape_interval` 对齐。
- **`infra/grafana/provisioning/dashboards/dashboard.yaml`**（新增）：配置 file provider，从 `/etc/grafana/provisioning/dashboards` 自动加载 JSON，`allowUiUpdates: true` 允许在 UI 二次调整后不被覆盖。
- **`infra/grafana/provisioning/dashboards/uaad-sprint3.json`**（新增）：预置 3 行 9 面板 Dashboard（uid `uaad-sprint3`，标签 `uaad/sprint3/enrollment`）：
  - **Row 1**：全站 HTTP 吞吐、报名接口吞吐、5xx 错误率、状态码 Stacked 分布；
  - **Row 2**：报名接口 P50/P95/P99 三联延迟、全站按 path 的 P95；
  - **Row 3**：Worker success/failure Stacked 吞吐（绿/红）、Worker 处理耗时 P95、Kafka 消费滞后近似 Gauge。
  - 所有查询使用 `$__rate_interval` 动态窗口；刷新间隔 10s；默认时间范围 `now-30m`。

### Diff 思路
- Grafana provisioning（数据源 + Dashboard）完全声明式，容器重建后自动恢复，无需手动点击 UI 建面板。
- Dashboard JSON 内 datasource 引用 `uid: prometheus` 与 datasource YAML 中 `uid` 字段一致，避免「数据源 not found」问题。
- Worker 面板 success 覆盖绿色、failure 覆盖红色，便于压测时一眼判断健康状态。

### 启动方式
```bash
cd infra
docker compose up -d
# Grafana: http://localhost:3000  (admin / admin)
# Prometheus: http://localhost:9090
# Dashboard 自动出现在 "UAAD / Sprint3" 文件夹
```

## 2026-05-02 · Grafana Dashboard 优化 — 填满空面板

### 变更概览
- **`infra/grafana/provisioning/dashboards/uaad-sprint3.json`**（重写）：将原 3 行 9 面板重构为 5 行 12 面板，消除大量空面板问题：
  - **新增 Row 0 — Overview Stats**（4 个 Stat 面板）：总请求量、5xx 错误数、平均请求延迟、Worker 处理消息数。Stat 面板在数据为 0 时显示 "0" 而非 "No data"，解决 rate() 在无流量时返回空的问题。
  - **Row 1 — HTTP 吞吐**：原 Panel 1（全站 RPS）与 Panel 2（报名接口 RPS）合并为一个面板，同时展示全站 RPS 与按 path 分组的吞吐；状态码 Stacked 分布保留。
  - **Row 2 — HTTP 时延**：原 Panel 5 从仅监控 `/api/v1/enrollments` 改为监控全站 P50/P95/P99；P95 by path 保留。
  - **Row 3 — Worker & Kafka**：原 3 个 8w 面板合并为 2 个 12w 面板（Worker 吞吐 + Worker 耗时与 Kafka Lag 合并）。
  - **新增 Row 4 — Go Runtime**（3 个面板）：Go Goroutines、进程内存（RSS + Heap）、GC Duration。这些指标由 Go Prometheus client 自动注册，从后端启动即有数据，确保 Dashboard 永远不会全部为空。
  - 所有 rate() 查询追加 `or vector(0)` 兜底，无流量时显示 0 而非 "No data"。
- **`backend/internal/middleware/metrics.go`**（修改）：pre-init 路径从 4 条扩展到 8 条（新增 `/api/v1/orders`、`/api/v1/recommendations`、`/api/v1/behaviors`、`/api/v1/notifications`）；状态码 pre-init 新增 `400`、`401`、`403`、`404`。

### Diff 思路
- 空面板的根本原因：(1) rate() 在无流量时返回 NaN；(2) 过于细分的 path 过滤条件在无流量时无匹配；(3) Worker 指标仅在 Kafka 消费时产生。
- 三管齐下：Stat 面板用 `increase()` + `$__range` 显示累计值（0 也有意义）；timeseries 追加 `or vector(0)` 防空；Go Runtime 指标天然有数据填充底部行。
- 面板合并策略：功能重叠的合并（全站 RPS + 按路径 RPS → 同一面板双查询），单一路径 P50/P95/P99 → 全站 P50/P95/P99，减少总面板数同时提高每个面板出数据的概率。

### 验证方式
- 重启 Grafana 容器：`docker compose restart grafana`，Dashboard 自动重新加载。
- 即使后端无流量，Row 0（Stat 全部显示 0）和 Row 4（Go Runtime 有实时数据）保证不出现空面板。

## 2026-05-02 · Sprint3 验收文档整理与运行链路清晰化

### 变更概览
- **`docs/STRESS_TEST/ST_BASELINE.md`**（重写）：
  - 新增「一、能力与监控现状总览」章节，包含两张表格：§1.1 已具备的能力（17 项，覆盖核心链路、数据一致性、压测、监控、前端）和 §1.2 仍为基础版或暂未覆盖的监控（7 项，如 Kafka Lag 精度、MySQL/Redis 组件指标、Alertmanager）。
  - 修正架构描述：MySQL 隔离级别从「可重复读（RR）」改为「READ-COMMITTED」，与 `docker-compose.yaml` 中 `--transaction-isolation=READ-COMMITTED` 一致。
  - 补充 §4.5 5000 并发双进程完整命令、§4.6 压测后验证命令（含重复报名检查）。
  - 新增 §5 压测结果基线摘要表与数据一致性验证总结。
  - 新增 §6 相关文件索引（9 个关键文件）。
- **`docs/STRESS_TEST/ST_REPORT.md`**（修改）：
  - §6.2 结论修正：5000 并发描述从「实际到达的 4061 并发」更新为「双进程方案成功完成完整 5000 并发测试」，与报告正文数据一致。
  - 新增 §6.3 当前监控能力边界表（7 项），明确哪些指标已就绪、哪些仍为基础版或未采集。
- **`docs/RUN_GUIDE.md`**（重写）：
  - 新增 §2 三种运行态详细说明（开发态 / 联调态 / 演示态），含各态的 Docker 服务组合、前后端配置要求、启动命令。
  - 新增 §3.2 各服务详情表（镜像、容器名、端口、数据卷）、§3.3 连接参数表。
  - 新增 §4 后端环境配置完整环境变量清单（11 个关键变量含默认值）。
  - 新增 §4.3 Seed 数据说明（含具体账号表）。
  - 新增 §5.3 Mock 策略对照表，明确联调/验收阶段必须关闭 Mock。
  - 新增 §6 Prometheus / Grafana 完整说明（数据源 provisioning、Dashboard 面板布局 5 行 12 面板、指标端点验证命令）。
  - 新增 §8 完整启动流程速查（从零拉起 + 日常开发两套 checklist）。

## 2026-05-06 · 后端闭环收口（Batch 1）

### 变更概览
- 报名链路改为先落库 `QUEUING`，再投递 Kafka；返回值包含 `enrollment_id` 与 `queue_position`，便于前端轮询。
- Enrollment Worker 改为更新已存在的报名记录为 `SUCCESS`，创建订单并递增 `activities.enroll_count`，避免重复入库；非 `QUEUING` 状态消息直接跳过。
- 库存查询改为 Redis 优先：新增 `StockEngine.GetStock`，`/activities/:id/stock` 与详情页库存读取 Redis，异常时回退 DB 审计值。
- 更新通知接线文档与报名接口示例，移除未实现的 `estimated_wait_seconds` 字段。

### Diff 思路
- 先解决排队 ID 丢失的问题（报名记录缺失会导致无法轮询），再让 Worker 基于 `enrollment_id` 完成幂等落盘。
- Redis 是实时库存的真来源，API 以 Redis 值为主，DB 仅作为审计回退。

### 验证结果
- 通过：`go test`（选定用例）
  - `backend/internal/service/stock_lua_test.go`
  - `backend/internal/handler/activity_handler_test.go`
  - `backend/internal/handler/enrollment_handler_test.go`

## 2026-05-06 · 后端闭环收口（Batch 2）

### 变更概览
- 订单状态更新改为乐观锁：新增 `UpdateStatusFromPending`，`Pay` 与 `ScanExpired` 使用条件更新避免重复回补。
- 订单过期扫描接入 `ORDER_EXPIRE` 通知，并补齐活动标题查询。

### Diff 思路
- 所有 `PENDING -> PAID/CLOSED` 的状态变更都通过一次条件更新完成，避免并发 TOCTOU。
- 只有成功关闭订单时才回补库存并发送通知，防止重复消息影响一致性。

### 验证结果
- 通过：`go test`（选定用例）
  - `backend/internal/service/order_service_flow_test.go`
  - `backend/internal/service/order_service_test.go`

## 2026-05-06 · 后端闭环收口（Batch 3）

### 变更概览
- 新增报名取消接口：支持 `QUEUING` 直接取消与 `SUCCESS + PENDING` 订单取消。
- 取消流程统一通过乐观锁更新订单状态，成功后回补库存并更新报名为 `CANCELLED`。
- 同步 SRS 与系统设计文档，补齐取消能力与 API 契约。

### Diff 思路
- 先对报名记录做状态判断，再走 `PENDING -> CLOSED` 条件更新，避免与支付/超时扫描并发重复回补。
- 对 `CANCELLED` 走幂等路径，避免重复取消报错。

### 验证结果
- 通过：`go test`（选定用例）
  - `backend/internal/service/enrollment_cancel_test.go`
  - `backend/internal/service/order_service_flow_test.go`
  - `backend/internal/service/order_service_test.go`

## 2026-05-06 · 后端闭环收口（Batch 4）

### 变更概览
- 新增 `/api/v1/health` 健康探针别名，和验收文档保持一致。
- 对齐系统设计文档中活动详情与库存响应示例，避免与实际响应不一致。

### Diff 思路
- 在现有 `/health` 基础上提供版本化别名，避免文档与实现分叉。
- 文档示例以代码为准，减少前后端联调误判。

### 验证结果
- 通过：`go test`（选定用例）
  - `backend/internal/handler/activity_handler_test.go`
  - 新增 §9 常见问题排查（10 类典型问题及解决步骤）。

### Diff 思路
- ST_BASELINE.md 从纯执行步骤文档升级为「能力现状 + 执行指南 + 结果基线」三段式，使验收评审可在一个文件内判断项目完成度。
- RUN_GUIDE.md 对标「新同学首日可独立拉起全栈」的目标，将三态配置差异显式列出，避免因环境变量、Mock 开关等隐式差异导致联调失败。
- ST_REPORT.md 的 4061 → 5000 修正消除了正文数据与结论文字的矛盾。

### 验证结果
- 文档交叉引用链路完整：`RUN_GUIDE.md` → `ST_BASELINE.md` → `ST_REPORT.md` → `SPRINT3_CHECKLIST.md`。

---

## 2026-05-02 · Docker 容器化压测 & 报告定稿

### 背景
macOS `kern.num_taskthreads=4096` 为只读硬限制，宿主机 JMeter 单进程无法创建 5000 线程。

### 主要变更
- 构建 ARM64 原生 JMeter Docker 镜像（`jmeter-arm64:5.6.3`，基于 `eclipse-temurin:21-jre-alpine`），绕过 macOS 线程限制。
- 三轮测试（1000/3000/5000）全部在 Docker 容器中执行，单进程一次性完成 5000 线程。
- **`docs/STRESS_TEST/ST_REPORT.md`**（重写）：
  - 数据全部替换为 Docker 容器执行结果（活动 45/46/47）。
  - 5000 并发 P95 = 2ms，P99 = 4ms，与 1000/3000 表现一致，证明服务端无瓶颈。
  - 新增 §四 Docker 容器方案说明（macOS 线程限制表、达成方式、JVM 配置）。
  - 新增 §5.2 Grafana 监控截图。
- HTML 报告归档至 `docs/STRESS_TEST/{1000,3000,5000}t_st/`。

### Diff 思路
- 将 JMeter 执行层从 macOS 宿主机迁移到 Docker Linux 容器，从根本上消除客户端线程瓶颈。
- 5000 并发从「双进程拆分 + 高尾部延迟」升级为「单进程 + 毫秒级延迟」，报告数据更准确反映服务端真实性能。
---

## 2026-05-09 · Sprint3 Backend 闭环组：通知与 Worker 收口

### 背景
对照 SPRINT3 §三 Backend 闭环组的 8 个任务做了一次严格审计，发现：

**已做 → 已验证：**
- task 4 状态原子性（Pay/ScanExpired 乐观锁）— 已通过 commit `7dab926`
- task 7 主动取消机制（`POST /enrollments/:id/cancel`）— 已通过 commit `ee4507c`
- 服务层 ScanExpired+通知测试 — `service/order_service_flow_test.go`

**已写代码但有 BUG：**
- task 1 通知文案禁用 "unknown" 占位 → **代码里反而存在三处 "unknown" 字面量**
- task 2 通知 `related_id` 一致性 → Worker 失败路径传 `0` 而非真实 enrollment_id
- task 4 状态对齐 → Worker 事务失败后 enrollment 永远卡 QUEUING
- task 5 Worker 单元测试 → `internal/worker/` 下零测试文件

**未做：**
- task 3 Kafka 批量消费（仍 `ReadMessage` 单条）
- task 8 活动逾期自动转 OFFLINE

### 主要变更（4 commit）

| Commit | 内容 |
|---|---|
| `00a7f78 fix(notification)` | 抽 resolveActivityTitle 助手；"unknown" → "活动 #<id>"；NotifyEnrollFail 用真实 enrollment_id；事务失败后 best-effort 把 status QUEUING→FAILED |
| `31a5c26 test(worker)` | `internal/worker/enrollment_worker_test.go`（7 tests, 全 PASS）：success / tx-fails / already-not-queuing / malformed-json / title-fallback × 3 |
| `c548c5a test(integration)` | `tests/sprint3_order_expire_test.go`：构造真实 OrderService 绕过 5min ticker，验证全链路 ORDER_EXPIRE 通知 + 库存回补 |
| `<本提交>` | walkthrough 记录 + 剩余缺口公示 |

### 验证
- `go build ./...` exit 0
- `go vet ./...` exit 0
- `go vet -tags=integration ./tests/` exit 0
- `go test ./internal/worker/` 7/7 PASS
- `go test ./internal/service/ -run TestOrderService_ScanExpired` PASS
- 集成测试需要完整运行栈（MySQL+Redis+Kafka+server），代码层面已验证

### 剩余缺口（明确未做 — 等下一次提交）

1. **task 3 — Kafka 批量消费**
   - 影响：单条处理在大流量下可能积压
   - 改造点：`worker.Run()` 用 `FetchMessage`/`ReadBatch` + 显式 `CommitMessages`
   - 风险：批量提交点位、错误隔离、与现有指标对接

2. **task 8 — 活动逾期自动转 OFFLINE**
   - 影响：过期活动一直显示在列表里
   - 改造点：新增独立 ticker job（参考 `stock_reconciler.go` 模式）扫描 `activity_at < NOW() AND status != 'OFFLINE'`
   - 与 task 8「热度计算口径」是两件事，需分开验收

### Diff 思路
- 通知质量是用户可见的硬指标，先把 "unknown" 字面量和断链 related_id 修掉
- Worker 失败路径既要补偿（Redis rollback）也要终结（status FAILED）+ 告知用户（通知）
- 测试分层：unit 用 sqlite in-memory + stub 跑事务路径，integration 用真实运行栈跑闭环
