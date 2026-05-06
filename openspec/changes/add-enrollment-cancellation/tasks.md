# Tasks

## 1. 后端核心与安全性
- [x] 1.1 在 `order_repository.go` 中实现 `UpdateStatusFromPending`，使用乐观锁更新逻辑。
- [x] 1.2 重构 `ScanExpired()`，改用安全更新逻辑，防止多个实例重复回补库存。
- [x] 1.3 重构 `Pay()`，改用安全更新逻辑。

## 2. API 实现
- [x] 2.1 在 `enrollment_service.go` 中新增 `Cancel(ctx, id, userID)`，包含状态检查和回滚分发逻辑。
- [x] 2.2 在 `enrollment_handler.go` 中接入 `POST /api/v1/enrollments/{id}/cancel`。

## 3. 前端集成
- [ ] 3.1 为 `/enrollments/{id}/cancel` 定义前端 API 封装。
- [ ] 3.2 为 `QUEUING` 状态卡片增加“取消”按钮。
- [ ] 3.3 为 `PENDING` 订单增加“取消”按钮，并补充确认提示，避免误操作。

## 4. 测试
- [x] 4.1 编写 Go table-test，验证乐观锁可以正确阻止重复取消导致的重复回补。
