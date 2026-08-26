import { Plus, Trash2 } from "lucide-react";
import { useRef, type RefObject } from "react";
import type { AppConfigDraft, EnvDraft, EnvFromDraft, PortDraft, ProbeDraft, VolumeDraft } from "../config";

interface AppConfigFieldsProps {
  draft: AppConfigDraft;
  errors: Record<string, string>;
  detailsRef: RefObject<HTMLDetailsElement | null>;
  onChange: (draft: AppConfigDraft) => void;
}

function replace<T>(items: T[], index: number, item: T): T[] {
  return items.map((current, currentIndex) => currentIndex === index ? item : current);
}

function ErrorText({ id, message }: { id: string; message?: string }) {
  return message ? <p className="field-error" id={id}>{message}</p> : null;
}

function ProbeFields({ id, label, value, error, onChange }: {
  id: string;
  label: string;
  value: ProbeDraft;
  error?: string;
  onChange: (probe: ProbeDraft) => void;
}) {
  const update = (patch: Partial<ProbeDraft>) => onChange({ ...value, ...patch });
  return (
    <fieldset className="nested-fieldset">
      <legend>{label}</legend>
      <div className="field">
        <label htmlFor={`${id}-type`}>探测方式</label>
        <select id={`${id}-type`} value={value.type} onChange={(event) => update({ type: event.target.value as ProbeDraft["type"] })} aria-invalid={Boolean(error)} aria-describedby={error ? `${id}-error` : undefined}>
          <option value="">不启用</option><option value="http">HTTP GET</option><option value="tcp">TCP Socket</option><option value="exec">Exec</option>
        </select>
      </div>
      {value.type === "http" ? (
        <>
          <div className="form-grid">
            <div className="field"><label htmlFor={`${id}-path`}>HTTP 路径</label><input id={`${id}-path`} value={value.path} onChange={(event) => update({ path: event.target.value })} placeholder="/health" /></div>
            <div className="field"><label htmlFor={`${id}-port`}>端口</label><input id={`${id}-port`} type="number" min="1" max="65535" value={value.port} onChange={(event) => update({ port: event.target.value })} /></div>
          </div>
          <div className="field"><label htmlFor={`${id}-scheme`}>协议</label><select id={`${id}-scheme`} value={value.scheme} onChange={(event) => update({ scheme: event.target.value as ProbeDraft["scheme"] })}><option value="HTTP">HTTP</option><option value="HTTPS">HTTPS</option></select></div>
          <div className="field"><label htmlFor={`${id}-headers`}>HTTP Header <span className="optional">选填，每行“名称: 值”</span></label><textarea id={`${id}-headers`} value={value.headers} onChange={(event) => update({ headers: event.target.value })} rows={3} /></div>
        </>
      ) : null}
      {value.type === "tcp" ? <div className="field"><label htmlFor={`${id}-port`}>TCP 端口</label><input id={`${id}-port`} type="number" min="1" max="65535" value={value.port} onChange={(event) => update({ port: event.target.value })} /></div> : null}
      {value.type === "exec" ? <div className="field"><label htmlFor={`${id}-command`}>Exec 命令 <span className="optional">每行一项</span></label><textarea id={`${id}-command`} value={value.command} onChange={(event) => update({ command: event.target.value })} rows={3} /></div> : null}
      {value.type ? (
        <div className="compact-grid">
          {[
            ["initialDelay", "初始延迟", "0"], ["period", "周期", "10"], ["timeout", "超时", "1"], ["success", "成功阈值", "1"], ["failure", "失败阈值", "3"],
          ].map(([key, text, placeholder]) => (
            <div className="field" key={key}>
              <label htmlFor={`${id}-${key}`}>{text}</label>
              <input id={`${id}-${key}`} type="number" min="0" value={value[key as keyof ProbeDraft] as string} placeholder={placeholder} onChange={(event) => update({ [key]: event.target.value })} />
            </div>
          ))}
        </div>
      ) : null}
      <ErrorText id={`${id}-error`} message={error} />
    </fieldset>
  );
}

export function AppConfigFields({ draft, errors, detailsRef, onChange }: AppConfigFieldsProps) {
  const nextID = useRef(1);
  const update = (patch: Partial<AppConfigDraft>) => onChange({ ...draft, ...patch });
  const addEnv = () => update({ env: [...draft.env, { id: nextID.current++, name: "", source: "value", value: "", refName: "", refKey: "" }] });
  const addEnvFrom = () => update({ envFrom: [...draft.envFrom, { id: nextID.current++, kind: "config_map", name: "", prefix: "" }] });
  const addPort = () => update({ ports: [...draft.ports, { id: nextID.current++, name: "", containerPort: "", protocol: "TCP", servicePort: "" }] });
  const addVolume = () => update({ volumes: [...draft.volumes, { id: nextID.current++, name: "", type: "empty_dir", refName: "", readOnly: false, medium: "", sizeLimit: "" }] });
  const addMount = () => update({ volumeMounts: [...draft.volumeMounts, { id: nextID.current++, name: "", mountPath: "", subPath: "", readOnly: false }] });

  return (
    <details className="advanced-config" ref={detailsRef}>
      <summary id="advanced-config-summary">高级配置 <span>按需展开</span></summary>
      <p className="advanced-help">仅支持安全白名单参数；引用的 ConfigMap、Secret 和 PVC 必须已存在于当前项目。</p>

      <fieldset className="config-group">
        <legend>容器</legend>
        <div className="field"><label htmlFor="config-command">启动命令 <span className="optional">每行一项</span></label><textarea id="config-command" value={draft.command} onChange={(event) => update({ command: event.target.value })} rows={3} aria-invalid={Boolean(errors.command)} aria-describedby={errors.command ? "config-command-error" : undefined} /><ErrorText id="config-command-error" message={errors.command} /></div>
        <div className="field"><label htmlFor="config-args">启动参数 <span className="optional">每行一项</span></label><textarea id="config-args" value={draft.args} onChange={(event) => update({ args: event.target.value })} rows={3} aria-invalid={Boolean(errors.args)} aria-describedby={errors.args ? "config-args-error" : undefined} /><ErrorText id="config-args-error" message={errors.args} /></div>
        <div className="form-grid">
          <div className="field"><label htmlFor="config-working-dir">工作目录</label><input id="config-working-dir" value={draft.workingDir} placeholder="/app" onChange={(event) => update({ workingDir: event.target.value })} aria-invalid={Boolean(errors.workingDir)} aria-describedby={errors.workingDir ? "config-working-dir-error" : undefined} /><ErrorText id="config-working-dir-error" message={errors.workingDir} /></div>
          <div className="field"><label htmlFor="config-pull-policy">镜像拉取策略</label><select id="config-pull-policy" value={draft.imagePullPolicy} onChange={(event) => update({ imagePullPolicy: event.target.value as AppConfigDraft["imagePullPolicy"] })}><option value="">使用 Kubernetes 默认值</option><option value="Always">Always</option><option value="IfNotPresent">IfNotPresent</option><option value="Never">Never</option></select></div>
        </div>
      </fieldset>

      <fieldset className="config-group">
        <legend>环境变量</legend>
        <div className="collection-heading"><p>单项环境变量</p><button className="button button-secondary button-small" type="button" onClick={addEnv}><Plus size={15} aria-hidden="true" />添加</button></div>
        <ErrorText id="config-env-error" message={errors.env} />
        {draft.env.map((item, index) => (
          <div className="repeat-card" key={item.id}>
            <div className="repeat-heading"><strong>环境变量 {index + 1}</strong><button className="icon-button icon-button-danger compact-button" type="button" aria-label={`删除环境变量 ${index + 1}`} onClick={() => update({ env: draft.env.filter((current) => current.id !== item.id) })}><Trash2 size={16} aria-hidden="true" /></button></div>
            <div className="compact-grid">
              <div className="field"><label htmlFor={`env-${item.id}-name`}>名称</label><input id={`env-${item.id}-name`} value={item.name} onChange={(event) => update({ env: replace(draft.env, index, { ...item, name: event.target.value }) })} aria-invalid={Boolean(errors[`env.${index}`])} aria-describedby={errors[`env.${index}`] ? `env-${item.id}-error` : undefined} /></div>
              <div className="field"><label htmlFor={`env-${item.id}-source`}>来源</label><select id={`env-${item.id}-source`} value={item.source} onChange={(event) => update({ env: replace(draft.env, index, { ...item, source: event.target.value as EnvDraft["source"] }) })}><option value="value">直接值</option><option value="config_map">ConfigMap 键</option><option value="secret">Secret 键</option></select></div>
              {item.source === "value" ? <div className="field field-wide"><label htmlFor={`env-${item.id}-value`}>值</label><input id={`env-${item.id}-value`} value={item.value} onChange={(event) => update({ env: replace(draft.env, index, { ...item, value: event.target.value }) })} /></div> : <><div className="field"><label htmlFor={`env-${item.id}-ref`}>资源名称</label><input id={`env-${item.id}-ref`} value={item.refName} onChange={(event) => update({ env: replace(draft.env, index, { ...item, refName: event.target.value }) })} /></div><div className="field"><label htmlFor={`env-${item.id}-key`}>键</label><input id={`env-${item.id}-key`} value={item.refKey} onChange={(event) => update({ env: replace(draft.env, index, { ...item, refKey: event.target.value }) })} /></div></>}
            </div>
            <ErrorText id={`env-${item.id}-error`} message={errors[`env.${index}`]} />
          </div>
        ))}
        <div className="collection-heading"><p>整项引用</p><button className="button button-secondary button-small" type="button" onClick={addEnvFrom}><Plus size={15} aria-hidden="true" />添加</button></div>
        {draft.envFrom.map((item, index) => (
          <div className="repeat-card" key={item.id}>
            <div className="repeat-heading"><strong>引用 {index + 1}</strong><button className="icon-button icon-button-danger compact-button" type="button" aria-label={`删除整项引用 ${index + 1}`} onClick={() => update({ envFrom: draft.envFrom.filter((current) => current.id !== item.id) })}><Trash2 size={16} aria-hidden="true" /></button></div>
            <div className="compact-grid">
              <div className="field"><label htmlFor={`env-from-${item.id}-kind`}>类型</label><select id={`env-from-${item.id}-kind`} value={item.kind} onChange={(event) => update({ envFrom: replace(draft.envFrom, index, { ...item, kind: event.target.value as EnvFromDraft["kind"] }) })}><option value="config_map">ConfigMap</option><option value="secret">Secret</option></select></div>
              <div className="field"><label htmlFor={`env-from-${item.id}-name`}>资源名称</label><input id={`env-from-${item.id}-name`} value={item.name} onChange={(event) => update({ envFrom: replace(draft.envFrom, index, { ...item, name: event.target.value }) })} aria-invalid={Boolean(errors[`envFrom.${index}`])} aria-describedby={errors[`envFrom.${index}`] ? `env-from-${item.id}-error` : undefined} /></div>
              <div className="field field-wide"><label htmlFor={`env-from-${item.id}-prefix`}>前缀 <span className="optional">选填</span></label><input id={`env-from-${item.id}-prefix`} value={item.prefix} onChange={(event) => update({ envFrom: replace(draft.envFrom, index, { ...item, prefix: event.target.value }) })} /></div>
            </div>
            <ErrorText id={`env-from-${item.id}-error`} message={errors[`envFrom.${index}`]} />
          </div>
        ))}
      </fieldset>

      <fieldset className="config-group">
        <legend>资源</legend>
        <div className="compact-grid">
          {[["requestCPU", "CPU request", "100m"], ["limitCPU", "CPU limit", "500m"], ["requestMemory", "内存 request", "128Mi"], ["limitMemory", "内存 limit", "512Mi"]].map(([key, label, placeholder]) => (
            <div className="field" key={key}><label htmlFor={`config-${key}`}>{label}</label><input id={`config-${key}`} value={draft[key as keyof AppConfigDraft] as string} placeholder={placeholder} onChange={(event) => update({ [key]: event.target.value })} aria-invalid={Boolean(errors[key])} aria-describedby={errors[key] ? `config-${key}-error` : undefined} /><ErrorText id={`config-${key}-error`} message={errors[key]} /></div>
          ))}
        </div>
      </fieldset>

      <fieldset className="config-group">
        <legend>网络</legend>
        <div className="collection-heading"><p>容器与 Service 端口</p><button className="button button-secondary button-small" type="button" onClick={addPort}><Plus size={15} aria-hidden="true" />添加</button></div>
        <p className="field-help">Service 端口留空时仅声明容器端口。使用新端口后请清空上方兼容端口。</p>
        <ErrorText id="config-ports-error" message={errors.ports} />
        {draft.ports.map((item, index) => (
          <div className="repeat-card" key={item.id}>
            <div className="repeat-heading"><strong>端口 {index + 1}</strong><button className="icon-button icon-button-danger compact-button" type="button" aria-label={`删除端口 ${index + 1}`} onClick={() => update({ ports: draft.ports.filter((current) => current.id !== item.id) })}><Trash2 size={16} aria-hidden="true" /></button></div>
            <div className="compact-grid">
              <div className="field"><label htmlFor={`port-${item.id}-name`}>名称</label><input id={`port-${item.id}-name`} value={item.name} placeholder="http" onChange={(event) => update({ ports: replace(draft.ports, index, { ...item, name: event.target.value }) })} aria-invalid={Boolean(errors[`ports.${index}`])} aria-describedby={errors[`ports.${index}`] ? `port-${item.id}-error` : undefined} /></div>
              <div className="field"><label htmlFor={`port-${item.id}-container`}>容器端口</label><input id={`port-${item.id}-container`} type="number" min="1" max="65535" value={item.containerPort} onChange={(event) => update({ ports: replace(draft.ports, index, { ...item, containerPort: event.target.value }) })} /></div>
              <div className="field"><label htmlFor={`port-${item.id}-protocol`}>协议</label><select id={`port-${item.id}-protocol`} value={item.protocol} onChange={(event) => update({ ports: replace(draft.ports, index, { ...item, protocol: event.target.value as PortDraft["protocol"] }) })}><option value="TCP">TCP</option><option value="UDP">UDP</option></select></div>
              <div className="field"><label htmlFor={`port-${item.id}-service`}>Service 端口 <span className="optional">选填</span></label><input id={`port-${item.id}-service`} type="number" min="1" max="65535" value={item.servicePort} onChange={(event) => update({ ports: replace(draft.ports, index, { ...item, servicePort: event.target.value }) })} /></div>
            </div>
            <ErrorText id={`port-${item.id}-error`} message={errors[`ports.${index}`]} />
          </div>
        ))}
      </fieldset>

      <fieldset className="config-group">
        <legend>健康检查</legend>
        <ProbeFields id="config-startup-probe" label="启动探针" value={draft.startupProbe} error={errors.startupProbe} onChange={(startupProbe) => update({ startupProbe })} />
        <ProbeFields id="config-readiness-probe" label="就绪探针" value={draft.readinessProbe} error={errors.readinessProbe} onChange={(readinessProbe) => update({ readinessProbe })} />
        <ProbeFields id="config-liveness-probe" label="存活探针" value={draft.livenessProbe} error={errors.livenessProbe} onChange={(livenessProbe) => update({ livenessProbe })} />
      </fieldset>

      <fieldset className="config-group">
        <legend>存储</legend>
        <div className="collection-heading"><p>卷</p><button className="button button-secondary button-small" type="button" onClick={addVolume}><Plus size={15} aria-hidden="true" />添加</button></div>
        <ErrorText id="config-volumes-error" message={errors.volumes} />
        {draft.volumes.map((item, index) => (
          <div className="repeat-card" key={item.id}>
            <div className="repeat-heading"><strong>卷 {index + 1}</strong><button className="icon-button icon-button-danger compact-button" type="button" aria-label={`删除卷 ${index + 1}`} onClick={() => update({ volumes: draft.volumes.filter((current) => current.id !== item.id) })}><Trash2 size={16} aria-hidden="true" /></button></div>
            <div className="compact-grid">
              <div className="field"><label htmlFor={`volume-${item.id}-name`}>卷名称</label><input id={`volume-${item.id}-name`} value={item.name} onChange={(event) => update({ volumes: replace(draft.volumes, index, { ...item, name: event.target.value }) })} aria-invalid={Boolean(errors[`volumes.${index}`])} aria-describedby={errors[`volumes.${index}`] ? `volume-${item.id}-error` : undefined} /></div>
              <div className="field"><label htmlFor={`volume-${item.id}-type`}>类型</label><select id={`volume-${item.id}-type`} value={item.type} onChange={(event) => update({ volumes: replace(draft.volumes, index, { ...item, type: event.target.value as VolumeDraft["type"] }) })}><option value="empty_dir">emptyDir</option><option value="persistent_volume_claim">已有 PVC</option><option value="config_map">已有 ConfigMap</option><option value="secret">已有 Secret</option></select></div>
              {item.type === "empty_dir" ? <><div className="field"><label htmlFor={`volume-${item.id}-medium`}>介质</label><select id={`volume-${item.id}-medium`} value={item.medium} onChange={(event) => update({ volumes: replace(draft.volumes, index, { ...item, medium: event.target.value as VolumeDraft["medium"] }) })}><option value="">节点磁盘</option><option value="Memory">内存</option></select></div><div className="field"><label htmlFor={`volume-${item.id}-size`}>容量限制 <span className="optional">选填</span></label><input id={`volume-${item.id}-size`} value={item.sizeLimit} placeholder="1Gi" onChange={(event) => update({ volumes: replace(draft.volumes, index, { ...item, sizeLimit: event.target.value }) })} /></div></> : <div className="field field-wide"><label htmlFor={`volume-${item.id}-ref`}>资源名称</label><input id={`volume-${item.id}-ref`} value={item.refName} onChange={(event) => update({ volumes: replace(draft.volumes, index, { ...item, refName: event.target.value }) })} /></div>}
            </div>
            {item.type === "persistent_volume_claim" ? <label className="check-field"><input type="checkbox" checked={item.readOnly} onChange={(event) => update({ volumes: replace(draft.volumes, index, { ...item, readOnly: event.target.checked }) })} />只读 PVC</label> : null}
            <ErrorText id={`volume-${item.id}-error`} message={errors[`volumes.${index}`]} />
          </div>
        ))}
        <div className="collection-heading"><p>卷挂载</p><button className="button button-secondary button-small" type="button" onClick={addMount}><Plus size={15} aria-hidden="true" />添加</button></div>
        {draft.volumeMounts.map((item, index) => (
          <div className="repeat-card" key={item.id}>
            <div className="repeat-heading"><strong>挂载 {index + 1}</strong><button className="icon-button icon-button-danger compact-button" type="button" aria-label={`删除卷挂载 ${index + 1}`} onClick={() => update({ volumeMounts: draft.volumeMounts.filter((current) => current.id !== item.id) })}><Trash2 size={16} aria-hidden="true" /></button></div>
            <div className="compact-grid">
              <div className="field"><label htmlFor={`mount-${item.id}-name`}>卷名称</label><input id={`mount-${item.id}-name`} value={item.name} onChange={(event) => update({ volumeMounts: replace(draft.volumeMounts, index, { ...item, name: event.target.value }) })} aria-invalid={Boolean(errors[`volumeMounts.${index}`])} aria-describedby={errors[`volumeMounts.${index}`] ? `mount-${item.id}-error` : undefined} /></div>
              <div className="field"><label htmlFor={`mount-${item.id}-path`}>挂载路径</label><input id={`mount-${item.id}-path`} value={item.mountPath} placeholder="/data" onChange={(event) => update({ volumeMounts: replace(draft.volumeMounts, index, { ...item, mountPath: event.target.value }) })} /></div>
              <div className="field field-wide"><label htmlFor={`mount-${item.id}-subpath`}>子路径 <span className="optional">选填</span></label><input id={`mount-${item.id}-subpath`} value={item.subPath} onChange={(event) => update({ volumeMounts: replace(draft.volumeMounts, index, { ...item, subPath: event.target.value }) })} /></div>
            </div>
            <label className="check-field"><input type="checkbox" checked={item.readOnly} onChange={(event) => update({ volumeMounts: replace(draft.volumeMounts, index, { ...item, readOnly: event.target.checked }) })} />只读挂载</label>
            <ErrorText id={`mount-${item.id}-error`} message={errors[`volumeMounts.${index}`]} />
          </div>
        ))}
      </fieldset>

      <fieldset className="config-group">
        <legend>安全与生命周期</legend>
        <div className="check-grid">
          <label className="check-field"><input type="checkbox" checked={draft.runAsNonRoot} onChange={(event) => update({ runAsNonRoot: event.target.checked })} />要求非 root 运行</label>
          <label className="check-field"><input type="checkbox" checked={draft.readOnlyRootFilesystem} onChange={(event) => update({ readOnlyRootFilesystem: event.target.checked })} />只读根文件系统</label>
          <label className="check-field"><input type="checkbox" checked={draft.allowPrivilegeEscalation} onChange={(event) => update({ allowPrivilegeEscalation: event.target.checked })} />禁止权限提升</label>
          <label className="check-field"><input type="checkbox" checked={draft.seccompRuntimeDefault} onChange={(event) => update({ seccompRuntimeDefault: event.target.checked })} />使用 RuntimeDefault seccomp</label>
        </div>
        <div className="compact-grid">
          {[["runAsUser", "运行用户 ID"], ["runAsGroup", "运行组 ID"], ["fsGroup", "文件系统组 ID"]].map(([key, label]) => <div className="field" key={key}><label htmlFor={`config-${key}`}>{label}</label><input id={`config-${key}`} type="number" min="0" max="4294967295" value={draft[key as keyof AppConfigDraft] as string} onChange={(event) => update({ [key]: event.target.value })} aria-invalid={Boolean(errors[key])} aria-describedby={errors[key] ? `config-${key}-error` : undefined} /><ErrorText id={`config-${key}-error`} message={errors[key]} /></div>)}
          <div className="field"><label htmlFor="config-termination">终止宽限期（秒）</label><input id="config-termination" type="number" min="0" max="300" value={draft.terminationGracePeriodSeconds} onChange={(event) => update({ terminationGracePeriodSeconds: event.target.value })} aria-invalid={Boolean(errors.terminationGracePeriodSeconds)} aria-describedby={errors.terminationGracePeriodSeconds ? "config-termination-error" : undefined} /><ErrorText id="config-termination-error" message={errors.terminationGracePeriodSeconds} /></div>
        </div>
        <div className="field"><label htmlFor="config-drop-capabilities">移除 Linux capabilities <span className="optional">逗号或换行分隔</span></label><textarea id="config-drop-capabilities" value={draft.dropCapabilities} onChange={(event) => update({ dropCapabilities: event.target.value })} rows={2} aria-invalid={Boolean(errors.dropCapabilities)} aria-describedby={errors.dropCapabilities ? "config-drop-capabilities-error" : undefined} /><ErrorText id="config-drop-capabilities-error" message={errors.dropCapabilities} /></div>
        <div className="field"><label htmlFor="config-image-pull-secrets">imagePullSecret <span className="optional">逗号或换行分隔</span></label><textarea id="config-image-pull-secrets" value={draft.imagePullSecrets} onChange={(event) => update({ imagePullSecrets: event.target.value })} rows={2} aria-invalid={Boolean(errors.imagePullSecrets)} aria-describedby={errors.imagePullSecrets ? "config-image-pull-secrets-error" : undefined} /><ErrorText id="config-image-pull-secrets-error" message={errors.imagePullSecrets} /></div>
      </fieldset>
    </details>
  );
}
