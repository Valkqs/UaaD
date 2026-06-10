# UAAD 文档中心

> 面向开发者、AI Agent、负责人使用的统一文档入口。
> 文档按“长期真理源 / 操作手册 / 阶段记录 / 治理规则”分组，减少在多个入口重复维护同一份说明。

**最后更新：2026-06-10**

---

## 1. 先读这些

按推荐顺序阅读：

1. [SRS.md](./SRS.md)：项目做什么，边界是什么
2. [SYSTEM_DESIGN.md](./SYSTEM_DESIGN.md)：架构、API 契约、核心数据流
3. [COLLABORATION.md](./COLLABORATION.md)：多人协作、冲突规避、Sprint 组织方式
4. [../CONTRIBUTING.md](../CONTRIBUTING.md)：提交、分支、质量要求
5. [../frontend/FRONTEND_SPEC.md](../frontend/FRONTEND_SPEC.md)：前端开发规范
6. [DOCS_PROMPTS_GOVERNANCE.md](./DOCS_PROMPTS_GOVERNANCE.md)：文档与 Prompt 管理规则

---

## 2. 长期真理源

这些文档描述长期稳定的项目事实，优先更新这里，而不是在其他入口重复解释。

| 文件 | 用途 | 谁最常看 |
|---|---|---|
| [SRS.md](./SRS.md) | 需求范围、用户角色、功能与非功能要求 | 全员 |
| [SYSTEM_DESIGN.md](./SYSTEM_DESIGN.md) | 架构、API 契约、模块设计、前端路由与数据流 | 全员 |
| [DDL.md](./DDL.md) | 数据库结构、字段、Redis 键位约定 | 后端 |
| [TICKET_ENGINE_DESIGN.md](./TICKET_ENGINE_DESIGN.md) | 抢票引擎、库存控制、并发设计 | 后端抢票链路 |
| [RECOMMENDATION_DESIGN.md](./RECOMMENDATION_DESIGN.md) | 推荐算法与数据链路设计 | 推荐模块 |
| [../frontend/FRONTEND_SPEC.md](../frontend/FRONTEND_SPEC.md) | 前端技术约束、数据流、命名规范 | 前端 |

---

## 3. 操作与联调

这些文档解决“怎么跑”“怎么测”“怎么观察系统”。

| 文件 | 用途 |
|---|---|
| [RUN_GUIDE.md](./RUN_GUIDE.md) | 本地拉起前后端与依赖服务 |
| [ST_BASELINE.md](./STRESS_TEST/ST_BASELINE.md) | 压测说明 |
| [Prometheus_And_Grafna.md](./Prometheus_And_Grafna.md) | 监控与可观测配置说明 |
| [walkthrough.md](./walkthrough.md) | 最近一轮联调、修复与验证记录 |
| [../backend/tests/jmeter/ACCEPTANCE_CHECKLIST.md](../backend/tests/jmeter/ACCEPTANCE_CHECKLIST.md) | 压测验收检查项 |

---

## 4. 协作与计划

这些文档描述“谁做什么、阶段推进到哪、当前有哪些活动变更”。

| 文件 | 用途 |
|---|---|
| [COLLABORATION.md](./COLLABORATION.md) | 团队协作方式、冲突规避、分工策略 |
| [SPRINT1.md](./SPRINT1.md) | Sprint 1 记录 |
| [SPRINT2.md](./SPRINT2.md) | Sprint 2 记录 |
| [SPRINT3.md](./SPRINT3.md) | Sprint 3 记录 |
| [SPRINT5.md](./SPRINT5.md) | Sprint 5 安全测试、实机运行演示与抗压能力展示 |
| [SPRINT6.md](./SPRINT6.md) | Sprint 6 最终总结、缺陷反思、课程建议与配套材料汇总 |
| [wbs.puml](./wbs.puml) | 工作分解结构图 |
| [../backend/task.md](../backend/task.md) | 后端任务清单 |
| [../frontend/task.md](../frontend/task.md) | 前端任务清单 |

> 说明：`SPRINT*.md` 更偏阶段性计划与执行记录；若结论已经稳定，应回写到长期真理源文档。

---

## 5. OpenSpec 变更

OpenSpec 用来管理“正在变化中的东西”，尤其是跨模块、非琐碎、需要留痕的需求、设计和治理变更。

| 位置 | 用途 |
|---|---|
| [../openspec/specs/](../openspec/specs/) | 已沉淀的 capability spec |
| [../openspec/changes/](../openspec/changes/) | 活跃或待归档的 change proposal / design / tasks |

### 5.1 什么时候必须使用 OpenSpec

下面这些情况必须先开 OpenSpec change：

- 新增或改变业务能力、状态机、权限规则、数据结构
- 修改 API 契约、错误码、响应结构、分页结构
- 调整跨前后端的数据流或联调方式
- 引入影响多个模块的重构、治理规则、目录结构变化
- 修改文档体系、Prompt 体系、AI 工作流、OpenSpec 管理规则
- 任何需要多人评审、分阶段实现、后续归档的非琐碎改动

下面这些情况可以不新建 OpenSpec：

- 修 typo、格式、链接等局部小问题
- 不改变行为的注释或文案微调
- 单文件、低风险、无协作影响的实现细节修补

### 5.2 基本流程

推荐流程：

1. Explore：先澄清问题、范围、风险，不急着写代码
2. Propose：创建 `openspec/changes/<change-name>/`
3. Design：记录关键方案、非目标、取舍
4. Spec：用 requirement / scenario 写可验收行为
5. Tasks：拆成可执行任务
6. Apply：按任务实现，并及时勾选 `tasks.md`
7. Verify：运行测试、检查文档和入口引用
8. Archive：完成后归档，并把稳定结论同步到长期文档或主 spec

### 5.3 常用命令

```powershell
# 查看当前变更
openspec list
openspec list --json

# 新建变更
openspec new change "<change-name>"

# 查看变更状态
openspec status --change "<change-name>"
openspec status --change "<change-name>" --json

# 获取 artifact 编写指令
openspec instructions proposal --change "<change-name>" --json
openspec instructions design --change "<change-name>" --json
openspec instructions specs --change "<change-name>" --json
openspec instructions tasks --change "<change-name>" --json

# 获取实现阶段指令和任务进度
openspec instructions apply --change "<change-name>" --json
```

### 5.4 产物要求

每个非琐碎 change 至少应包含：

- `proposal.md`：为什么做、做什么、不做什么、影响范围
- `design.md`：怎么做、关键决策、替代方案、风险
- `tasks.md`：可执行任务清单，必须使用 `- [ ]` / `- [x]`
- `specs/<capability>/spec.md`：当变更新增或修改可验收能力时必须提供

### 5.5 当前约定

- 长期稳定规则写入 `docs/`、`frontend/FRONTEND_SPEC.md` 等真理源
- 非琐碎变更先进入 `openspec/changes/<change>/`
- 稳定结论再回写到长期文档
- 已完成的 change 应归档，不长期堆在活跃变更目录

---

## 6. 文档维护规则

精简后的维护原则：

- 不在多个入口重复维护同一段详细说明
- 根 `README.md` 只做项目入口，不再充当超长文档清单
- `docs/README.md` 作为文档导航中心
- 详细治理规则见 [DOCS_PROMPTS_GOVERNANCE.md](./DOCS_PROMPTS_GOVERNANCE.md)

---

## 7. 推荐更新路径

遇到下面这些变更时，优先改这里：

- 新需求或范围调整：先改 [SRS.md](./SRS.md)
- API / 状态机 / 数据流变化：先改 [SYSTEM_DESIGN.md](./SYSTEM_DESIGN.md)
- 数据表或字段变化：先改 [DDL.md](./DDL.md)
- 前端规范变化：先改 [../frontend/FRONTEND_SPEC.md](../frontend/FRONTEND_SPEC.md)
- 协作流程变化：先改 [COLLABORATION.md](./COLLABORATION.md)
- 文档或 Prompt 管理规则变化：先改 [DOCS_PROMPTS_GOVERNANCE.md](./DOCS_PROMPTS_GOVERNANCE.md)
- 非琐碎跨模块改动：先开 [../openspec/changes/](../openspec/changes/) change
