# 目录结构

> 各类代码放在哪里，以及分层调用规则。

---

## 布局

```
cmd/server/main.go      # 唯一入口：加载配置 → 初始化 logger/DB/K8s → 注册路由
internal/
├── handler/            # HTTP 处理器：参数绑定校验、调用 service、统一响应
│   ├── response.go     # 统一响应结构 Response{code,message,data} + 辅助函数
│   ├── user.go         # 用户注册/登录
│   ├── project.go      # 项目创建、查询与空项目删除
│   └── app.go          # 项目内应用创建、查询、删除、启停与日志
├── service/            # 业务逻辑：权限检查、errcode 包装、编排 repository 与 k8s
├── repository/         # 数据访问：GORM 查询，db.go 持有包级 DB 变量
├── model/              # GORM 模型（BaseModel + User + Project + App）
├── middleware/         # Gin 中间件（auth.go：JWT 认证）
└── k8s/                # K8s 封装（client.go 连接，adapter.go AppAdapter 接口）
pkg/
├── config/             # Viper 配置加载，GlobalConfig 全局变量
├── errcode/            # 错误码枚举 + Error 类型
└── logger/             # Zap 封装，包级函数 Info/Error/...
configs/config.yaml     # 唯一配置文件
docs/                   # swag 生成的 Swagger 文档（make swagger）
```

## 分层规则

调用方向单向：**handler → service → repository / k8s adapter**。

- handler 不直接访问 repository 或 `repository.DB`。
- service 不引用 gin（不接触 `*gin.Context`），入参用自定义 Request 结构体（如 `service.CreateAppRequest`）。
- repository 只做 GORM 操作，不含业务判断。
- K8s 操作一律经 `k8s.AppAdapter` 接口（`internal/k8s/adapter.go`），service 不直接用 client-go。

## 命名约定

- 每层按领域分文件：`user.go` / `app.go`，新领域（如 template）在每层各加一个同名文件。
- 每层的类型 + 构造函数模式：`AppHandler`/`NewAppHandler()`、`AppService`/`NewAppService()`、`AppRepository`/`NewAppRepository()`（无依赖注入框架，构造函数内部直接 `New` 下层）。
- handler 层的请求/响应 DTO 定义在同文件顶部，带 `binding` 与 `example` tag（见 `internal/handler/app.go` 的 `CreateAppRequest`）。

## 禁止

- 在 `internal/` 外引用 `internal/` 包以外的业务逻辑（业务只在 internal）。
- 新增第二个 main 入口；只有 `cmd/server`。
- handler 里写 SQL 或 GORM 调用。
