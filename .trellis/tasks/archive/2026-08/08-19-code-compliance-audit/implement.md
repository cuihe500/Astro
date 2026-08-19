# 代码规范合规审查执行计划

## 顺序清单

1. 完成仓库、模块、路由和规范文件清单。
2. 阅读并核对启动、配置、日志、错误码、响应、中间件、用户/OAuth2、应用 service、repository、K8s adapter 和模型迁移代码。
3. 执行静态规则扫描，并对每个命中回溯上下文与调用链，建立候选发现表。
4. 检查跨层数据流：handler 参数与身份 → service 权限/错误转换 → repository/K8s → handler 响应。
5. 检查 OAuth2 安全契约和敏感信息边界，核对配置示例及 Swagger 产物。
6. 使用 Makefile 目标运行 `make build`、`make test`、`make lint`；必要时运行 `make swagger` 做一致性验证。不得用手拼等价命令替代 Makefile 目标。
7. 对候选发现进行二次证据核验，去除误报，按 P0-P3 排序。
8. 将完整结果写入 `report.md`，复核报告与 Git 状态，向用户报告结论。

## 验证命令

```bash
make build
make test
make lint
make swagger
```

`make lint` 或 `make swagger` 若受本机工具、网络或外部服务影响，记录原始失败原因及影响范围；不擅自修改环境或业务代码。

## 风险与回滚点

- 本任务预期只新增任务规划和 `report.md`；不触碰业务源码、配置和生成 Swagger 文件。
- 若运行 `make swagger` 会改写生成物，先记录工作树并在结果中区分工具生成差异；除非用户明确要求，不保留生成物改动。
- 若审查结论依赖数据库/Kubernetes 实例而当前环境不可用，标为“静态审查未验证”，不得推断运行时行为。
- 报告写入错误时只重写任务目录内报告，不回滚用户已有改动。
