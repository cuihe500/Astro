# 仓库证据与规划结论

## 当前创建链路

- `internal/handler/app.go:25`：创建请求只有 `name`、`image`、`replicas`、`port`；`app.go:50` 在 handler 中绑定并校验基础字段。
- `internal/service/app.go:48`：service 请求仍是四个业务字段加项目/用户 ID；`app.go:58` 校验项目归属和重名后，在项目行锁事务中创建数据库记录与 Kubernetes 资源。
- `internal/model/model.go:52`：`App` 只保存名称、镜像、副本数、状态和项目归属，没有可恢复的 Pod 配置。
- `internal/k8s/adapter.go:18`：`AppSpec` 只有名称、命名空间、镜像、副本数、单端口和标签；`adapter.go:102` 创建单容器 Deployment，并在单端口大于零时创建同名 Service。
- `internal/repository/db.go:29`：数据库迁移使用 GORM `AutoMigrate`，没有独立迁移框架。

## 现有一致性与测试边界

- `internal/service/app_test.go:63` 与 `:98` 已覆盖 Kubernetes 创建失败和数据库提交失败后的补偿；新参数不能绕过该事务/补偿路径。
- `internal/k8s/adapter_test.go:35` 已覆盖 Service 创建失败后删除 Deployment；多端口仍只创建一个 Service，可复用同一补偿语义。
- `internal/handler/routes_test.go:51` 已证明 handler 是参数错误的统一入口，应在此补严格 JSON、请求大小和跨字段校验。
- `pkg/errcode/code.go` 已有 `ErrBadRequest`、`ErrAppCreateFailed`、`ErrDatabase` 和 `ErrK8sOperation`；当前范围无需新增错误码，只需提供不含 Secret 的明确消息。

## Web 现状

- `web/src/features/apps/types.ts:1` 与 `:12` 只有基础 App/CreateApp 类型。
- `web/src/features/apps/pages/CreateAppPage.tsx:17` 是单页受控表单，已有可见标签、字段级错误和提交状态。
- `web/src/features/apps/pages/AppDetailPage.tsx:22` 只读展示基础信息、生命周期操作和日志。
- `web/src/features/apps/api.test.ts:15` 已有响应边界解析测试；新增 `config` 必须在中央解析，页面不得自行断言原始 JSON。
- 前端现有 React/Vitest/Lucide 足够；不需要新增表单、状态或 UI 运行时依赖。

## 已验证的项目约束

- 分层固定为 `handler → service → repository / k8s`；权限由 service 以 Project 所有权保证，Namespace 不从请求读取。
- 列表与详情都使用统一 `Response{code,message,data}`，错误码不得硬编码。
- 所有命令通过 Makefile；后端检查为 `make fmt/test/lint/build/swagger`，前端检查为 `make frontend-check`。
- 本机测试配置已被 Git 忽略；可以用于测试，但任何具体值不得写入任务、日志或提交。
- Git 历史和归档任务中没有已实现的 Pod 参数契约可复用。

## 技术结论

- 用一个受控 `AppConfig` 作为 API、持久化和 Kubernetes 映射的唯一领域契约；不复制 `corev1.PodSpec`，不增加通用资源抽象。
- `apps` 增加可空 JSON 配置列并由 GORM 内置 JSON serializer 管理；旧记录按空配置读取，不拆参数表、不引入依赖。
- 旧 `port` 在 handler 归一化为新端口结构，后续层只处理一个模型。
- 引用预检在数据库/Kubernetes 创建前执行。Secret 必须通过 client-go metadata client 请求 `PartialObjectMetadata`，代码不得接收或记录 Secret data；Kubernetes RBAC 的 `get secrets` 本身仍是高权限，部署侧必须保持最小作用域。
- 高级表单使用原生 `<details>/<summary>` 渐进披露，按语义 `fieldset/legend` 分组；复用现有 CSS token 和响应式布局。

## UI 验收依据

依据本次使用的 `ui-ux-pro-max` 表单与可访问性规则：

- 默认只呈现基础字段，高级字段按需展开；
- 每个输入有可见 label，复杂字段有持久帮助文本；
- 错误紧邻字段，多错误提交后聚焦错误摘要或首个无效字段；
- 使用原生控件和键盘可操作的增删按钮，不依赖 hover 或拖拽；
- 375px 宽度下无横向滚动，focus 样式与 reduced-motion 行为沿用现有全局规则。
