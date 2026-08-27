# 扩展应用创建的 Pod 参数支持

## Goal

让 C 端用户无需编写 Kubernetes YAML，即可在现有应用创建流程中配置常用的单容器 Pod 参数；Astro 通过受控字段、明确上限与安全默认值保持易用性、租户隔离和集群安全。

## Background

- GitHub 需求事实源：[Issue #7](https://github.com/cuihe500/Astro/issues/7)。
- 现有 `POST /api/v1/projects/:project_id/apps` 仅接收名称、镜像、副本数和单个端口。
- 现有 Kubernetes 适配器创建一个单容器 Deployment；声明端口时另建同名 ClusterIP Service。
- 现有应用详情、日志和启动/停止/重启能力均以单容器 Deployment 为基础。
- 现有数据库只持久化应用名称、镜像、副本数、状态和项目归属，尚无面向用户的 PVC、ConfigMap 或 Secret 管理能力。

## Requirements

### R1. 兼容的创建入口

- 扩展现有项目内应用创建 API，不新增裸 Pod 创建接口。
- 保留 `name`、`image`、`replicas` 与原单端口请求的兼容行为。
- 未提交高级配置时，生成的 Deployment 与 Service 行为应保持现状。
- 旧 `port` 字段继续有效，并自动映射为一个 TCP 容器端口及同值 Service 端口；旧 `port` 与新 `ports` 不得同时提交。

### R2. 受控单容器参数

创建请求支持以下白名单能力：

- 容器命令、参数、工作目录与镜像拉取策略；
- 普通环境变量，以及 ConfigMap/Secret 的整项引用或键引用；
- 可整体省略且可独立填写的 CPU/内存 requests 与 limits；值必须是正数 Kubernetes Quantity，同时填写同一资源的 request 与 limit 时 request 不得大于 limit；
- 多端口：每项包含唯一名称、容器端口、协议 `TCP`/`UDP` 与可选 Service 端口；填写 Service 端口即暴露到同一个 ClusterIP Service，未填写则仅声明容器端口；
- 可分别配置的 startup、readiness、liveness 探针；每个探针支持 HTTP GET、TCP Socket、Exec 三种 handler，且一次只能选择一种；
- 卷与卷挂载；
- 非特权安全上下文：`runAsNonRoot`、非负的 `runAsUser`/`runAsGroup`/`fsGroup`、`readOnlyRootFilesystem`、仅允许为 `false` 的 `allowPrivilegeEscalation`、仅允许 `drop` 的 capabilities，以及仅允许 `RuntimeDefault` 的 seccomp；
- Pod 终止宽限期；
- imagePullSecret 引用。

### R3. 存储与敏感配置边界

- 卷类型仅允许 `emptyDir`、已有 PVC、已有 ConfigMap、已有 Secret。
- PVC、ConfigMap、Secret 与 imagePullSecret 必须位于应用所属项目命名空间，由用户预先创建。
- Astro 本次只保存引用，不创建或管理这些资源，也不保存或返回 Secret 内容。
- 创建应用前必须使用已授权项目的命名空间校验所有引用资源存在；Secret 只执行元数据 `get`，不得读取或记录其 data/stringData 内容。
- 引用资源不存在或 Astro 运行身份缺少对应 `get` 权限时，创建失败并返回明确的统一错误响应，不写入应用记录或创建工作负载。
- `hostPath` 明确禁止。

### R4. 安全边界

- 不接收或透传 Kubernetes 原生 `PodSpec`。
- 不支持多容器、Init Container、Sidecar 或直接创建 Pod。
- 不允许特权容器、hostNetwork、hostPID、hostIPC 或 hostPort。
- 不允许 capabilities `add`、自定义 seccomp，或将 `allowPrivilegeEscalation` 设为 `true`。
- 安全上下文字段未提交时保持现有 Kubernetes 默认行为，避免破坏旧镜像。
- 不允许 NodePort；同一请求内端口名称、容器端口+协议、Service 端口+协议不得重复。
- 不开放节点选择、亲和性、污点容忍、调度器、ServiceAccount、任意标签或注解。
- 所有外部输入在 handler 层完成格式、范围、数量、重复与冲突校验。
- 创建请求体最大 64 KiB。
- 环境变量与 ConfigMap/Secret 引用合计最多 100 项；端口最多 20 项；卷与挂载各最多 20 项；imagePullSecret 最多 10 项；command 最多 20 项；args 最多 100 项；单个探针的 Exec 参数或 HTTP Header 最多 20 项。
- 名称、端口、路径、Quantity 等字段继续按 Kubernetes 原生格式及本需求的安全白名单校验。
- Astro 不硬编码 CPU/内存容量上限；项目命名空间的 `ResourceQuota`/`LimitRange` 负责容量约束，Kubernetes 准入拒绝需转换为清晰的创建失败信息。
- 项目所有权检查、统一响应格式和 `pkg/errcode` 错误码规则保持不变。

### R5. 生命周期与持久化

- 高级配置仅能在创建时设置；本次不新增配置编辑接口。
- 应用详情 API 只读返回创建时保存的高级配置，Web 详情页只读展示。
- 修改配置需删除并重建应用。
- 创建失败继续执行现有数据库事务回滚与 Kubernetes 资源补偿，不得遗留 Deployment 或 Service。

### R6. Web 创建体验

- Web 创建页继续默认展示现有基础字段。
- 新增参数放入默认折叠的“高级配置”，并按容器、环境、资源、网络、健康检查、存储和安全等语义分组。
- 展开控件具备可访问名称与展开状态；所有字段具有可见标签和必要帮助文本。
- 校验错误显示在对应字段附近；多字段提交失败后焦点移至错误摘要或首个无效字段。
- 页面在小屏幕下不得产生横向滚动。

### R7. 文档与质量

- Swagger 文档同步请求和响应契约。
- 自动化测试覆盖 handler 校验、service 传递、Kubernetes 对象映射、失败补偿、Web 响应解析和关键页面交互。
- 使用 Makefile 已有命令完成格式化、测试、静态检查和前端检查。

## Out of Scope

- 应用配置编辑与滚动更新。
- 多容器、Init Container、Sidecar。
- PVC、ConfigMap、Secret 的创建、编辑或删除。
- NodePort、LoadBalancer、Ingress 或外部域名暴露。
- Kubernetes 原生对象或任意 YAML 透传。
- gRPC 探针。
- 高风险宿主能力和高级调度能力。

## Acceptance Criteria

- [ ] 给定仅含现有基础字段的合法请求，API 与 Web 创建行为向后兼容。
- [ ] 给定白名单内的高级参数，创建出的 Deployment PodTemplate 准确包含对应配置。
- [ ] 给定多个端口，创建对应的 ClusterIP Service；未声明端口时不创建 Service。
- [ ] 给定允许的卷或配置引用，工作负载只引用项目命名空间内的资源，响应和日志不包含 Secret 内容。
- [ ] 给定不存在的引用资源或不足的元数据读取权限，创建在写入资源前失败，并返回可定位原因的统一错误响应。
- [ ] 给定白名单外、高风险、重复、冲突、越界或格式非法的参数，handler 返回统一错误响应，且不产生数据库或 Kubernetes 残留。
- [ ] 应用详情 API 与页面只读展示持久化配置，Secret 仅显示资源名和键。
- [ ] Web 高级配置默认折叠、可访问、响应式，并提供字段级错误与可发现的提交失败反馈。
- [ ] 自动化测试和 Swagger 文档覆盖新增契约，相关 Make 质量检查全部通过。

## Dependencies

- 仓库未包含 Astro 运行身份的 Kubernetes RBAC 清单；部署环境必须为项目命名空间内的 PVC、ConfigMap、Secret 提供只读 `get` 权限，且不得授予读取 Secret 内容之外的额外业务能力。
- 部署环境应以项目命名空间的 `ResourceQuota`/`LimitRange` 提供实际容量边界；本功能不新增配额管理。
