## Session 2: Astro 主机控制面恢复检查点

**Date**: 2026-08-21  
**Task**: 建立 Astro 主机部署控制面  
**Branch**: `main`

### 已持久化状态

- 可恢复检查点：`/root/base-workspace/backup/applications/astro/20260821T035452Z-host-control-plane-final-checkpoint/`
- 检查点为 `root:root`、目录 `700`、文件 `600`，`MANIFEST.sha256` 已完整校验。
- 内容包含：Astro Compose/部署/回滚/状态、运行时秘密、`astro_test` 与 `astro_production` 的一致性 MariaDB dump 及授权、Astro TLS 归档、OpenResty、SSH/sudo、网络和语法验证证据。
- 主机 README 已记录恢复检查点；生产继续保持 `enabled=false` 且没有独立生产 kubeconfig。

### 恢复后续工作

1. 先核验检查点 manifest，再修改任何主机控制面文件。
2. 后续 `astro-image-release-pipeline` 发布首个有效 ARM64 GHCR digest 后，执行首次测试部署、成功 state 写入及故障 digest 自动回滚演练。
3. 不要把主机 Compose、OpenResty、部署脚本、状态、kubeconfig 或秘密写入 Astro Git 仓库。
