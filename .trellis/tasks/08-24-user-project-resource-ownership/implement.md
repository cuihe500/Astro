# 引入用户项目与项目级资源归属：实施计划

## 实施门禁

- [x] 用户批准最新的 `prd.md`、`design.md` 与本计划。
- [x] GitHub Issue #3 仍为 OPEN，且 Astro Development 中 Work Type=`Feature`、Priority=`P2`、Assignee=`cuihe500`。
- [x] 在 Project 填写 Start date=`2026-08-24`、Trellis Task=`.trellis/tasks/08-24-user-project-resource-ownership`，并将 Status 改为 `In Progress`。
- [x] 运行 `make trellis TRELLIS_ARGS='start .trellis/tasks/08-24-user-project-resource-ownership'` 后再修改产品代码、文档或外部资源。

## 实施步骤

### 1. 建立最小后端模型与项目能力

- [x] 在 `model.go` 增加 Project，并将 App 改为必填 `ProjectID`；以数据库唯一索引和外键落实项目/应用归属约束。
- [x] 增加窄范围 schema gate：旧 App 仍存在时拒绝启动，已清空时重建 `apps` 表；更新 `AutoMigrate` 列表、Project repository，并把 App repository 查询改为项目作用域。
- [x] 在 K8s adapter 增加 Namespace 删除，保持 NotFound 幂等；项目 Namespace 使用 `astro-project-<UUID>`。
- [x] 增加 Project service/handler：创建、列表、详情、禁止非空删除及失败补偿；项目删除和应用创建锁定同一 Project 行以关闭并发竞态；新增 `22xxx` 错误码。
- [x] 注册 `/projects` 路由，并为新增 service 分支补最小测试。

### 2. 强制应用走项目边界

- [x] 将所有 App service 方法改为接收并校验 `projectID + userID`，Namespace 只从 Project 获取。
- [x] 将 App handler 路由和 Swagger 注解改为 `/projects/:project_id/apps...`，删除旧 `/apps` 路由。
- [x] 修正现有被改动路径中的忽略错误，尤其是数据库/K8s 创建补偿和状态写回；为越权、错项目、项目内重名及旧路由缺失补测试。
- [x] 将 K8s 应用创建移入 `CreateInProject` 的事务回调，提交前不暴露 App；K8s 调用开始后的任何事务失败均幂等清理可能残留的资源，并补最小回归测试。

### 3. 完成 Web 项目闭环

- [x] 增加最小 Project 类型/API，以及项目列表和创建页面；列表提供进入项目与删除空项目操作。
- [x] 将应用 API 与三个页面改为携带 `projectId`，路由迁至 `/projects/:projectId/apps...`。
- [x] 将登录后入口、品牌链接和导航改为 `/projects`；无项目时展示创建项目引导。
- [x] 复用现有 Feedback、原生 dialog、Lucide 和 CSS token，补齐加载/错误/确认、键盘可达与移动端布局测试，不添加依赖。

### 4. 同步文档并验证破坏性切换

- [x] 更新 `docs/architecture-design.md` 的数据模型、Namespace 架构、API 和 Web 流程；运行 `make swagger` 更新生成文档。
- [x] 更新受本次实现影响的 `.trellis/spec/backend/` 真实约定。
- [ ] 通过 Makefile 入口只读盘点测试数据库旧 App 与带 `managed-by=astro` 标签的旧 `astro-user-*` Namespace，向用户展示精确清单并再次取得删除确认。
- [ ] 先经现有应用删除流程逐项清理获批的测试 App，再定点删除清单中的空旧 Namespace；在测试环境部署后验证 schema gate 已重建 `apps` 表、没有孤儿 App，且每个 Project Namespace 与记录一致。

当前阻塞：`make legacy-inventory` 已实现并确认会在缺少依赖时失败关闭；当前环境缺少 `mariadb/mysql` 客户端，因此尚未连接测试数据库或 Kubernetes 获取清单，也未执行任何删除。

## 验证命令

所有命令从 Makefile 入口执行：

```bash
make test
make lint
make build
make frontend-check
make swagger
make governance-check TASK=.trellis/tasks/08-24-user-project-resource-ownership
```

Swagger 生成后再次执行 `make git GIT_ARGS='grep -n --untracked /api/v1/apps -- docs internal web/src'`，结果只允许出现在验证旧路由缺失的测试断言中；并检查所有新应用路由均包含 `projects/:project_id`。

## 手工验收矩阵

- [ ] 两个用户可创建同名项目；同一用户重名失败；每个项目 Namespace 唯一。
- [ ] 无项目用户无法创建应用，Web 只提供创建项目入口。
- [ ] 同项目应用重名失败，跨项目同名成功。
- [ ] 伪造其他用户的 `project_id`，项目与所有 App 操作均失败。
- [ ] 非空项目删除失败且资源不变；空项目删除后记录消失，Namespace 进入删除/不存在状态。
- [ ] 旧 `/api/v1/apps` 与旧 `/apps` Web 路由不可用。
- [ ] Web 可完成项目创建 → 进入项目 → 创建应用 → 查看/启停/重启/日志/删除应用 → 删除空项目。

## 回滚点

- 步骤 1–3 未触碰测试数据或 K8s 时，可直接回退代码。
- 步骤 4 清理前必须保存精确盘点；清理后的测试资源不承诺恢复。
- 新模型产生需保留数据后，禁止直接回滚到旧模型，应先停机并制定数据处置方案。

## 范围控制

不引入迁移框架、项目编辑、默认项目、通用 Kubernetes 资源抽象、独立 Service/Ingress 管理、旧路由兼容层或前端状态库。
