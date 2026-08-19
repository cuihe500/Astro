# Astro 后端代码规范合规审查报告

## 1. 审查结论

审查基线为根目录 `AGENTS.md`、`.trellis/spec/backend/` 及当前任务 `prd.md`。范围包括 `cmd/`、`internal/`、`pkg/`、`configs/`、`Makefile`、`go.mod`，以及 handler 注解与 `docs/` Swagger 生成物的一致性。

确认 17 项需要整改的问题：

| 级别 | 数量 | 结论 |
|---|---:|---|
| P0 | 1 | 默认 JWT 密钥可直接导致认证绕过 |
| P1 | 4 | OAuth 凭据日志泄露、Kubernetes 残留、状态一致性和核心启停逻辑问题 |
| P2 | 6 | 输入/响应/错误映射和错误处理明确不符合规范 |
| P3 | 6 | Swagger、测试、Makefile、注释、错误码和架构文档问题 |

本次审查没有修改业务源码、配置文件或 Swagger 生成物；只新增了本报告。数据库、Kubernetes 集群和真实 OAuth Provider 未连接，因此运行时依赖外部系统的结论标为静态验证。

## 2. 已执行检查

| 命令 | 结果 | 说明 |
|---|---|---|
| `make build` | 通过 | 成功生成 `bin/astro` |
| `make test` | 通过 | 全部包通过；`TestOAuth2State`、`TestOAuth2StateShape` 通过 |
| `make lint` | 通过 | `golangci-lint run`，0 issues |
| `make swagger` | 通过 | `docs/docs.go`、`docs/swagger.json`、`docs/swagger.yaml` 重新生成后与工作树无差异 |
| `git diff --check -- docs` | 通过 | 无空白错误 |
| 缺失构建目录探针 | 通过 | 使用 `make build BUILD_DIR=<临时不存在目录>` 成功，未确认 Makefile 的构建目录问题 |

`make swagger` 输出了 `warning: failed to get package name in dir: ./ ... no Go files`，原因是仓库根目录没有 Go 文件；命令最终退出成功，生成物没有变化，不作为代码违规。

## 3. 已确认发现

### P0-01 默认 JWT Secret 可预测且启动时不校验

- **位置**：`configs/config.yaml:13-15`；`pkg/config/config.go:69-83`；`internal/service/user.go:81-90`；`internal/middleware/auth.go:37-43`。
- **规范**：`.trellis/spec/backend/auth-guidelines.md:61` 要求生产 JWT secret 通过环境变量或 Secret 注入，并在为空、仍为默认值或强度不足时启动失败；`AGENTS.md:125-128` 的安全要求。
- **证据**：仓库配置直接使用 `secret: astro-secret-key`。`config.Load` 只读取 YAML 并执行 `viper.Unmarshal`，没有 `AutomaticEnv`、`BindEnv` 或 secret 强度检查；签发和校验路径直接使用该值。
- **影响**：按仓库默认配置部署时，知道仓库内容即可离线签发任意 `user_id` 的有效 JWT，随后访问其他用户的应用并执行启停、删除等操作。
- **最小整改建议**：绑定明确的 `JWT_SECRET` 环境变量或 Secret 文件；加载配置时拒绝空值、示例/默认值和不足长度的 secret；生产配置不得回退到仓库默认值。补充启动失败测试。
- **验证状态**：静态调用链已确认；未连接真实部署环境。

### P1-01 Gin 默认访问日志记录 OAuth2 callback 的 code/state

- **位置**：`cmd/server/main.go:68-69`；`internal/handler/user.go:133-142`；生成路由见 `docs/swagger.yaml:355-395`。
- **规范**：`.trellis/spec/backend/logging-guidelines.md:3,42-46` 禁止 HTTP 请求日志记录 OAuth2 callback 的 `code`、`state` 或直接输出 `URL.RawQuery`，并要求使用 `pkg/logger`。
- **证据**：`gin.Default()` 自动安装 Gin Logger。当前锁定的 Gin v1.11 默认 formatter 会在请求路径后拼接非空 `URL.RawQuery`；callback 的凭据正是 `code` 和 `state` 查询参数。该日志直接写 Gin 默认输出，不经过 `pkg/logger` 的文件轮转和结构化字段。
- **影响**：授权码和 state 会进入 stdout、容器日志或日志聚合系统；在换码失败或请求未完成时，授权码可能仍处于可使用窗口，造成 OAuth 会话被劫持的风险。
- **最小整改建议**：改用 `gin.New()`，显式安装自定义 Recovery 和经 `pkg/logger` 的访问日志；对 callback 路由跳过 query 或只记录脱敏后的路径。
- **验证状态**：通过项目代码和 Gin 1.11 依赖源码静态确认；未在实际日志聚合系统复测。

### P1-02 创建应用失败时可能遗留不可管理的 Kubernetes Deployment

- **位置**：`internal/k8s/adapter.go:145-173`；`internal/service/app.go:61-76`。
- **规范**：`AGENTS.md:115-118` 的错误处理/可靠性要求；任务 PRD 要求验证跨层数据一致性。
- **证据**：适配器先创建 Deployment（145-148），有端口时再创建 Service（150-173）。Service 创建失败时直接返回错误，没有删除已经创建的 Deployment。上层随后只尝试删除数据库记录（73-76）。
- **影响**：API 返回创建失败，但 Deployment 仍运行并消耗集群资源；数据库记录被删除后，用户无法再通过 Astro 找到或删除该工作负载，资源会成为孤儿。
- **最小整改建议**：在适配器内对 Service 创建失败执行 Deployment 补偿删除，并显式处理补偿失败；上层数据库清理失败也要记录并返回可诊断的系统错误。将该路径作为回滚测试覆盖。
- **验证状态**：静态控制流已确认；需要故意让 Service 创建失败的 Kubernetes 集群测试运行时补证。

### P1-03 启停、重启和状态同步静默忽略数据库更新错误

- **位置**：`internal/service/app.go:130`、`147-148`、`164-165`、`229-239`。
- **规范**：`AGENTS.md:115-116` 禁止忽略错误；`.trellis/spec/backend/error-handling.md:54` 及 `.trellis/spec/backend/quality-guidelines.md:46-51` 禁止用 `_` 丢弃错误。补偿清理是详细规范明确列出的唯一例外，不适用于状态更新。
- **证据**：Kubernetes 操作成功后，`UpdateStatus`/`UpdateReplicas` 返回值被 `_ =` 丢弃，随后 API 返回成功。异步 `syncAppStatus` 对 K8s 查询错误直接 `return`，对两次数据库更新也全部丢弃。
- **影响**：API 宣布操作成功但数据库仍可能保存旧状态或错误副本数；列表和详情会长期与实际集群状态不一致，且异步失败没有诊断记录。
- **最小整改建议**：同步路径检查更新错误并映射为 `ErrDatabase`；异步路径至少使用 `pkg/logger` 记录带 `app_id`、`namespace` 和 `zap.Error(err)` 的结构化错误，必要时增加重试或状态标记。
- **验证状态**：源码静态确认；未连接数据库验证具体故障注入结果。

### P1-04 停止应用会丢失期望副本数，启动时退化为 1

- **位置**：`internal/service/app.go:120-124`、`136-150`；数据字段见 `internal/model/model.go:42-50`。
- **规范**：应用生命周期和跨层数据一致性要求；代码自身注释 `StartApp:120` 声明“恢复到原来的副本数”。架构文档 `docs/architecture-design.md:604-607` 也将副本数定义为用户输入的期望值。
- **证据**：`StopApp` 先将 K8s 副本缩为 0，再把数据库 `replicas` 写成 0。`StartApp` 读取该字段，遇到 0 后固定设置为 1。因此原先 2-10 副本的应用 stop/start 后不会恢复原值。
- **影响**：核心启停流程改变用户配置，造成容量和服务行为变化；状态同步又只在 `status.Replicas > 0` 时更新副本数，无法可靠恢复被覆盖的期望值。
- **最小整改建议**：把 `App.Replicas` 保留为期望副本数；停止时只更新状态和集群实际副本，不将期望值写成 0。若需要保存实际副本数，增加独立字段并明确读写契约。
- **验证状态**：静态数据流已确认；建议补充 stop/start 的最小服务层回归测试。

### P2-01 Handler 层外部输入校验不完整且存在约束矛盾

- **位置**：`internal/handler/user.go:20-30`；`internal/handler/app.go:23-29`、`287-292`；下游资源使用见 `internal/k8s/adapter.go:145-172`、`309-333`。
- **规范**：`AGENTS.md:120-123` 要求所有外部输入在 handler 层有效校验；`.trellis/spec/backend/quality-guidelines.md:26-37` 要求请求 DTO 使用完整 `binding` 校验。架构文档 `docs/architecture-design.md:1101-1110` 明确示例为用户名 3-20、密码 6-30。
- **证据**：注册用户名和密码只有 `required`，可接受任意长度；应用名只有非空校验，未限制 Kubernetes 名称/标签格式；镜像没有长度限制；端口没有 `0..65535` 范围校验，负数会在适配器中被静默视为“不暴露端口”；`replicas` 同时使用 `required,min=0`，整数 0 会被 `required` 判为空而无法按文档提交；`lines` 解析失败时静默回退 100，而不是返回 `ErrBadRequest`。
- **影响**：无效值进入 bcrypt、数据库和 Kubernetes 后才失败，导致错误码和用户反馈不准确；非法资源名称可能留下失败记录/触发补偿路径，日志查询参数还可能造成资源压力。
- **最小整改建议**：在 handler DTO 中补充用户名/密码长度和格式、名称 DNS-1123 约束、镜像长度、端口范围和副本范围；`lines` 解析失败或超上限直接 `BadRequest`。保留 0 副本时去掉整数 `required` 或改用指针区分未传与 0。
- **验证状态**：源码与文档静态确认；未对 Gin validator 的所有边界组合做运行时测试。

### P2-02 日志行数无上限，结果被完整读入内存

- **位置**：`internal/handler/app.go:287-300`；`internal/k8s/adapter.go:309-333`。
- **规范**：`AGENTS.md:120-123` 的外部输入校验要求；质量规范禁止把未经限制的外部参数直接传入业务操作。
- **证据**：任意正的 `int64` `lines` 都会传入 Kubernetes `TailLines`，没有最大值；适配器随后用 `io.Copy` 将整个日志流写入 `bytes.Buffer` 后一次性转换为字符串。
- **影响**：已认证用户可以请求极大的日志窗口，导致 Astro 进程占用过多内存和带宽，形成可重复的服务资源压力。
- **最小整改建议**：在 handler 设定明确上限并拒绝超限值；适配器同时设置服务端/读取上限，必要时使用流式分页而不是无限制 `bytes.Buffer`。
- **验证状态**：静态确认；未进行压力测试。

### P2-03 统一响应结构在错误和部分成功路径中缺少 data

- **位置**：`internal/handler/response.go:10-35`；`cmd/server/main.go:71-74`。
- **规范**：`AGENTS.md:120-123` 要求所有 API 响应包含 `code`、`message`、`data`；`.trellis/spec/backend/error-handling.md:50-52` 要求统一响应。
- **证据**：`Response.Data` 使用 `json:"data,omitempty"`，`Error` 没有给 Data 赋值，因此所有错误响应都省略 `data`；`Success(c, nil)` 的注册、删除、启停响应也省略 `data`。`/health` 直接返回 `{"status":"ok"}`，完全绕过统一响应。
- **影响**：客户端无法依赖稳定的响应 schema，健康检查和业务 API 的协议不一致。
- **最小整改建议**：移除 `omitempty`，让辅助函数始终输出 `data`（无数据时为 `null`）；健康检查也使用统一响应或明确将其排除在 API 规范之外并在文档中说明。
- **验证状态**：由 JSON tag、响应辅助函数和路由实现静态确认。

### P2-04 GetApp 二次 repository 查询绕过 service 错误转换

- **位置**：`internal/service/app.go:185-195`。
- **规范**：`.trellis/spec/backend/error-handling.md:21-32` 要求 repository 原始错误由 service 转换为 `errcode.Error`；`auth-guidelines.md:63-73` 也要求非 NotFound repository 错误转为 `ErrDatabase`。
- **证据**：`GetApp` 先通过 `getAppWithPermission` 做权限检查和一次查询，状态同步后直接 `return s.repo.GetByID(appID)`，没有对第二次查询的 `gorm.ErrRecordNotFound` 或其他数据库错误做映射。
- **影响**：并发删除或数据库故障会被 handler 的 `FromError` 归为通用 `ErrInternal`，而不是 `ErrAppNotFound`/`ErrDatabase`；错误契约不稳定，并可能把底层错误文本返回客户端。
- **最小整改建议**：对第二次查询完全复用 service 层的 NotFound/Database 映射；必要时避免重复查询或让状态同步返回更新后的对象。
- **验证状态**：静态确认；并发删除场景未运行时注入。

### P2-05 本地注册重复邮箱和数据库失败的错误码不符合分层契约

- **位置**：`internal/service/user.go:26-50`；`internal/model/model.go:21-25`；错误码定义 `pkg/errcode/code.go:23-33,66-76`。
- **规范**：错误码分段见 `AGENTS.md:130-137`；service 应将 repository 的数据库错误映射为系统段错误，认证规范的错误矩阵要求邮箱冲突返回 `ErrEmailExists`。
- **证据**：`Register` 只预查用户名，没有预查邮箱；邮箱有数据库唯一索引。插入冲突或其他 `CreateUser` 数据库错误全部包装成业务码 `ErrRegisterFailed`，已定义的 `ErrEmailExists` 未用于本地注册。
- **影响**：客户端无法区分邮箱已占用与数据库故障；数据库故障被错误归类为业务失败，违反统一错误码和跨层错误传播契约。
- **最小整改建议**：增加邮箱冲突检查，并对最终唯一约束竞态做安全映射；非业务数据库错误统一返回 `ErrDatabase`，详细原因只进入受控日志。
- **验证状态**：静态确认；未连接 MariaDB 验证具体驱动错误类型。

### P2-06 Logger fallback 忽略构造错误

- **位置**：`pkg/logger/logger.go:106-112`。
- **规范**：`AGENTS.md:115-116`、`.trellis/spec/backend/quality-guidelines.md:46-51` 禁止使用 `_` 忽略错误。
- **证据**：`Default` 在 logger 尚未初始化时执行 `defaultLogger, _ = zap.NewDevelopment()`。
- **影响**：fallback 构造失败时错误被隐藏，后续日志调用可能得到不可用 logger 或丢失故障信息；同时形成明确的错误处理违规。
- **最小整改建议**：显式处理返回值；无法恢复时使用已知可靠的 `zap.NewNop()` 或返回初始化错误，并补充测试。
- **验证状态**：静态确认；当前 `make lint` 未启用能捕获该模式的规则，因此 lint 通过不代表符合项目规则。

### P3-01 Swagger 声明的 HTTP 错误状态与实际协议不一致

- **位置**：`internal/handler/user.go:51-53,78-79,130-131`；`internal/handler/app.go:44-46,82-83,108-110` 等全部 handler 注解；实现 `internal/handler/response.go:26-35`；生成物 `docs/swagger.yaml:86-135,320-455`。
- **规范**：`.trellis/spec/backend/error-handling.md:50-52` 明确所有业务错误返回 HTTP 200；质量规范 `quality-guidelines.md:39-41` 要求注解和生成文档同步真实接口。
- **证据**：所有 `Error` 路径调用 `c.JSON(http.StatusOK, ...)`，但注解和生成 Swagger 声明 400、401、404、500 响应。
- **影响**：按 Swagger 生成的客户端、网关和监控会依据错误 HTTP 状态做错误处理，而实际只能读取 body 中的业务码。
- **最小整改建议**：若继续采用项目的 HTTP 200 协议，把失败结果统一描述为 200 响应体并在 schema/说明中明确业务码；修改注解后再次执行 `make swagger`。若改为语义 HTTP 状态，则需同步修改统一响应规范和全部 handler。
- **验证状态**：源注解、实现和生成物三方静态确认；`make swagger` 已执行且生成物当前同步。

### P3-02 OAuth2 用户映射/创建逻辑缺少规范要求的测试

- **位置**：现有测试 `internal/service/oauth2_test.go:9-41`；被测逻辑 `internal/service/oauth2.go:99-137`。
- **规范**：`.trellis/spec/backend/auth-guidelines.md:76` 要求涉及 UserInfo 解析或用户创建时，至少覆盖 `sub` 缺失、email 冲突或已有 identity 命中之一。
- **证据**：现有测试只覆盖 state 生成、篡改、provider 不匹配和过期；`findOrCreateUser` 的 identity 命中、email 冲突和创建路径没有测试。
- **影响**：OAuth2 账号绑定和冲突保护回归无法由现有测试及时发现。
- **最小整改建议**：增加一个最小化的 service/repository 替身测试，至少覆盖 email 冲突或已有 identity 命中，并用 `make test` 验证。
- **验证状态**：通过测试文件静态确认；`make test` 当前通过但无法证明缺失分支正确。

### P3-03 Makefile 的 test、lint 未声明为 phony

- **位置**：`Makefile:1,15-19`。
- **规范**：`.trellis/spec/backend/quality-guidelines.md:7-18` 要求通过 Makefile 稳定执行项目检查目标。
- **证据**：`.PHONY` 只有 `build run clean swagger`，但 `test` 和 `lint` 也是实际目标。
- **影响**：仓库根目录若出现名为 `test` 或 `lint` 的文件/目录，Make 可能跳过对应检查，导致维护者误以为测试或静态检查已执行。
- **最小整改建议**：把 `test lint` 加入 `.PHONY`。
- **验证状态**：静态确认；当前工作树没有同名冲突文件，故未触发现象。

### P3-04 多个导出类型/函数缺少中文注释

- **位置**：`pkg/config/config.go:7,39,44,53,58`；`internal/service/user.go:16,20`；`internal/repository/user.go:8,10`；`internal/handler/user.go:8,13`；`pkg/errcode/code.go:10` 等。
- **规范**：`AGENTS.md:54-55` 的导出函数/类型注释要求；`.trellis/spec/backend/quality-guidelines.md:20-24` 要求导出函数/类型有中文 Go doc。
- **证据**：例如 `Config`、`ServerConfig`、`UserService`、`UserRepository`、`UserHandler` 及其构造函数直接导出，没有紧邻的中文 Go doc。`RegisterUserRoutes` 的注释写成 `// RegisterRoutes`，标识符也不匹配（`internal/handler/user.go:150-151`）。
- **影响**：公共包 API 文档不完整，静态文档工具和维护者无法准确理解导出符号职责。
- **最小整改建议**：为每个导出类型/函数补充以标识符开头的中文注释，并修正 `RegisterUserRoutes` 注释名称。
- **验证状态**：静态确认；`make lint` 当前配置未启用 exported-comment 检查。

### P3-05 重复且未使用的应用创建错误码

- **位置**：`pkg/errcode/code.go:37-45,80-88`；当前 service 使用 `internal/service/app.go:76` 的 `ErrAppCreateFailed`。
- **规范**：`AGENTS.md:110-113` 的 KISS/YAGNI 原则及错误码单一契约要求。
- **证据**：`ErrAppCreateFail=21003` 与 `ErrAppCreateFailed=21009` 的默认消息均为“创建应用失败”，后者注释还标为别名，但两者是不同数字；静态搜索未发现 `ErrAppCreateFail` 的实际调用。
- **影响**：同一业务故障可能被不同调用者映射为不同 code，增加客户端分支和维护成本。
- **最小整改建议**：保留一个正式错误码，或将旧码明确标记 deprecated 并提供迁移说明；不要用不同数字伪造别名。
- **验证状态**：静态搜索确认；未改变现有错误码，避免影响外部客户端。

### P3-06 架构文档宣称支持环境注入，但配置实现未执行

- **位置**：`docs/architecture-design.md:1119-1148`；`pkg/config/config.go:69-83`；`configs/config.yaml:13-15`。
- **规范**：`AGENTS.md:139-142` 要求文档与实现同步；认证规范 `auth-guidelines.md:61` 要求生产 secret 走环境变量或 Secret 管理。
- **证据**：架构文档生产示例明确写出 `os.Getenv("JWT_SECRET")`，但实际 `config.Load` 没有环境变量绑定或读取逻辑；运行时仍只读取 YAML 中的固定 secret。该文档与实现同时造成部署者错误安全假设。
- **影响**：运维按文档配置环境变量后，程序可能继续使用仓库默认值；P0 的认证风险因此更容易在生产部署中被触发。
- **最小整改建议**：先实现并测试环境/Secret 注入，再同步更新文档；在实现完成前不要宣称生产配置已受支持。
- **验证状态**：文档与源码静态对照确认。

## 4. 关键排除项与残余风险

以下项目经过调用链核对，没有计入上述违规：

- 应用资源操作均经过认证路由和 service 层 `app.UserID != userID` 归属检查；列表使用 `GetByUserID`，未确认跨租户绕过。
- GORM 查询均使用 `?` 参数占位符；没有发现 `Raw`、`Exec` 或 SQL 字符串拼接。`fmt.Sprintf` 构造 DSN/Kubernetes label selector 不属于 SQL 拼接。
- OAuth2 state 已包含 provider、过期时间、随机 nonce，并使用 HMAC 和 `hmac.Equal` 校验；身份查询使用 `provider + sub/id`，email 冲突不会自动绑定。
- OAuth2 用户密码使用非法 bcrypt 占位值，当前本地 `/login` 会验证失败；`User.Password` 有 `json:"-"`，未发现密码进入 JSON 响应。
- 当前未发现真实 client secret、access token 或授权码写入 Swagger 示例；Swagger 生成物与源码注解同步。callback 凭据进入访问日志是 P1-01，不能因为生成物未泄露而排除。
- `internal/service/app.go:75` 的数据库删除属于详细错误规范明确允许的补偿清理例外，因此没有把该行单独列为“忽略错误”违规；但其失败未记录会放大 P1-02 的残留风险。根 `AGENTS.md` 对 `_` 的文字更严格，建议后续统一两份规范。
- 除 OAuth2 规范明确要求的用户映射测试外，其他历史业务层测试缺口被质量规范标为技术债，本报告没有扩大为违规。

待确认/建议关注但不计入确认数量的项目：

- OAuth2 state 当前没有服务端单次消费或浏览器会话绑定；现行认证规范只要求签名、provider、过期和 nonce，因此列为安全加固建议而非当前规范违规。
- `http.DefaultClient` 没有独立的 client-level timeout，但请求通过 `NewRequestWithContext` 绑定了 callback context；没有据此认定为永久阻塞缺陷，仍建议为外部 Provider 增加明确 deadline。
- `App` 没有 `(user_id, name)` 复合唯一索引；服务层已有重复检查，且 Kubernetes 名称也形成实际约束。若产品要求数据库层独立保证并发唯一，应补充明确契约和索引测试。
- `User.Status` 已建模但登录路径未检查禁用状态；当前没有启用/禁用 API，需先确认该字段是否属于本阶段功能，再决定是否列为认证缺陷。
- `server.mode: debug` 是当前配置示例；若直接用于生产会增加调试输出，应在生产部署约束中强制覆盖为 release，但未把开发默认值单独计为代码违规。
- `main.go:37,43,48` 的 `fmt.Fprintf(os.Stderr, ...)` 按日志规范字面命中 `fmt.Print*` 禁止项，但前两处发生在 logger 初始化前，属于启动失败兜底。建议在规范中明确 bootstrap stderr 例外，或提供可靠的 bootstrap logger；本报告暂不把它计入确认数量。

## 5. 建议整改顺序

1. 立即阻断生产使用默认 JWT secret，并修复 Gin callback query 日志；核查现有日志是否已经收集过授权码。
2. 修复创建失败的 Kubernetes 补偿、启停副本数语义和数据库状态更新错误处理。
3. 补齐 handler 输入边界和日志行数上限，统一所有响应的 `data` 字段。
4. 修复 service 二次查询和注册错误码映射，避免错误契约漂移。
5. 同步 Swagger/架构文档，补齐 OAuth2 用户映射测试、导出注释和 Makefile `.PHONY`。

## 6. 审查时 Git 状态

- 分支：`main`
- 审查基线提交：`b4c5861833eb80d33cfecaa25079a5db55a92399`
- 审查期间未修改业务代码、配置或 `docs/`。
- 工作树中的已有规范修改：
  - `.trellis/spec/backend/auth-guidelines.md`
  - `.trellis/spec/backend/logging-guidelines.md`
  - `.trellis/spec/backend/quality-guidelines.md`
- 本任务目录为未跟踪内容，审查期间新增本文件 `report.md`；任务规划文件属于既有任务产物。
