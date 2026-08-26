# 应用 Pod 创建契约

> Astro 只开放受控的单容器配置，不接收 Kubernetes 原生 `PodSpec` 或任意 YAML。

## 1. Scope / Trigger

- 修改应用创建请求、`model.AppConfig`、`apps.config`、Kubernetes Deployment/Service 映射、引用资源预检或应用详情响应时适用。
- 配置只在创建时写入；当前没有编辑接口，也不负责创建 PVC、ConfigMap 或 Secret。

## 2. Signatures

- API：`POST /api/v1/projects/:project_id/apps`。
- 列表/详情：`GET /api/v1/projects/:project_id/apps`、`GET /api/v1/projects/:project_id/apps/:id`。
- handler：`decodeAndValidateCreateAppRequest(io.Reader) (CreateAppRequest, error)`。
- service：`CreateApp(context.Context, service.CreateAppRequest) (*model.App, error)`。
- K8s：`ValidateAppReferences(context.Context, namespace, model.AppConfig) error`、`CreateApp(context.Context, k8s.AppSpec) error`。
- DB：`model.App.Config model.AppConfig` 使用 `gorm:"serializer:json;type:json"` 保存到 `apps.config`。

## 3. Contracts

- 创建请求保留 `name`、`image`、`replicas` 和可选旧字段 `port`，并增加可选 `config`。
- `config` 只允许：`command`、`args`、`working_dir`、`image_pull_policy`、`env`、`env_from`、`resources`、`ports`、三类 probe、`volumes`、`volume_mounts`、`security_context`、`termination_grace_period_seconds`、`image_pull_secrets`。
- `port` 为正数时归一化为名为 `default` 的 TCP 容器端口和同值 Service 端口；为兼容旧 Web，显式 `port: 0` 等同未提交，但 Swagger 不把 `0` 作为新客户端可用值公开。`port` 与 `config.ports` 不能同时提供有效端口配置。
- `config.ports` 中仅有 `service_port` 的项进入同一个 ClusterIP Service；没有 `service_port` 时只创建 Deployment。
- PVC、ConfigMap、Secret 和 imagePullSecret 必须引用已授权项目 Namespace 中的现有资源。预检只通过 metadata client 读取 `PartialObjectMetadata`，不得读取、保存或记录 Secret 内容。
- 创建与详情响应使用 `AppDetailResponse` 显式返回 `config`；列表仍返回 `[]model.App`，不得包含 `config`。
- 配置只有一份领域结构，沿 `handler -> service -> model/GORM -> k8s adapter` 传递；禁止另建同构 DTO 或直接暴露 client-go 类型。

## 4. Validation & Error Matrix

| 条件 | 结果 |
|---|---|
| 请求体超过 64 KiB、未知字段、尾随 JSON、格式/范围/数量错误 | handler 返回 `ErrBadRequest`，不进入 service |
| `port` 为 `0` | 视为未配置，仍继续校验全部 `config` |
| `port` 为负数、超过 65535，或有效旧端口与 `config.ports` 并存 | `ErrBadRequest` |
| 环境变量/引用 one-of、卷来源 one-of、探针 handler one-of 不成立 | `ErrBadRequest` |
| Quantity 非正、request 大于 limit、端口/名称/路径重复或非法 | `ErrBadRequest` |
| 请求特权提升、自定义 seccomp、capability add、hostPath 或其他白名单外字段 | `ErrBadRequest` |
| 项目不存在或不属于当前用户 | 既有 `ErrProjectNotFound` / `ErrForbidden`，不得预检其他 Namespace |
| 引用资源不存在或不可读取 | `ErrAppCreateFailed`，不写数据库、不创建工作负载 |
| Deployment、Service 或事务失败 | 返回既有应用/数据库错误，并按现有补偿路径幂等清理 |

## 5. Good / Base / Bad Cases

- Good：完整白名单配置经过 handler 归一化，JSON 往返后映射为一个 Deployment 和至多一个多端口 ClusterIP Service，详情返回相同配置。
- Base：只提交基础字段，或旧客户端提交 `port: 0`，得到无端口的既有单容器 Deployment 行为。
- Bad：传入原生 `pod_spec`、Secret 数据、`hostPath`、多容器或 NodePort；严格 JSON 边界必须拒绝，不能透传到 Kubernetes。

## 6. Tests Required

- handler：断言 64 KiB、未知/尾随 JSON、所有 one-of、数量/格式/重复/安全边界；特别覆盖 `port: 0` 兼容且不能绕过高级配置校验。
- service：断言项目授权先于引用预检，预检失败不触碰数据库/Kubernetes，创建或提交失败执行补偿。
- adapter：逐字段断言 Deployment/Service 映射、metadata 引用去重、Secret 只使用 metadata，以及 Service 失败清理 Deployment。
- repository/MariaDB：在隔离空库中断言 `apps.config` 自动迁移并可 JSON round-trip；没有隔离测试环境时必须明确跳过，不能复用业务库。
- API/Web：断言列表无 `config`、创建与详情有 `config`，中央解析拒绝畸形响应，创建页基础路径和高级配置详情可用。
- 修改契约后至少运行 `make fmt`、`make swagger`、`make test`、`make lint`、`make build`、`make frontend-check`。

## 7. Wrong vs Correct

### Wrong

```go
// 直接接受原生类型会把高风险字段和 client-go 版本一起暴露为公开契约。
type CreateAppRequest struct {
	PodSpec corev1.PodSpec `json:"pod_spec"`
}
```

### Correct

```go
// 公开契约只复用经过 handler 严格校验的领域白名单。
type CreateAppRequest struct {
	Name     string          `json:"name"`
	Image    string          `json:"image"`
	Replicas int             `json:"replicas"`
	Port     *int32          `json:"port,omitempty"`
	Config   model.AppConfig `json:"config,omitempty"`
}
```

这样可从类型层阻断未授权 Kubernetes 能力，并让存储、响应和适配器共享同一份受控配置。
