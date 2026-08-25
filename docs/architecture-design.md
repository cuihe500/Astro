# Astro 容器即服务平台 - 架构设计文档

> 版本：v1.1
> 日期：2026-08-24
> 状态：MVP 实施中

---

## 文档目录

1. [项目概述](#1-项目概述)
2. [架构设计](#2-架构设计)
3. [技术选型](#3-技术选型)
4. [模块设计](#4-模块设计)
5. [接口设计](#5-接口设计)
6. [数据模型](#6-数据模型)
7. [部署方案](#7-部署方案)
8. [安全设计](#8-安全设计)
9. [监控与运维](#9-监控与运维)
10. [风险与挑战](#10-风险与挑战)

---

## 1. 项目概述

### 1.1 项目定位

Astro 是一个面向 C 端用户的容器即服务（CaaS）平台，目标是：
- **降低容器技术使用门槛**：让非技术用户也能轻松部署容器化应用
- **简化运维复杂度**：提供一键部署、自动化管理
- **开箱即用**：预置常用应用模板（MySQL、Redis、Nginx 等）

### 1.2 目标用户

- 个人开发者和技术爱好者
- 小型创业团队（<10 人）
- 需要快速验证想法的产品经理
- 学习容器技术的初学者

### 1.3 核心价值

| 传统方式 | Astro 方案 |
|---------|-----------|
| 需要编写复杂的 YAML | Web 界面填几个参数 |
| 理解 Pod/Deployment/Service | 只需懂"部署应用" |
| 手动管理容器生命周期 | 一键启停、重启、删除 |
| 查看日志需要 kubectl | 浏览器中直接查看 |

### 1.4 MVP 功能范围

**第一阶段（当前设计）** - 实现进度：✅ 已完成
- ✅ 用户注册/登录/JWT 认证
- ✅ 应用创建/删除
- ✅ 应用启动/停止/重启
- ✅ 应用日志查看
- ✅ 项目级多租户隔离（每个项目独立命名空间）

**第二阶段（未来扩展）** - 状态：🔄 规划中
- 🔄 应用模板市场
- 🔄 资源监控（CPU/内存）
- 🔄 自动扩缩容
- 🔄 持久化存储管理
- 🔄 域名绑定和证书管理

---

## 1.5 开发进度 TODO

> 最后更新：2026-08-24

### 基础设施层 ✅ 已完成
- [x] `pkg/config` - 配置加载（Viper）
- [x] `pkg/logger` - 日志系统（Zap + 日志轮转）
- [x] `pkg/errcode` - 错误码定义
- [x] `internal/k8s/client.go` - K8s 客户端初始化
- [x] `cmd/server/main.go` - 服务启动入口

### 用户模块 ✅ 已完成
- [x] `internal/model/model.go` - User 数据模型
- [x] `internal/repository/user.go` - UserRepository
- [x] `internal/service/user.go` - UserService（注册、登录、JWT）
- [x] `internal/handler/user.go` - UserHandler（Register、Login）
- [x] `internal/middleware/auth.go` - JWT 认证中间件

### 应用模块 ✅ 已完成
- [x] `internal/model/model.go` - App 数据模型
- [x] `internal/k8s/adapter.go` - K8s Adapter 接口和实现
- [x] `internal/repository/app.go` - AppRepository
- [x] `internal/service/app.go` - AppService（CRUD、启停、日志）
- [x] `internal/handler/app.go` - AppHandler（8 个 API）

### 项目模块 ✅ 已完成
- [x] `internal/model/model.go` - Project 数据模型及 App 项目外键
- [x] `internal/repository/project.go` - ProjectRepository
- [x] `internal/service/project.go` - ProjectService（创建、查询、空项目删除）
- [x] `internal/handler/project.go` - ProjectHandler（4 个 API）
- [x] `web/src/features/projects/` - 项目列表、创建及删除界面

### 待开发功能 📋 TODO
- [x] 前端 Web 界面
- [ ] 应用模板市场功能
- [ ] 资源监控（Prometheus 集成）
- [ ] 持久化存储管理（PVC）
- [ ] 域名绑定和 TLS 证书
- [ ] 用户邮箱验证
- [ ] 应用配额限制（ResourceQuota）
- [ ] 网络策略隔离（NetworkPolicy）

---

## 2. 架构设计

### 2.1 总体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                          用户层                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │  Web 前端    │  │  移动端 App（不实现） │  │  CLI 工具（不实现） │          │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘          │
│         │                  │                  │                  │
│         └──────────────────┴──────────────────┘                  │
│                            │                                     │
│                            │ HTTPS/REST API                      │
│                            ▼                                     │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                      Astro API 服务层                            │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Gin HTTP Server                                         │   │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐        │   │
│  │  │ Auth 中间件│  │ Logger 中间件│ │ CORS 中间件│       │   │
│  │  └────────────┘  └────────────┘  └────────────┘        │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Handler 层（HTTP 处理器）                                │   │
│  │  ┌─────────────────┐  ┌──────────────────┐              │   │
│  │  │ UserHandler     │  │ AppHandler       │              │   │
│  │  │ - Register      │  │ - CreateApp      │              │   │
│  │  │ - Login         │  │ - DeleteApp      │              │   │
│  │  └─────────────────┘  │ - Start/Stop     │              │   │
│  │                       │ - GetLogs        │              │   │
│  │                       └──────────────────┘              │   │
│  └──────────────────────────────────────────────────────────┘   │
│                            │                                     │
│  ┌──────────────────────────▼─────────────────────────────┐    │
│  │  Service 层（业务逻辑）                                  │    │
│  │  ┌─────────────────┐  ┌──────────────────┐            │    │
│  │  │ UserService     │  │ AppService       │            │    │
│  │  │ - JWT 生成      │  │ - 应用生命周期   │            │    │
│  │  │ - 密码加密      │  │ - 状态同步       │            │    │
│  │  └─────────────────┘  │ - 权限检查       │            │    │
│  │                       └──────────────────┘            │    │
│  └──────────────────────────┬─────────────────────────────┘    │
│                            │                                    │
│  ┌──────────────────────────▼─────────────────────────────┐    │
│  │  Repository 层（数据访问）                               │    │
│  │  ┌─────────────────┐  ┌──────────────────┐            │    │
│  │  │ UserRepository  │  │ AppRepository    │            │    │
│  │  │ - CRUD          │  │ - CRUD           │            │    │
│  │  └─────────────────┘  └──────────────────┘            │    │
│  └──────────────────────────┬─────────────────────────────┘    │
│                            │                                    │
└────────────────────────────┼────────────────────────────────────┘
                             │
        ┌────────────────────┴─────────────────────┐
        │                    │                      │
        ▼                    ▼                      ▼
┌───────────────┐  ┌──────────────────┐  ┌─────────────────┐
│  MariaDB      │  │  K8s Adapter     │  │  Zap Logger     │
│  (GORM)       │  │  (抽象层)        │  │  (日志系统)      │
│               │  └────────┬─────────┘  └─────────────────┘
│  ┌─────────┐  │           │
│  │ users   │  │           │
│  │ projects│  │           ▼
│  │ apps    │  │  ┌───────────────────────────────────┐
│  └─────────┘  │  │  Kubernetes 集群                   │
└───────────────┘  │  ┌─────────────────────────────┐  │
                   │  │  Namespace: astro-project-… │  │
                   │  │  ┌──────────┐  ┌─────────┐  │  │
                   │  │  │Deployment│  │ Service │  │  │
                   │  │  └────┬─────┘  └─────────┘  │  │
                   │  │       │                      │  │
                   │  │  ┌────▼────┐  ┌─────────┐  │  │
                   │  │  │ Pod 1   │  │ Pod 2   │  │  │
                   │  │  └─────────┘  └─────────┘  │  │
                   │  └─────────────────────────────┘  │
                   │  ┌─────────────────────────────┐  │
                   │  │  Namespace: astro-project-… │  │
                   │  │  ...                         │  │
                   │  └─────────────────────────────┘  │
                   └───────────────────────────────────┘
```

### 2.2 分层架构说明

#### 2.2.1 Handler 层
**职责**：
- 接收 HTTP 请求，解析参数
- 参数校验（使用 Gin 的 binding）
- 调用 Service 层
- 格式化响应（统一 Response 结构）

**设计原则**：
- 轻薄层，不包含业务逻辑
- 只做参数转换和错误翻译

#### 2.2.2 Service 层
**职责**：
- 业务逻辑实现
- 权限检查
- 事务管理
- 调用 Repository 和 K8s Adapter

**设计原则**：
- 单一职责：每个 Service 对应一个业务领域
- 可测试：依赖注入，便于 Mock

#### 2.2.3 Repository 层
**职责**：
- 数据库 CRUD 操作
- SQL 查询封装

**设计原则**：
- 纯数据访问，不含业务逻辑
- 返回领域模型（model.User, model.App）

#### 2.2.4 K8s Adapter 层（关键设计）
**职责**：
- 封装 Kubernetes 操作
- 提供统一的接口抽象

**设计亮点**：
```go
type AppAdapter interface {
    EnsureNamespace(ctx context.Context, namespace string) error
    DeleteNamespace(ctx context.Context, namespace string) error
    CreateApp(ctx context.Context, spec AppSpec) error
    DeleteApp(ctx context.Context, name, namespace string) error
    ScaleApp(ctx context.Context, name, namespace string, replicas int32) error
    GetAppStatus(ctx context.Context, name, namespace string) (*AppStatus, error)
    RestartApp(ctx context.Context, name, namespace string) error
    GetAppLogs(ctx context.Context, name, namespace string, lines int64) (string, error)
}
```

**优势**：
1. **解耦**：Service 层不依赖具体的 K8s 实现
2. **可扩展**：未来可以无缝切换到 Operator 模式
3. **可测试**：可以 Mock Adapter 进行单元测试

### 2.3 渐进式演进路线

#### 阶段 1：client-go 直接调用（当前方案）

```
Service → ClientGoAdapter → Kubernetes API
```

**特点**：
- 简单直接，开发速度快
- 代码量：~300 行
- 适用场景：MVP 验证，用户量 < 1000

#### 阶段 2：引入 Operator（可选升级）

```
Service → OperatorAdapter → 自定义 CRD → Operator Controller → Kubernetes API
```

**触发条件**：
- 用户量 > 100，应用实例 > 500
- 需要支持 StatefulSet、CronJob 等复杂资源
- 需要自动化运维（备份、升级、自愈）

**优势**：
- 声明式管理，自动调谐
- 复杂生命周期管理
- 符合 Kubernetes 原生理念

---

## 3. 技术选型

### 3.1 后端技术栈

| 组件 | 技术选型 | 版本 | 选型理由 |
|------|---------|------|---------|
| 编程语言 | Go | 1.25+ | 高性能、静态类型、K8s 生态友好 |
| Web 框架 | Gin | 1.9.1 | 轻量、高性能、文档完善 |
| 数据库 | MariaDB | 10.6+ | 开源、兼容 MySQL、性能优秀 |
| ORM | GORM | 1.25.0 | 功能强大、自动迁移、关联查询 |
| 日志 | Zap | 1.26.0 | 高性能、结构化日志、支持轮转 |
| 配置管理 | Viper | 1.17.0 | 支持多格式、环境变量、热更新 |
| 认证 | JWT | golang-jwt/v5 | 无状态、跨域友好 |
| 密码加密 | bcrypt | crypto/bcrypt | 业界标准、防彩虹表 |
| K8s 客户端 | client-go | 0.28.0 | 官方库、功能完整 |

### 3.2 技术选型对比

#### 3.2.1 为什么选 Gin 而不是其他框架？

| 框架 | 性能 | 生态 | 学习曲线 | 结论 |
|------|-----|-----|---------|------|
| Gin | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 低 | ✅ 选择 |
| Echo | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | 低 | 备选 |
| Fiber | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | 中 | 生态不如 Gin |
| Beego | ⭐⭐⭐ | ⭐⭐⭐⭐ | 高 | 过于重量级 |

#### 3.2.2 为什么用 client-go 而不是直接 HTTP 调用 K8s API？

| 方案 | 优势 | 劣势 | 结论 |
|------|-----|-----|------|
| client-go | 类型安全、自动认证、重试机制 | 学习成本高 | ✅ 选择 |
| HTTP API | 简单直接 | 需要手动处理认证、序列化、错误 | ❌ 不推荐 |

---

## 4. 模块设计

### 4.1 用户模块（User）

#### 4.1.1 功能列表
- 用户注册（用户名/密码/邮箱）
- 用户登录（返回 JWT Token）
- 用户信息查询（未来扩展）

#### 4.1.2 核心流程

**注册流程**：
```
1. Handler 接收请求，参数校验
   ↓
2. Service 检查用户名/邮箱是否重复
   ↓
3. bcrypt 加密密码
   ↓
4. Repository 写入数据库
   ↓
5. 返回成功响应
```

**登录流程**：
```
1. Handler 接收用户名/密码
   ↓
2. Service 查询用户
   ↓
3. bcrypt 验证密码
   ↓
4. 生成 JWT Token（有效期 24h）
   ↓
5. 返回 Token 和 User UUID
```

#### 4.1.3 安全设计
- 密码使用 bcrypt 加密（cost=10）
- JWT 签名使用 HMAC-SHA256
- Token 包含 user_id 和 exp（过期时间）
- 登录失败不区分"用户不存在"和"密码错误"（防信息泄露）

### 4.2 应用模块（App）

#### 4.2.1 功能列表
- 创建应用（指定镜像、副本数、端口）
- 删除应用
- 启动应用（Scale to N）
- 停止应用（Scale to 0）
- 重启应用（Rolling Restart）
- 查看应用列表
- 查看应用详情（包含 Pod 状态）
- 查看应用日志

#### 4.2.2 核心流程

**创建应用流程**：
```
┌─────────────────────────────────────────────────────────────┐
│  1. Handler 接收请求                                         │
│     - name: "my-app"                                        │
│     - image: "nginx:latest"                                 │
│     - replicas: 2                                           │
│     - port: 80                                              │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│  2. Service 层处理                                           │
│     ┌─────────────────────────────────────────────────┐     │
│     │ 2.1 校验项目归属                                │     │
│     └────────────┬────────────────────────────────────┘     │
│                  │                                           │
│     ┌────────────▼────────────────────────────────────┐     │
│     │ 2.2 检查应用名是否重复（同一项目内）            │     │
│     └────────────┬────────────────────────────────────┘     │
│                  │                                           │
│     ┌────────────▼────────────────────────────────────┐     │
│     │ 2.3 开启事务并锁定项目，读取项目 Namespace       │     │
│     │     写入尚未提交的非空 project_id 应用记录       │     │
│     └────────────┬────────────────────────────────────┘     │
│                  │                                           │
│     ┌────────────▼────────────────────────────────────┐     │
│     │ 2.4 在事务回调内调用 Adapter.CreateApp()         │     │
│     │     - 创建 Deployment                            │     │
│     │     - 创建 Service（如果有端口）                 │     │
│     └────────────┬────────────────────────────────────┘     │
│                  │                                           │
│     ┌────────────▼────────────────────────────────────┐     │
│     │ 2.5 提交事务，应用记录此时才对其他请求可见       │     │
│     │     提交失败则删除已创建的 Kubernetes 资源       │     │
│     └────────────┬────────────────────────────────────┘     │
│                  │                                           │
│     ┌────────────▼────────────────────────────────────┐     │
│     │ 2.6 异步同步状态（Goroutine）                    │     │
│     └─────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│  3. K8s Adapter 执行                                         │
│     ┌─────────────────────────────────────────────────┐     │
│     │ 3.1 构建 Deployment YAML                         │     │
│     │     - metadata: name, namespace, labels         │     │
│     │     - spec: replicas, selector, template        │     │
│     │     - container: image, ports, resources        │     │
│     └────────────┬────────────────────────────────────┘     │
│                  │                                           │
│     ┌────────────▼────────────────────────────────────┐     │
│     │ 3.2 调用 K8s API: CreateDeployment()            │     │
│     └────────────┬────────────────────────────────────┘     │
│                  │                                           │
│     ┌────────────▼────────────────────────────────────┐     │
│     │ 3.3 构建 Service YAML（如果有端口）             │     │
│     └────────────┬────────────────────────────────────┘     │
│                  │                                           │
│     ┌────────────▼────────────────────────────────────┐     │
│     │ 3.4 调用 K8s API: CreateService()               │     │
│     └─────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│  4. K8s 集群执行                                             │
│     ┌─────────────────────────────────────────────────┐     │
│     │ 4.1 Scheduler 调度 Pod 到节点                    │     │
│     └────────────┬────────────────────────────────────┘     │
│                  │                                           │
│     ┌────────────▼────────────────────────────────────┐     │
│     │ 4.2 Kubelet 拉取镜像并启动容器                   │     │
│     └────────────┬────────────────────────────────────┘     │
│                  │                                           │
│     ┌────────────▼────────────────────────────────────┐     │
│     │ 4.3 Readiness Probe 检查容器状态                 │     │
│     └────────────┬────────────────────────────────────┘     │
│                  │                                           │
│     ┌────────────▼────────────────────────────────────┐     │
│     │ 4.4 Service 更新 Endpoints                       │     │
│     └─────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│  5. 状态同步（异步）                                         │
│     ┌─────────────────────────────────────────────────┐     │
│     │ 5.1 Adapter.GetAppStatus()                       │     │
│     │     - 查询 Deployment.Status                     │     │
│     │     - 查询 Pod 列表                              │     │
│     └────────────┬────────────────────────────────────┘     │
│                  │                                           │
│     ┌────────────▼────────────────────────────────────┐     │
│     │ 5.2 更新数据库                                   │     │
│     │     - status: "Running" / "Pending" / "Failed"  │     │
│     │     - replicas: 实际副本数                       │     │
│     └─────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────┘
```

Kubernetes 创建失败会回滚未提交的 App 记录。由于失败请求也可能已创建部分资源，Service 从调用开始即在事务失败时使用同名、同 Namespace 的幂等删除进行补偿；资源不存在时 NotFound 视为成功。这样并发删除无法观察到只有数据库记录、尚无 Kubernetes 资源的中间状态。

#### 4.2.3 多租户隔离设计

**命名空间策略**：
- 每个项目分配稳定的独立命名空间：`astro-project-<UUID>`。
- 项目内应用共享该 Namespace；创建项目时建立，删除空项目时删除。

**资源命名策略**：
- Deployment 与可选 Service 使用应用名。
- 应用名在同一项目内唯一，不同项目可重名。

**好处**：
- ✅ 避免命名冲突（不同项目可以创建同名应用）
- ✅ 资源隔离（网络策略、资源配额）
- ✅ 便于清理（仅空项目可删除，删除时同步删除 Namespace）

#### 4.2.4 状态同步机制

**同步策略**：
```go
// 创建应用后立即异步同步一次
go syncAppStatus(app.ID, app.Name, project.Namespace)

// 查询应用列表时异步同步所有应用
for _, app := range apps {
    go syncAppStatus(app.ID, app.Name, project.Namespace)
}

// 查询单个应用时同步等待
syncAppStatus(app.ID, app.Name, project.Namespace)
app = repo.GetByProjectAndID(project.ID, app.ID) // 重新查询
```

**状态定义**：
| 状态 | 含义 | 触发条件 |
|-----|------|---------|
| pending | 等待中 | 刚创建，Pod 未就绪 |
| running | 运行中 | ReadyReplicas == Replicas |
| stopped | 已停止 | Replicas == 0 |
| starting | 启动中 | 正在扩容 |
| restarting | 重启中 | 触发了滚动更新 |
| unknown | 未知 | K8s 查询失败 |

---

## 5. 接口设计

### 5.1 RESTful API 规范

**基础规范**：
- 基础路径：`/api/v1`
- 请求格式：`Content-Type: application/json`
- 响应格式：统一的 Response 结构

**统一响应结构**：
```json
{
  "code": 0,              // 错误码，0 表示成功
  "message": "成功",      // 消息描述
  "data": {}             // 业务数据（可选）
}
```

### 5.2 用户相关接口

#### 5.2.1 用户注册

```
POST /api/v1/register
```

**请求参数**：
```json
{
  "username": "johndoe",        // 必填，3-20 字符
  "password": "password123",    // 必填，6-30 字符
  "email": "john@example.com"   // 必填，邮箱格式
}
```

**成功响应**：
```json
{
  "code": 0,
  "message": "注册成功",
  "data": null
}
```

**错误响应**：
```json
{
  "code": 20001,
  "message": "用户已存在"
}
```

#### 5.2.2 用户登录

```
POST /api/v1/login
```

**请求参数**：
```json
{
  "username": "johndoe",
  "password": "password123"
}
```

**成功响应**：
```json
{
  "code": 0,
  "message": "登录成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "uuid": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

### 5.3 项目与应用接口

所有接口都要求 `Authorization: Bearer {token}`，并在 Service 层校验项目所有权。

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/projects` | 创建项目及独立 Namespace |
| GET | `/api/v1/projects` | 获取当前用户的项目列表 |
| GET | `/api/v1/projects/{project_id}` | 获取项目详情 |
| DELETE | `/api/v1/projects/{project_id}` | 删除空项目及其 Namespace |
| POST | `/api/v1/projects/{project_id}/apps` | 在项目内创建应用 |
| GET | `/api/v1/projects/{project_id}/apps` | 获取项目应用列表 |
| GET | `/api/v1/projects/{project_id}/apps/{id}` | 获取应用详情 |
| DELETE | `/api/v1/projects/{project_id}/apps/{id}` | 删除应用 |
| POST | `/api/v1/projects/{project_id}/apps/{id}/start` | 启动应用 |
| POST | `/api/v1/projects/{project_id}/apps/{id}/stop` | 停止应用 |
| POST | `/api/v1/projects/{project_id}/apps/{id}/restart` | 重启应用 |
| GET | `/api/v1/projects/{project_id}/apps/{id}/logs?lines=100` | 获取最近 1-1000 行日志 |

项目创建请求只包含 `name`。应用创建请求包含 `name`、`image`、`replicas` 与可选 `port`；项目由路径参数确定，应用响应使用 `project_id` 表示唯一归属，不再复制用户 ID 或 Namespace。

Web 入口与 API 保持相同层级：`/projects` → `/projects/:projectId/apps` → 应用详情。未创建项目的用户只能看到创建项目引导。

### 5.4 错误码定义

| 错误码 | 含义 | HTTP 状态码 |
|-------|------|-----------|
| 0 | 成功 | 200 |
| 10001 | 请求参数错误 | 200 |
| 10002 | 未登录或 Token 无效 | 200 |
| 10003 | 无权限访问 | 200 |
| 20001 | 用户已存在 | 200 |
| 20009 | 登录失败 | 200 |
| 21001 | 应用不存在 | 200 |
| 21002 | 应用已存在 | 200 |
| 21003 | 创建应用失败 | 200 |
| 22001 | 项目不存在 | 200 |
| 22002 | 项目已存在 | 200 |
| 22003 | 项目仍包含应用 | 200 |
| 22004 | 创建项目失败 | 200 |
| 30001 | 服务器内部错误 | 200 |
| 30002 | 数据库错误 | 200 |
| 30003 | K8s 操作错误 | 200 |
| 30004 | K8s 连接失败 | 200 |
| 30005 | K8s 操作失败 | 200 |

**注意**：所有错误都返回 HTTP 200，通过 `code` 字段区分成功/失败。

---

## 6. 数据模型

### 6.1 用户表（users）

```sql
CREATE TABLE users (
  id            INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  uuid          CHAR(36) UNIQUE NOT NULL COMMENT 'UUID',
  username      VARCHAR(64) UNIQUE NOT NULL COMMENT '用户名',
  password      VARCHAR(128) NOT NULL COMMENT '密码（bcrypt加密）',
  email         VARCHAR(128) UNIQUE COMMENT '邮箱',
  status        TINYINT DEFAULT 1 COMMENT '状态：1-正常，0-禁用',
  created_at    DATETIME NOT NULL COMMENT '创建时间',
  updated_at    DATETIME NOT NULL COMMENT '更新时间',
  deleted_at    DATETIME COMMENT '删除时间（软删除）',

  INDEX idx_username (username),
  INDEX idx_email (email),
  INDEX idx_uuid (uuid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';
```

**字段说明**：
- `uuid`: 对外暴露的用户标识（防止 ID 泄露）
- `password`: bcrypt 加密后的密码（60 字符）
- `deleted_at`: 软删除标记（GORM 自动处理）

### 6.2 项目表（projects）

```sql
CREATE TABLE projects (
  id            INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name          VARCHAR(64) NOT NULL COMMENT '项目名称',
  user_id       INT UNSIGNED NOT NULL COMMENT '所属用户ID',
  namespace     VARCHAR(63) UNIQUE NOT NULL COMMENT 'K8s命名空间',
  created_at    DATETIME NOT NULL,
  updated_at    DATETIME NOT NULL,
  deleted_at    DATETIME,
  UNIQUE KEY idx_projects_user_name (name, user_id),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='项目表';
```

项目名在用户内唯一；Namespace 使用 `astro-project-<UUID>`，全局唯一且不随项目名称变化。

### 6.3 应用表（apps）

```sql
CREATE TABLE apps (
  id            INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name          VARCHAR(64) NOT NULL COMMENT '应用名称',
  image         VARCHAR(256) NOT NULL COMMENT '镜像地址',
  replicas      INT DEFAULT 1 COMMENT '副本数',
  status        VARCHAR(32) DEFAULT 'stopped' COMMENT '状态：pending/running/stopped/starting/restarting/unknown',
  project_id    INT UNSIGNED NOT NULL COMMENT '所属项目ID',
  created_at    DATETIME NOT NULL COMMENT '创建时间',
  updated_at    DATETIME NOT NULL COMMENT '更新时间',
  deleted_at    DATETIME COMMENT 'GORM 基础字段；业务删除使用硬删除',

  UNIQUE KEY idx_apps_project_name (name, project_id),
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='应用表';
```

**字段说明**：
- `project_id`: 必填外键，应用的用户归属和 Namespace 均通过项目确定。
- `idx_apps_project_name`: 同一项目内应用名唯一；应用删除使用硬删除，因此名称可复用。

### 6.4 ER 图

```
┌───────────────────┐
│      users        │
├───────────────────┤
│ id (PK)           │
│ uuid (UK)         │
│ username (UK)     │
│ password          │
│ email (UK)        │
│ status            │
│ created_at        │
│ updated_at        │
│ deleted_at        │
└─────────┬─────────┘
          │ 1
          │
          │ owns
          │
          │ N
┌─────────▼─────────┐
│     projects      │
├───────────────────┤
│ id (PK)           │
│ user_id (FK)      │
│ name (UK/user)    │
│ namespace (UK)    │
│ created_at        │
│ updated_at        │
│ deleted_at        │
└─────────┬─────────┘
          │ 1
          │ contains
          │ N
┌─────────▼─────────┐
│       apps        │
├───────────────────┤
│ id (PK)           │
│ project_id (FK)   │
│ name (UK/project) │
│ image             │
│ replicas          │
│ status            │
└───────────────────┘
```

### 6.5 旧模型切换门禁

切换前在测试环境运行 `make legacy-inventory`，只读列出旧 `apps` 活动记录以及带 `managed-by=astro` 标签的 `astro-user-*` Namespace。该命令要求 `ASTRO_RUNTIME_ENV=test` 和 `ASTRO_DATABASE_PORT`，按宿主端口唯一定位运行中的 Docker 数据库容器，并在容器内使用现有客户端与凭据。Kubernetes 优先使用显式 kubeconfig 或 kubectl 默认配置，否则仅在本机恰有一个 kind 集群时回退；所有路径都只接受 `kind-*` context。盘点只执行 SQL `SELECT` 与 `kubectl get`；用户按清单确认后，再通过 `make legacy-delete-namespace LEGACY_NAMESPACE=astro-user-<数字>` 定点删除已复核为空的旧 Namespace。

服务启动时仅识别同时包含 `user_id`、`namespace` 且不包含 `project_id` 的旧 `apps` 表。存在活动 App 时拒绝启动；表为空时删除旧表，再由 `AutoMigrate` 建立带必填项目外键的新表。不会自动创建默认项目，也不会迁移旧应用。

---

## 7. 部署方案

### 7.1 部署架构

```
┌──────────────────────────────────────────────────────────────┐
│                       Kubernetes 集群                         │
│                                                               │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  Namespace: astro-system                               │  │
│  │                                                         │  │
│  │  ┌──────────────────────┐  ┌───────────────────────┐  │  │
│  │  │  Astro API           │  │  MariaDB              │  │  │
│  │  │  Deployment          │  │  StatefulSet          │  │  │
│  │  │  - replicas: 3       │  │  - replicas: 1        │  │  │
│  │  │  - HPA enabled       │  │  - PV: 20GB           │  │  │
│  │  └──────────┬───────────┘  └───────────────────────┘  │  │
│  │             │                                          │  │
│  │  ┌──────────▼───────────┐                             │  │
│  │  │  Service             │                             │  │
│  │  │  Type: ClusterIP     │                             │  │
│  │  └──────────┬───────────┘                             │  │
│  └─────────────┼──────────────────────────────────────────┘  │
│                │                                              │
│  ┌─────────────▼──────────────────────────────────────────┐  │
│  │  Ingress Controller (Nginx)                            │  │
│  │  - TLS 证书                                             │  │
│  │  - 路由规则: api.astro.com → astro-api                 │  │
│  └─────────────┬──────────────────────────────────────────┘  │
│                │                                              │
└────────────────┼──────────────────────────────────────────────┘
                 │
                 │ HTTPS
                 │
         ┌───────▼───────┐
         │   Internet    │
         └───────────────┘
```

### 7.2 资源配置

#### 7.2.1 Astro API Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: astro-api
  namespace: astro-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: astro-api
  template:
    metadata:
      labels:
        app: astro-api
    spec:
      serviceAccountName: astro-api
      containers:
      - name: api
        image: astro/api:v1.0.0
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          value: mariadb.astro-system.svc
        - name: DB_PORT
          value: "3306"
        - name: DB_USER
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: username
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: password
        resources:
          requests:
            memory: "256Mi"
            cpu: "200m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

#### 7.2.2 MariaDB StatefulSet

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mariadb
  namespace: astro-system
spec:
  serviceName: mariadb
  replicas: 1
  selector:
    matchLabels:
      app: mariadb
  template:
    metadata:
      labels:
        app: mariadb
    spec:
      containers:
      - name: mariadb
        image: mariadb:10.6
        ports:
        - containerPort: 3306
        env:
        - name: MYSQL_ROOT_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: root-password
        - name: MYSQL_DATABASE
          value: astro
        volumeMounts:
        - name: data
          mountPath: /var/lib/mysql
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 20Gi
```

### 7.3 RBAC 配置

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: astro-api
  namespace: astro-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: astro-api-role
rules:
# 管理 Deployment
- apiGroups: ["apps"]
  resources: ["deployments", "deployments/scale"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
# 管理 Service
- apiGroups: [""]
  resources: ["services"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
# 查看 Pods
- apiGroups: [""]
  resources: ["pods", "pods/log"]
  verbs: ["get", "list", "watch"]
# 管理命名空间
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list", "create", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: astro-api-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: astro-api-role
subjects:
- kind: ServiceAccount
  name: astro-api
  namespace: astro-system
```

---

## 8. 安全设计

### 8.1 认证与授权

#### 8.1.1 JWT 认证流程

```
1. 用户登录 → 验证用户名/密码
   ↓
2. 生成 JWT Token
   - Header: {"alg": "HS256", "typ": "JWT"}
   - Payload: {"user_id": 123, "uuid": "xxx", "exp": 1702300800}
   - Signature: HMAC-SHA256(base64(header) + "." + base64(payload), secret)
   ↓
3. 返回 Token 给客户端
   ↓
4. 客户端在后续请求中携带 Token
   - Header: Authorization: Bearer {token}
   ↓
5. 中间件验证 Token
   - 检查签名是否有效
   - 检查是否过期
   - 提取 user_id 到 Context
   ↓
6. Handler 从 Context 获取 user_id
```

#### 8.1.2 权限检查

**资源所有权检查**：
```go
// 所有应用操作先验证项目归属，再按 project_id + app_id 查询应用。
func (s *AppService) DeleteApp(ctx context.Context, projectID, appID, userID uint) error {
    project, err := s.getOwnedProject(projectID, userID)
    if err != nil {
        return err
    }
    // 后续只使用 project.Namespace 定位 Kubernetes 资源。
    ...
}
```

### 8.2 输入验证

**参数校验**（使用 Gin 的 binding）：
```go
type RegisterRequest struct {
    Username string `json:"username" binding:"required,min=3,max=20"`
    Password string `json:"password" binding:"required,min=6,max=30"`
    Email    string `json:"email" binding:"required,email"`
}
```

**防止注入攻击**：
- SQL 注入：使用 GORM 的参数化查询
- XSS：前端对用户输入进行转义
- CSRF：使用 Token 验证

### 8.3 安全配置

#### 8.3.1 JWT Secret 管理

**开发环境**：
```yaml
# configs/config.yaml
jwt:
  secret: "astro-secret-key"  # 固定值，方便开发
  expire: "24h"
```

**生产环境**：
```yaml
# 使用 K8s Secret
apiVersion: v1
kind: Secret
metadata:
  name: jwt-secret
  namespace: astro-system
type: Opaque
data:
  secret: <base64编码的随机字符串>
```

```go
// 从环境变量读取
jwtSecret := os.Getenv("JWT_SECRET")
if jwtSecret == "" {
    jwtSecret = config.GlobalConfig.JWT.Secret
}
```

#### 8.3.2 数据库密码管理

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: db-secret
  namespace: astro-system
type: Opaque
stringData:
  username: astro
  password: <生成的强密码>
  root-password: <生成的强密码>
```

### 8.4 网络安全

#### 8.4.1 网络策略

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: astro-api-network-policy
  namespace: astro-system
spec:
  podSelector:
    matchLabels:
      app: astro-api
  policyTypes:
  - Ingress
  - Egress
  ingress:
  # 只允许 Ingress Controller 访问
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  egress:
  # 允许访问数据库
  - to:
    - podSelector:
        matchLabels:
          app: mariadb
    ports:
    - protocol: TCP
      port: 3306
  # 允许访问 K8s API Server
  - to:
    - namespaceSelector: {}
      podSelector: {}
    ports:
    - protocol: TCP
      port: 443
```

#### 8.4.2 多租户网络隔离

```yaml
# 每个项目的命名空间都有独立的网络策略
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-all
  namespace: astro-project-550e8400-e29b-41d4-a716-446655440000
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-same-namespace
  namespace: astro-project-550e8400-e29b-41d4-a716-446655440000
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector: {}
  egress:
  - to:
    - podSelector: {}
```

---

## 9. 监控与运维

### 9.1 日志设计

#### 9.1.1 日志级别

| 级别 | 使用场景 | 示例 |
|-----|---------|------|
| Debug | 调试信息 | `logger.Debug("查询用户", zap.String("username", "john"))` |
| Info | 常规操作 | `logger.Info("用户登录成功", zap.Uint("user_id", 123))` |
| Warn | 异常但可处理 | `logger.Warn("K8s客户端初始化失败", zap.Error(err))` |
| Error | 错误需要关注 | `logger.Error("创建应用失败", zap.Uint("app_id", 1), zap.Error(err))` |
| Fatal | 致命错误退出 | `logger.Fatal("数据库连接失败", zap.Error(err))` |

#### 9.1.2 结构化日志

**好的日志示例**：
```go
logger.Info("创建应用",
    zap.Uint("user_id", req.UserID),
    zap.String("app_name", req.Name),
    zap.String("image", req.Image),
    zap.Int("replicas", req.Replicas),
)
```

**输出**：
```json
{
  "level": "info",
  "time": "2025-12-11T10:00:00+08:00",
  "caller": "service/app.go:123",
  "msg": "创建应用",
  "user_id": 123,
  "app_name": "my-nginx",
  "image": "nginx:latest",
  "replicas": 2
}
```

#### 9.1.3 日志轮转

```yaml
# configs/config.yaml
log:
  level: info               # 生产环境使用 info
  file: logs/astro.log
  max_size: 100             # 单文件最大 100MB
  max_backups: 10           # 保留 10 个历史文件
  max_age: 30               # 保留 30 天
  compress: true            # 启用压缩
```

### 9.2 监控指标

#### 9.2.1 业务指标

| 指标 | 说明 | 监控方式 |
|-----|------|---------|
| 用户总数 | 注册用户数量 | 查询数据库 |
| 活跃用户数 | 最近 7 天登录 | 查询数据库 |
| 应用总数 | 所有应用数量 | 查询数据库 |
| 运行中应用数 | status=running | 查询数据库 |
| Pod 总数 | 所有 Pod 数量 | K8s API |

#### 9.2.2 性能指标

| 指标 | 说明 | 告警阈值 |
|-----|------|---------|
| API 响应时间 | P50/P95/P99 | P95 > 1s |
| API 错误率 | 5xx 错误占比 | > 1% |
| 数据库连接数 | 活跃连接数 | > 80% |
| 数据库慢查询 | > 1s 的查询 | > 10 次/分钟 |
| K8s API 调用延迟 | 调用耗时 | > 500ms |

### 9.3 健康检查

#### 9.3.1 Liveness Probe（存活探针）

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
  failureThreshold: 3
```

**健康检查接口**：
```go
// 简单检查（服务是否存活）
r.GET("/health", func(c *gin.Context) {
    c.JSON(200, gin.H{"status": "ok"})
})
```

#### 9.3.2 Readiness Probe（就绪探针）

```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  failureThreshold: 3
```

**就绪检查接口**：
```go
// 深度检查（数据库/K8s 是否可用）
r.GET("/ready", func(c *gin.Context) {
    // 检查数据库连接
    if err := repository.DB.Ping(); err != nil {
        c.JSON(503, gin.H{"status": "db_unavailable"})
        return
    }

    // 检查 K8s 连接（可选）
    if !k8s.IsAvailable() {
        c.JSON(503, gin.H{"status": "k8s_unavailable"})
        return
    }

    c.JSON(200, gin.H{"status": "ready"})
})
```

### 9.4 运维命令

#### 9.4.1 查看日志

```bash
# 查看 API 日志
kubectl logs -f -n astro-system deployment/astro-api

# 查看最近 100 行
kubectl logs -n astro-system deployment/astro-api --tail=100

# 查看所有 Pod 的日志
kubectl logs -n astro-system -l app=astro-api --all-containers=true
```

#### 9.4.2 扩容缩容

```bash
# 扩容到 5 个副本
kubectl scale deployment astro-api -n astro-system --replicas=5

# 查看 HPA 状态
kubectl get hpa -n astro-system
```

#### 9.4.3 滚动更新

```bash
# 更新镜像
kubectl set image deployment/astro-api astro-api=astro/api:v1.1.0 -n astro-system

# 查看更新状态
kubectl rollout status deployment/astro-api -n astro-system

# 回滚到上一版本
kubectl rollout undo deployment/astro-api -n astro-system
```

---

## 10. 风险与挑战

### 10.1 技术风险

| 风险 | 影响 | 概率 | 应对措施 |
|-----|------|-----|---------|
| K8s 集群不稳定 | 应用创建失败 | 中 | 增加重试机制，优雅降级 |
| 数据库性能瓶颈 | 响应变慢 | 中 | 读写分离，缓存热点数据 |
| 用户应用耗尽集群资源 | 影响其他用户 | 高 | 资源配额（ResourceQuota） |
| 恶意用户创建大量应用 | 资源浪费 | 中 | 限制单用户应用数量 |
| 日志文件占满磁盘 | 服务崩溃 | 低 | 日志轮转，定期清理 |

### 10.2 安全风险

| 风险 | 影响 | 概率 | 应对措施 |
|-----|------|-----|---------|
| JWT Secret 泄露 | 用户身份伪造 | 低 | 使用 K8s Secret，定期轮换 |
| SQL 注入 | 数据泄露 | 低 | 使用 ORM 参数化查询 |
| XSS 攻击 | 用户信息泄露 | 中 | 前端输入转义 |
| 用户跨租户访问 | 数据泄露 | 中 | 严格的权限检查 |
| 恶意镜像 | 挖矿、攻击 | 高 | 镜像白名单，安全扫描 |

### 10.3 业务风险

| 风险 | 影响 | 概率 | 应对措施 |
|-----|------|-----|---------|
| 用户部署挖矿程序 | 资源浪费，成本增加 | 高 | CPU 限额，异常检测 |
| 用户应用占用过多内存 | OOM，影响其他应用 | 中 | 内存限额，自动重启 |
| 用户应用间互相访问 | 数据泄露 | 中 | 网络策略隔离 |
| 用户删除应用后数据恢复 | 用户投诉 | 低 | 软删除，保留数据 7 天 |

### 10.4 可扩展性挑战

| 挑战 | 场景 | 解决方案 |
|-----|------|---------|
| 单个 K8s 集群资源不足 | 用户量 > 1000 | 多集群管理，智能调度 |
| 状态同步延迟 | 应用列表刷新慢 | 引入 K8s Informer |
| 数据库写入瓶颈 | 高并发创建应用 | 分库分表，异步写入 |
| 应用日志查询慢 | 日志量过大 | 引入 ELK 或 Loki |

---

## 附录

### A. 术语表

| 术语 | 全称 | 解释 |
|-----|------|-----|
| CaaS | Container as a Service | 容器即服务 |
| K8s | Kubernetes | 容器编排平台 |
| CRD | Custom Resource Definition | 自定义资源定义 |
| JWT | JSON Web Token | 无状态认证令牌 |
| RBAC | Role-Based Access Control | 基于角色的访问控制 |
| HPA | Horizontal Pod Autoscaler | 水平 Pod 自动扩缩容 |
| ORM | Object-Relational Mapping | 对象关系映射 |

### B. 参考文档

- [Kubernetes 官方文档](https://kubernetes.io/docs/)
- [client-go 文档](https://pkg.go.dev/k8s.io/client-go)
- [Gin 框架文档](https://gin-gonic.com/docs/)
- [GORM 文档](https://gorm.io/docs/)
- [JWT 规范](https://jwt.io/introduction)

### C. 版本历史

| 版本 | 日期 | 作者 | 变更说明 |
|-----|------|------|---------|
| v1.0 | 2025-12-11 | Claude | 初始版本 |

---

**文档结束**
