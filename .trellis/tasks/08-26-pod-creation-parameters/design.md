# 扩展应用创建的 Pod 参数支持：技术设计

## 设计目标

在现有单容器 Deployment 模型内增加常用 Pod 配置，同时保持旧请求、项目隔离、事务补偿和 C 端简单体验。方案只增加一个受控领域契约，不暴露 `corev1.PodSpec`，不引入新依赖、通用 Kubernetes 资源层或配置编辑状态机。

## API 契约

创建入口仍为 `POST /api/v1/projects/:project_id/apps`。基础字段保持不变，新增可选 `config`：

```json
{
  "name": "demo",
  "image": "nginx:latest",
  "replicas": 1,
  "config": {
    "command": ["/bin/sh"],
    "args": ["-c", "nginx -g 'daemon off;'"],
    "working_dir": "/app",
    "image_pull_policy": "IfNotPresent",
    "env": [
      {"name": "MODE", "value": "production"},
      {"name": "PASSWORD", "value_from": {"secret_key_ref": {"name": "demo-secret", "key": "password"}}}
    ],
    "env_from": [
      {"prefix": "APP_", "config_map_ref": {"name": "demo-config"}}
    ],
    "resources": {
      "requests": {"cpu": "100m", "memory": "128Mi"},
      "limits": {"cpu": "500m", "memory": "512Mi"}
    },
    "ports": [
      {"name": "http", "container_port": 8080, "protocol": "TCP", "service_port": 80}
    ],
    "readiness_probe": {
      "http_get": {"path": "/ready", "port": 8080, "scheme": "HTTP"},
      "period_seconds": 10,
      "failure_threshold": 3
    },
    "volumes": [
      {"name": "cache", "empty_dir": {}},
      {"name": "data", "persistent_volume_claim": {"claim_name": "demo-data", "read_only": false}}
    ],
    "volume_mounts": [
      {"name": "data", "mount_path": "/data"}
    ],
    "security_context": {
      "run_as_non_root": true,
      "run_as_user": 1000,
      "run_as_group": 1000,
      "fs_group": 1000,
      "read_only_root_filesystem": true,
      "allow_privilege_escalation": false,
      "drop_capabilities": ["ALL"],
      "seccomp_profile": "RuntimeDefault"
    },
    "termination_grace_period_seconds": 30,
    "image_pull_secrets": ["registry-secret"]
  }
}
```

### 字段形状

| 字段 | 契约 |
|---|---|
| `command` / `args` | 字符串数组；不 trim 单项内容，保持容器执行语义 |
| `working_dir` | 可选绝对 POSIX 路径 |
| `image_pull_policy` | `Always`、`IfNotPresent`、`Never`；缺失时交给 Kubernetes 默认化 |
| `env[]` | `name` 加 `value` 或 `value_from` 二选一；`value_from` 内部再从 `config_map_key_ref`、`secret_key_ref` 二选一；`value` 用指针语义以保留显式空字符串 |
| `env_from[]` | 可选 `prefix`，加 `config_map_ref`、`secret_ref` 二选一 |
| key/ref | 只含资源 `name` 与必要的 `key`；不开放 optional，创建前必须验证资源存在 |
| `resources` | requests/limits 各自只含可选 `cpu`、`memory` Quantity |
| `ports[]` | 唯一 `name`、`container_port`、可选 `protocol` 与 `service_port` |
| 三类 probe | 各自可选；`http_get`、`tcp_socket`、`exec` 三选一，加标准时间/阈值字段 |
| HTTP probe | 数字端口、绝对 path、`HTTP`/`HTTPS` scheme、最多 20 个 header；不开放自定义 host |
| TCP probe | 数字端口 |
| Exec probe | 非空 command 数组 |
| `volumes[]` | `empty_dir`、`persistent_volume_claim`、`config_map`、`secret` 四选一 |
| `volume_mounts[]` | `name`、绝对 `mount_path`、可选相对 `sub_path`、`read_only` |
| `security_context` | 仅 PRD 白名单；`fs_group` 映射 PodSecurityContext，其余映射容器 SecurityContext |
| `image_pull_secrets` | Secret 名称数组 |

创建和详情成功响应新增 `config`。应用列表继续返回原有摘要字段，不携带配置，避免列表响应膨胀及普通环境变量值的无关扩散。Secret 引用在任何响应中都只有资源名和键。

### 兼容与归一化

- 请求体最大 64 KiB，并对当前创建请求启用严格 JSON 解码：未知字段、多段 JSON 和超限请求均返回 `ErrBadRequest`。
- 旧 `port` 为正数时仍须在 `1..65535`；若存在，将其归一化为名称 `default`、协议 TCP、容器端口与 Service 端口相同的新端口项。为兼容旧 Web，显式 `port: 0` 等同未提交，但不作为新客户端的公开合法值。
- `port` 与 `config.ports` 同时出现时拒绝，避免两个事实源。
- 新端口协议缺失时归一化为 TCP；HTTP probe scheme 缺失时归一化为 HTTP；可选数组缺失时按空数组处理。
- 旧数据库记录的配置按 `{}` 返回；旧客户端忽略新增响应字段，旧请求不填写 `config` 时行为不变。

## 校验契约

handler 负责全部信任边界校验，service 与 adapter 只接收已归一化的 `model.AppConfig`。

| 类别 | 规则 |
|---|---|
| 请求 | 最大 64 KiB；拒绝未知字段和尾随 JSON |
| 数组 | env/envFrom 合计 ≤100；ports ≤20；volumes/volumeMounts 各 ≤20；imagePullSecrets ≤10；command ≤20；args ≤100；每个探针 Exec/header ≤20；drop capabilities ≤20 |
| Kubernetes 名称 | 复用 apimachinery validation；端口名按 Kubernetes 有效端口名，卷名按 DNS label，引用名按 DNS subdomain，环境变量名/前缀按原生规则 |
| 端口 | `1..65535`；协议仅 TCP/UDP；名称、容器端口+协议、Service 端口+协议分别唯一 |
| 资源 | Quantity 必须可解析且大于 0；同一资源同时有 request/limit 时 request ≤ limit；不设置容量上限 |
| 路径 | workingDir/mountPath/HTTP path 必须为绝对 POSIX 路径；subPath 必须相对且不能含 `..`；单路径最大 4096 字符 |
| 探针 | 恰好一个 handler；initialDelay `0..3600`；period/timeout 为 `0`（使用默认值）或 `1..3600`；success 为 `0` 或 `1..10`，startup/liveness 显式值只能为 1；failure 为 `0` 或 `1..60` |
| 安全 ID | runAsUser/runAsGroup/fsGroup 为 `0..4294967295`；allowPrivilegeEscalation 若出现只能为 false；seccomp 只能 RuntimeDefault；drop 名称必须唯一且符合 Linux capability 名称格式 |
| Pod 终止 | 缺失使用 Kubernetes 默认值；显式值为 `0..300` 秒 |
| 卷 | 名称唯一；每项恰好一个来源；mount 名称必须引用已声明卷；mountPath 唯一；emptyDir medium 仅空值/Memory，sizeLimit 为正 Quantity |
| 引用 | 相同种类/名称先去重；按请求出现顺序报告首个失败资源；不校验 Secret key 内容 |

纯格式错误使用 `ErrBadRequest`。引用不存在、无读取权限或 Kubernetes 准入/创建失败继续使用现有 `ErrAppCreateFailed` 并带可修复但不含 Secret 内容的消息；数据库失败继续使用 `ErrDatabase`。本次不增加重复错误码。

## 数据模型与分层

在 `internal/model` 定义唯一的 `AppConfig` 及其嵌套领域类型。它们只表达白名单字段，不引用 Kubernetes 类型。`App` 增加：

```go
Config AppConfig `gorm:"serializer:json;type:json" json:"-"`
```

GORM 内置 JSON serializer 负责 MariaDB JSON 列，不增加 `gorm.io/datatypes`。`AutoMigrate` 为现有表增加可空列；读取 NULL 得到零值配置。配置字段默认不随 `model.App` 序列化，由 handler 的详情响应 DTO 显式加入；列表沿用当前形状。

数据流只有一份配置：

```text
严格 JSON + handler 校验/归一化
  → service.CreateAppRequest.Config
  → adapter.ValidateAppReferences(namespace, config)
  → model.App.Config JSON 持久化
  → k8s.AppSpec.Config
  → Deployment/Service 构建

数据库 App.Config
  → service.GetApp
  → handler AppDetailResponse
  → Web 中央 parseApp
  → 详情分组只读展示
```

不建立“API DTO → service DTO → persistence DTO → k8s DTO”的四套同构结构；handler 的外层请求仍独立，嵌套配置复用领域结构并由专用校验函数把关。

## Kubernetes 预检与资源创建

`AppAdapter` 增加 `ValidateAppReferences(ctx, namespace, config)`。service 顺序为：项目权限 → 应用重名 → 引用预检 → `CreateInProject` 事务与现有 `CreateApp`。预检在项目行锁事务外完成，避免为只读 API 调用延长数据库锁；项目若并发删除，后续 `CreateInProject` 仍会失败。

client 初始化时同时建立 typed client 与 client-go metadata client。PVC、ConfigMap、Secret/imagePullSecret 全部通过 `PartialObjectMetadata` 获取存在性；代码不接收 Secret data。名称先按 kind 去重，避免重复 API 调用。Kubernetes RBAC 无“仅元数据 Secret get”权限粒度，因此部署身份的 Secret `get` 必须限制在 Astro 管理的项目命名空间；应用代码仍只请求 metadata 表示。

预检只能改善即时反馈，不能消除资源在预检后被删除的竞态；Kubernetes 仍是最终事实源。该竞态不引入锁或资源所有权系统。

`internal/k8s` 用纯构建函数把 `AppSpec` 映射为：

- 一个 Deployment、一个容器；固定 selector/managed-by 标签仍由 Astro 生成；
- 仅存在至少一个 `service_port` 时创建一个 ClusterIP Service；ServicePort 名称/协议来自同一端口项，TargetPort 使用容器端口；
- 环境、资源、探针、卷、挂载、安全上下文、termination grace 与 imagePullSecrets 映射到 PodTemplate；
- 不生成任何白名单外字段。

Service 创建失败后的 Deployment 清理、事务失败后的幂等 `DeleteApp` 和 `context.WithoutCancel` 补偿保持现状。

## Web 交互

创建页保留名称、镜像、副本数和旧单端口作为默认基础路径。原生 `<details>/<summary>` 提供“高级配置”渐进披露，内部按容器、环境变量、资源、网络、健康检查、存储、安全七组展示。动态集合使用明确的“添加/删除”按钮，不使用拖拽，不增加表单库或状态库。

前端定义与 API 同形的 `AppConfig`，集中完成 draft → request 的 trim、空值省略和基本校验。后端仍是最终校验者。每个字段有 label/helper/error 关联；提交出现多个客户端错误时展开所属分组并聚焦首个错误，服务端错误聚焦 `role=alert` 的表单反馈。重复行使用稳定的本地 ID，仅在提交时移除 ID。

详情页仅在配置非空时显示“高级配置”只读区，按创建页相同分组展示。普通环境变量显示保存值并提示敏感值应改用 Secret；Secret 只显示引用名/键。长命令、镜像和路径允许换行；375px 宽度不横向滚动。

## 文档、上线与回滚

- 更新 Swagger 创建请求与详情响应，并同步 `docs/architecture-design.md`、必要的 Web 文档。
- 数据库变更是单个可空 JSON 列；旧版本代码会忽略该列，代码回滚无需删列。
- 回滚后已创建的 Deployment/Service 继续由 Kubernetes 运行，但旧 UI/API 无法展示其高级配置；不得自动删除这些工作负载。
- 上线前确认运行身份具备项目命名空间内相关资源的 `get`，并确认每个项目命名空间有适当的 ResourceQuota/LimitRange；本任务不修改外部部署 RBAC 或配额。

## 取舍

- 不拆配置关系表：当前配置只在创建/详情整块读写，JSON 列更短且避免无价值 join。
- 不复用 `corev1.PodSpec` 作为 API/数据库模型：它会暴露高风险字段并把平台契约绑定到 client-go 版本。
- 不做配置编辑：避免不可变字段、Deployment 更新和失败回滚状态机。
- 不新增 UI/表单依赖：现有 React 状态、原生控件和 CSS 足够。
