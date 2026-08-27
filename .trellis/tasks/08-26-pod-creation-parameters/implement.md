# 扩展应用创建的 Pod 参数支持：实施计划

## 实施门禁

- [ ] 用户批准最新 `prd.md`、`design.md` 与本计划。
- [ ] Issue #7 保持 OPEN，Project 字段 Work Type=`Feature`、Priority=`P2`、Assignee=`cuihe500`。
- [ ] 在 Project 填写 Start date=`2026-08-26`、Trellis Task=`.trellis/tasks/08-26-pod-creation-parameters`，并将 Status 改为 `In Progress`。
- [ ] 运行 `make trellis TRELLIS_ARGS='start .trellis/tasks/08-26-pod-creation-parameters'` 后再修改产品代码、文档或部署状态。

## 实施步骤

### 1. 建立唯一配置契约与严格请求边界

- [ ] 在 `internal/model` 增加最小 `AppConfig` 领域结构，并在 `App` 增加 GORM JSON serializer 字段；用 AutoMigrate/集成测试验证旧 NULL 配置和新配置 round-trip。
- [ ] 扩展创建请求和详情响应；实现 64 KiB、未知字段、尾随 JSON、数量、格式、one-of、重复和跨字段校验。
- [ ] 将旧 `port` 归一化为唯一 `ports` 模型；后续 service/k8s 不再维护第二套端口逻辑。
- [ ] 为 handler 校验补表驱动测试，至少覆盖合法最小请求、完整配置、旧端口兼容、未知/超限/冲突/高风险输入。

### 2. 完成资源预检与 Kubernetes 映射

- [ ] 初始化 client-go metadata client，并在既有 `AppAdapter` 增加引用预检；按 kind/name 去重，只读取 PartialObjectMetadata。
- [ ] service 在项目权限和重名检查后、数据库事务前执行预检；不存在/无权限返回清晰 `ErrAppCreateFailed`，不触碰数据库或工作负载。
- [ ] 用纯构建函数映射 Deployment 与可选单个 ClusterIP Service，覆盖环境、资源、端口、三类探针、卷/挂载、安全上下文、终止宽限期和 imagePullSecrets。
- [ ] 保持 Service 失败清理 Deployment、事务失败幂等 DeleteApp 和取消请求后的补偿语义；扩展 fake client/service 测试。

### 3. 完成持久化读取与 Web 闭环

- [ ] 创建时保存归一化配置；创建/详情响应显式返回 `config`，应用列表保持当前摘要响应。
- [ ] 在 `web/src/features/apps` 增加中央 AppConfig 类型、响应解析与 draft → request 校验，不在页面散落原始 JSON cast。
- [ ] 创建页用原生 `<details>` 和语义分组实现高级配置；动态项提供键盘可达的添加/删除，错误就近显示并聚焦首错，基础四字段路径保持不变。
- [ ] 详情页按相同分组只读展示非空配置，Secret 仅展示引用；补 API 解析和关键创建/详情页面测试。
- [ ] 复用现有 Lucide、CSS token、Feedback 与响应式规则，不增加运行时依赖；验证 375px 无横向滚动。

### 4. 文档、跨层核对与完整验证

- [ ] 更新 handler Swagger 注解并运行 `make swagger`，同步架构文档与受影响的项目规范。
- [ ] 核对 `请求 → 归一化 → JSON 存储 → 读取 → Web 展示` 无字段丢失，旧 `port` 只在入口存在，Secret data 不进入日志/响应/数据库。
- [ ] 核对预检后竞态、Kubernetes 准入失败、Service 创建失败、事务提交失败均保留现有补偿和明确错误。
- [ ] 执行全部验证命令并记录真实结果；若本机隔离测试环境可用，再执行集成测试和最小真实 Kubernetes 烟测，不输出本机配置值。

## 验证命令

只使用 Makefile 已有入口：

```bash
make fmt
make test
make test-integration
make lint
make build
make frontend-check
make swagger
make governance-check TASK=.trellis/tasks/08-26-pod-creation-parameters
```

`make test-integration` 仅在已配置的隔离空 MariaDB 且满足目标门禁时运行；否则明确报告未运行。真实 Kubernetes 烟测只使用已忽略的 `configs/config.local.yaml`，不得输出其值。

## 手工验收矩阵

- [ ] 旧四字段请求创建 Deployment/Service，旧 Web 简单路径无回归。
- [ ] 一个完整白名单请求能创建预期 Deployment 和单个多端口 ClusterIP Service，详情往返字段一致。
- [ ] 无 Service 端口时只创建 Deployment；TCP/UDP 与不同 target/service 端口映射正确。
- [ ] 允许的 PVC/ConfigMap/Secret/imagePullSecret 存在时通过；缺失或无权限时无数据库/Kubernetes 残留且提示可修复。
- [ ] 未知字段、原生 PodSpec、高风险字段、重复/冲突、超限、非法 Quantity/路径/探针在 handler 被拒绝。
- [ ] Service 创建、Kubernetes 准入和数据库提交失败均完成补偿；日志与响应不含 Secret data。
- [ ] Web 默认折叠高级配置，键盘可完成增删/提交，错误聚焦正确，详情只读分组在 375px 下无横向滚动。

## 回滚点

- 代码回滚可保留新增的可空 JSON 列，旧代码会忽略它；不要破坏性删列。
- 已创建的高级配置工作负载不自动删除；回滚后只能继续运行/按现有生命周期管理，无法从旧 UI 查看配置。
- 本任务不直接更改外部 RBAC、ResourceQuota 或 LimitRange；部署前置不满足时停止真实环境验收并报告。

## 范围控制

不引入原生 PodSpec/YAML、多容器、配置编辑、PVC/ConfigMap/Secret CRUD、Ingress/外部 Service、高级调度、迁移框架、表单库、状态库或通用 Kubernetes 资源抽象。
