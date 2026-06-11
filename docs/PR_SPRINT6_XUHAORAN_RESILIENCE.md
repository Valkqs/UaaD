# PR 草稿：补全 Sprint6 徐浩然弹性设计报告

## Summary

- 补全 `docs/SPRINT6.md` 中徐浩然个人心得体会，说明其在报名、库存、订单闭环和弹性设计专题中的工作。
- 扩展 Sprint6 `8.4 弹性设计`，详细描述 Redis 原子扣减、Kafka 削峰、Worker 异步落库、幂等状态流转和库存回滚补偿。
- 补充相关测试与验证材料引用，明确当前不足和后续弹性演进方向。

## Details

- Redis Lua 脚本承担报名入口准入、库存扣减、重复报名拦截和排队序号生成。
- Kafka 将报名请求接收和订单持久化解耦，Worker 在事务中完成报名成功、订单创建和活动计数更新。
- 条件状态更新用于保证报名、订单、取消和过期扫描的幂等性。
- Redis `Rollback` 覆盖 Kafka 投递失败、Worker 事务失败、用户取消和订单过期等补偿路径。
- 后续建议包括 Kafka 重试和死信队列、断路器、降级、限流、Worker 水平扩展以及 Redis/MySQL 库存对账。

## Validation

- 文档变更，无运行时代码变更。
- 已人工核对 `backend/internal/service/stock_lua.go`、`backend/internal/service/enrollment_service.go`、`backend/internal/worker/enrollment_worker.go`、`backend/internal/service/order_service.go` 与报告描述一致。

## Notes

- 本 PR 只覆盖 Sprint6 中“徐浩然”负责的弹性设计专题和个人心得补充，不修改业务代码。
