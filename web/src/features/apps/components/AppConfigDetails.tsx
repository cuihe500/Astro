import { hasAppConfig } from "../config";
import type { AppConfig, Probe } from "../types";

function Value({ children }: { children: React.ReactNode }) {
  return <span className="config-value">{children}</span>;
}

function ProbeValue({ name, probe }: { name: string; probe?: Probe }) {
  if (!probe) return null;
  let action = "";
  if (probe.http_get) action = `${probe.http_get.scheme ?? "HTTP"} ${probe.http_get.path} :${probe.http_get.port}`;
  if (probe.tcp_socket) action = `TCP :${probe.tcp_socket.port}`;
  if (probe.exec) action = `Exec ${probe.exec.command.join(" ")}`;
  const headers = probe.http_get?.http_headers?.map((header) => `${header.name}: ${header.value}`).join(" · ");
  const timing = [
    probe.initial_delay_seconds !== undefined ? `延迟 ${probe.initial_delay_seconds}s` : "",
    probe.period_seconds !== undefined ? `周期 ${probe.period_seconds}s` : "",
    probe.timeout_seconds !== undefined ? `超时 ${probe.timeout_seconds}s` : "",
    probe.success_threshold !== undefined ? `成功 ${probe.success_threshold}` : "",
    probe.failure_threshold !== undefined ? `失败 ${probe.failure_threshold}` : "",
  ].filter(Boolean).join(" · ");
  return <li><strong>{name}</strong><Value>{action}{headers ? ` · Header ${headers}` : ""}{timing ? `（${timing}）` : ""}</Value></li>;
}

export function AppConfigDetails({ config }: { config?: AppConfig }) {
  if (!hasAppConfig(config) || !config) return null;
  const hasContainer = Boolean(config.command?.length || config.args?.length || config.working_dir || config.image_pull_policy);
  const hasEnvironment = Boolean(config.env?.length || config.env_from?.length);
  const hasResources = Boolean(config.resources?.requests?.cpu || config.resources?.requests?.memory || config.resources?.limits?.cpu || config.resources?.limits?.memory);
  const hasProbes = Boolean(config.startup_probe || config.readiness_probe || config.liveness_probe);
  const hasStorage = Boolean(config.volumes?.length || config.volume_mounts?.length);
  const security = config.security_context;
  const hasSecurity = Boolean(security || config.termination_grace_period_seconds !== undefined || config.image_pull_secrets?.length);

  return (
    <section className="detail-section" aria-labelledby="config-title">
      <div className="section-heading"><div><h2 id="config-title">高级配置</h2><p>创建时保存，只读展示；修改需删除并重建应用。</p></div></div>
      <div className="config-detail-groups">
        {hasContainer ? <div><h3>容器</h3><ul className="config-list">
          {config.command?.length ? <li><strong>命令</strong><Value>{config.command.join(" · ")}</Value></li> : null}
          {config.args?.length ? <li><strong>参数</strong><Value>{config.args.join(" · ")}</Value></li> : null}
          {config.working_dir ? <li><strong>工作目录</strong><Value>{config.working_dir}</Value></li> : null}
          {config.image_pull_policy ? <li><strong>拉取策略</strong><Value>{config.image_pull_policy}</Value></li> : null}
        </ul></div> : null}
        {hasEnvironment ? <div><h3>环境变量</h3><ul className="config-list">
          {config.env?.map((env, index) => <li key={`${env.name}-${index}`}><strong>{env.name}</strong><Value>{env.value !== undefined ? env.value : env.value_from?.secret_key_ref ? `Secret ${env.value_from.secret_key_ref.name}/${env.value_from.secret_key_ref.key}` : `ConfigMap ${env.value_from?.config_map_key_ref?.name}/${env.value_from?.config_map_key_ref?.key}`}</Value></li>)}
          {config.env_from?.map((source, index) => <li key={`env-from-${index}`}><strong>{source.prefix || "整项导入"}</strong><Value>{source.secret_ref ? `Secret ${source.secret_ref.name}` : `ConfigMap ${source.config_map_ref?.name}`}</Value></li>)}
        </ul><p className="field-help">敏感值仅显示 Secret 引用，不读取 Secret 内容。</p></div> : null}
        {hasResources ? <div><h3>资源</h3><ul className="config-list">
          {config.resources?.requests?.cpu ? <li><strong>CPU request</strong><Value>{config.resources.requests.cpu}</Value></li> : null}
          {config.resources?.limits?.cpu ? <li><strong>CPU limit</strong><Value>{config.resources.limits.cpu}</Value></li> : null}
          {config.resources?.requests?.memory ? <li><strong>内存 request</strong><Value>{config.resources.requests.memory}</Value></li> : null}
          {config.resources?.limits?.memory ? <li><strong>内存 limit</strong><Value>{config.resources.limits.memory}</Value></li> : null}
        </ul></div> : null}
        {config.ports?.length ? <div><h3>网络</h3><ul className="config-list">{config.ports.map((port) => <li key={port.name}><strong>{port.name}</strong><Value>{port.protocol ?? "TCP"} 容器 {port.container_port}{port.service_port !== undefined ? ` → Service ${port.service_port}` : "（不创建 Service 端口）"}</Value></li>)}</ul></div> : null}
        {hasProbes ? <div><h3>健康检查</h3><ul className="config-list"><ProbeValue name="启动探针" probe={config.startup_probe} /><ProbeValue name="就绪探针" probe={config.readiness_probe} /><ProbeValue name="存活探针" probe={config.liveness_probe} /></ul></div> : null}
        {hasStorage ? <div><h3>存储</h3><ul className="config-list">
          {config.volumes?.map((volume) => <li key={volume.name}><strong>卷 {volume.name}</strong><Value>{volume.empty_dir ? `emptyDir${volume.empty_dir.medium ? ` / ${volume.empty_dir.medium}` : ""}${volume.empty_dir.size_limit ? ` / ${volume.empty_dir.size_limit}` : ""}` : volume.persistent_volume_claim ? `PVC ${volume.persistent_volume_claim.claim_name}${volume.persistent_volume_claim.read_only ? "（只读）" : ""}` : volume.config_map ? `ConfigMap ${volume.config_map.name}` : `Secret ${volume.secret?.name}`}</Value></li>)}
          {config.volume_mounts?.map((mount, index) => <li key={`${mount.name}-${mount.mount_path}-${index}`}><strong>挂载 {mount.name}</strong><Value>{mount.mount_path}{mount.sub_path ? ` / ${mount.sub_path}` : ""}{mount.read_only ? "（只读）" : ""}</Value></li>)}
        </ul></div> : null}
        {hasSecurity ? <div><h3>安全与生命周期</h3><ul className="config-list">
          {security?.run_as_non_root !== undefined ? <li><strong>非 root</strong><Value>{security.run_as_non_root ? "是" : "否"}</Value></li> : null}
          {security?.run_as_user !== undefined ? <li><strong>用户 ID</strong><Value>{security.run_as_user}</Value></li> : null}
          {security?.run_as_group !== undefined ? <li><strong>组 ID</strong><Value>{security.run_as_group}</Value></li> : null}
          {security?.fs_group !== undefined ? <li><strong>文件系统组</strong><Value>{security.fs_group}</Value></li> : null}
          {security?.read_only_root_filesystem !== undefined ? <li><strong>只读根文件系统</strong><Value>{security.read_only_root_filesystem ? "是" : "否"}</Value></li> : null}
          {security?.allow_privilege_escalation !== undefined ? <li><strong>权限提升</strong><Value>禁止</Value></li> : null}
          {security?.drop_capabilities?.length ? <li><strong>移除 capabilities</strong><Value>{security.drop_capabilities.join(", ")}</Value></li> : null}
          {security?.seccomp_profile ? <li><strong>seccomp</strong><Value>{security.seccomp_profile}</Value></li> : null}
          {config.termination_grace_period_seconds !== undefined ? <li><strong>终止宽限期</strong><Value>{config.termination_grace_period_seconds} 秒</Value></li> : null}
          {config.image_pull_secrets?.length ? <li><strong>镜像拉取凭据</strong><Value>{config.image_pull_secrets.join(", ")}</Value></li> : null}
        </ul></div> : null}
      </div>
    </section>
  );
}
