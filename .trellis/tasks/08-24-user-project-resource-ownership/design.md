# 引入用户项目与项目级资源归属：技术设计

## 设计目标

在现有 `handler → service → repository / k8s` 分层内增加项目这一归属边界，使数据库和所有应用操作都无法绕过项目，同时保持当前“一个应用对应 Deployment，可选 Service”的模型不变。

## 数据模型

### Project

在 `internal/model/model.go` 增加 `Project`：

| 字段 | 约束 | 用途 |
|---|---|---|
| `ID` / 时间字段 | 复用 `BaseModel` | 内部主键与审计时间 |
| `Name` | `not null`，与 `UserID` 组成唯一索引 | 用户可见项目名 |
| `UserID` | `not null`、索引、外键 | 项目所有者 |
| `Namespace` | `not null`、全局唯一索引 | 稳定的 Kubernetes Namespace |

`App` 改为只保存非空 `ProjectID`，并与 `Name` 组成唯一索引；删除 `UserID` 和 `Namespace`。应用所属用户及 Namespace 统一通过 Project 获取，避免同一事实存在多份可漂移副本。

数据库外键采用限制删除：Project 有 App 时由数据库约束和业务检查共同阻止删除；App 不允许引用不存在的 Project。由于当前只有测试数据，本次不引入迁移框架或兼容字段。

### Namespace 命名

创建项目前使用仓库已有的 `google/uuid` 生成 `astro-project-<UUID>`，并把完整结果存入 Project。该名称不超过 63 字符且符合 Kubernetes DNS label；不依赖用户输入，也无需增加公开 UUID 字段或命名配置。数据库唯一约束是最终防线，Namespace 在项目生命周期内不变。

## 后端边界与数据流

### 项目创建

```text
handler 校验并清理名称
  → service 检查当前用户内重名并生成 Namespace
  → K8s EnsureNamespace
  → repository 创建 Project
  → 成功返回 Project
```

`Project` 的记录创建与 Namespace 建立不能跨数据库/Kubernetes 做原子事务。先创建 Namespace，保证数据库中不会出现缺少 Namespace 的可用项目；若数据库写入失败，则删除刚创建的空 Namespace。若补偿也失败，返回包含两次失败的系统错误并记录 Namespace，不能用 `_` 丢弃错误。`EnsureNamespace` 与删除均保持幂等，以便安全重试。

### 项目删除

```text
handler 校验 project_id
  → service 在数据库事务内锁定 project_id + user_id 对应项目
  → repository 统计项目内 App，非空则拒绝
  → K8s 删除 Namespace（NotFound 视为成功）
  → repository 硬删除空 Project
```

应用创建也锁定同一 Project 行，防止“检查为空”后并发插入应用。项目删除事务会短暂跨越一次 Kubernetes 删除请求；这是当前低并发 MVP 为避免额外状态机采用的最小一致性方案。先删 Namespace，避免数据库先消失后留下无法从产品内追踪的 Kubernetes 资源。若 Namespace 删除成功但数据库提交失败，项目仍可再次删除，Namespace NotFound 按成功处理。删除接口成功只表示 Kubernetes 已接受 Namespace 删除；不轮询 Namespace 的异步终止。

### 应用操作

所有应用 service 方法接收 `projectID` 与 `userID`。共用的查询路径以 `project_id + app_id` 查 App，并校验所属 Project 的 `user_id`；列表以 `project_id` 过滤，重名检查使用 `project_id + name`。Kubernetes 调用只使用项目记录中的 Namespace。

创建应用时在事务内锁定并确认项目归属，写入带非空 `ProjectID` 的 App 后，通过 repository 的提交前回调调用现有 K8s `CreateApp`。K8s 创建失败时事务直接回滚，因此其他请求看不到尚未创建资源的 App；K8s 创建成功后才提交事务并暴露 App。由于一次失败的 K8s 请求也可能已创建部分资源，service 从调用开始即标记资源可能被触碰；此后只要事务失败，都调用幂等的 K8s `DeleteApp` 补偿，资源不存在时 NotFound 视为成功。补偿失败时返回同时保留原始创建/提交错误和清理错误的中性文案并记录结构化上下文。该事务会跨越一次 Kubernetes 创建请求，以最小改动同时关闭并发删除和孤儿资源竞态，不引入状态机。项目和应用记录均采用硬删除，使数据库唯一约束不会阻止用户删除后复用名称。现有启停、重启、状态同步、日志和删除逻辑只替换资源定位参数，不新增通用 Kubernetes 资源抽象。

## Kubernetes 适配器

在既有 `AppAdapter` 上增加 `DeleteNamespace(ctx, namespace)`，复用当前 client-go Client；Namespace 不存在视为成功。项目创建继续复用 `EnsureNamespace`。

应用创建不再负责建立 Namespace：Namespace 是项目创建成功的必要条件。`CreateApp` 只在已存在的项目 Namespace 内创建 Deployment/Service，避免应用调用掩盖损坏的项目状态。

Project Namespace 继续使用现有 `managed-by=astro` 标签，便于定点盘点；不引入通用标签构建器。

## API 契约

全部接口位于认证组：

| 方法 | 路径 | 行为 |
|---|---|---|
| `POST` | `/api/v1/projects` | 创建项目 |
| `GET` | `/api/v1/projects` | 当前用户项目列表 |
| `GET` | `/api/v1/projects/:project_id` | 当前用户项目详情 |
| `DELETE` | `/api/v1/projects/:project_id` | 删除空项目及 Namespace |
| `POST` | `/api/v1/projects/:project_id/apps` | 在项目内创建应用 |
| `GET` | `/api/v1/projects/:project_id/apps` | 项目内应用列表 |
| `GET` | `/api/v1/projects/:project_id/apps/:id` | 应用详情 |
| `DELETE` | `/api/v1/projects/:project_id/apps/:id` | 删除应用 |
| `POST` | `/api/v1/projects/:project_id/apps/:id/{start,stop,restart}` | 生命周期操作 |
| `GET` | `/api/v1/projects/:project_id/apps/:id/logs` | 日志 |

handler 统一校验正整数路径参数和请求体。不存在的自身资源返回项目/应用不存在；已确认属于其他用户的资源返回 `ErrForbidden`，沿用当前越权语义。新增 `22xxx` 项目错误：项目不存在、重名、非空和创建失败；删除中的 K8s/数据库错误继续复用现有系统错误码。

旧 `/api/v1/apps` 路由直接删除，不保留重定向或兼容 handler。

## Web 路由与交互

采用可深链的项目路径：

- `/projects`：项目列表；无项目时只展示“创建第一个项目”引导。
- `/projects/new`：创建项目。
- `/projects/:projectId/apps`：项目内应用列表。
- `/projects/:projectId/apps/new`：创建应用。
- `/projects/:projectId/apps/:id`：应用详情和生命周期管理。

登录、注册与 OAuth 回调成功后进入 `/projects`。顶部导航改为“项目”，项目内页面通过标题/返回链接保持项目上下文；不增加全局项目选择器或额外状态管理。项目卡片进入应用列表，删除仅在项目列表提供。

创建表单保留可见 label、字段级错误、提交中禁用。删除项目使用原生 `<dialog>` 二次确认，明确“仅空项目可删除”；确认和取消均可键盘操作，图标复用 Lucide 并保持可访问名称。现有响应式布局、颜色 token、反馈组件和 CSS 体系继续复用，不加依赖或新设计系统。

## 破坏性切换与清理

实施前先通过只读命令盘点数据库 App 记录以及 `managed-by=astro` 的旧 `astro-user-*` Namespace。经用户再次确认准确清单后，才允许定点删除这些测试 Kubernetes 资源与 App 数据；不使用宽泛 selector、通配符或 `docker compose down`。

清理通过现有应用删除流程逐项完成，确保旧 Deployment/Service 先于数据库活动记录消失；旧 `astro-user-*` 空 Namespace 再按盘点清单定点删除。所有外部命令必须先有对应 Makefile 入口，且删除前再次展示解析后的精确名称。

新版本启动时执行一个窄范围 schema gate：若检测到旧 `apps.user_id` 列且仍有活动 App，启动失败并提示先完成清理；确认无活动 App 后才删除并由 `AutoMigrate(Project, App, ...)` 重建空 `apps` 表。这样不会依赖 `AutoMigrate` 自动删除旧 `user_id`/`namespace` 列，也不会把存量应用偷偷迁为默认项目。该 gate 只匹配明确的旧表结构，不清理用户、OAuth 身份或其他表。

回滚代码前提是没有在新模型创建需保留的数据；若已有新项目/应用，先停止回滚并人工决定数据处置。代码回滚不会自动恢复已删除的测试资源。

## 测试策略

- repository/model：验证用户内项目唯一、Namespace 全局唯一、App 的非空有效 Project 外键、项目内应用唯一和限制删除。
- service：使用最小 fake adapter 覆盖项目创建补偿、越权、非空删除、Namespace 删除失败、App 提交前不可见、K8s 部分创建失败及提交失败后的 Kubernetes 补偿，以及所有 App 操作的项目边界。
- handler/router：验证路径参数与请求校验、嵌套路由存在且旧 `/apps` 不存在。
- Web：验证项目响应解码、路由构造、无项目空状态、创建/删除交互及项目上下文传播。
- 全量执行 Makefile 现有的后端、前端、Swagger、构建与 lint 校验。

## 明确不做

不新增项目编辑、成员/RBAC、默认项目、级联删除、旧 API 兼容、独立 Service/Ingress CRUD、通用资源仓库、迁移框架或前端全局状态库。
