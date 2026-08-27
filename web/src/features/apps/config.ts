import type { AppConfig, AppPort, EnvFromSource, EnvVar, Probe, ResourceRequirements, SecurityContext, Volume, VolumeMount } from "./types";

type RecordValue = Record<string, unknown>;

export interface AppConfigDraft {
  command: string;
  args: string;
  workingDir: string;
  imagePullPolicy: "" | "Always" | "IfNotPresent" | "Never";
  env: EnvDraft[];
  envFrom: EnvFromDraft[];
  requestCPU: string;
  requestMemory: string;
  limitCPU: string;
  limitMemory: string;
  ports: PortDraft[];
  startupProbe: ProbeDraft;
  readinessProbe: ProbeDraft;
  livenessProbe: ProbeDraft;
  volumes: VolumeDraft[];
  volumeMounts: VolumeMountDraft[];
  runAsNonRoot: boolean;
  runAsUser: string;
  runAsGroup: string;
  fsGroup: string;
  readOnlyRootFilesystem: boolean;
  allowPrivilegeEscalation: boolean;
  dropCapabilities: string;
  seccompRuntimeDefault: boolean;
  terminationGracePeriodSeconds: string;
  imagePullSecrets: string;
}

export interface EnvDraft { id: number; name: string; source: "value" | "config_map" | "secret"; value: string; refName: string; refKey: string }
export interface EnvFromDraft { id: number; kind: "config_map" | "secret"; name: string; prefix: string }
export interface PortDraft { id: number; name: string; containerPort: string; protocol: "TCP" | "UDP"; servicePort: string }
export interface ProbeDraft { type: "" | "http" | "tcp" | "exec"; path: string; port: string; scheme: "HTTP" | "HTTPS"; headers: string; command: string; initialDelay: string; period: string; timeout: string; success: string; failure: string }
export interface VolumeDraft { id: number; name: string; type: "empty_dir" | "persistent_volume_claim" | "config_map" | "secret"; refName: string; readOnly: boolean; medium: "" | "Memory"; sizeLimit: string }
export interface VolumeMountDraft { id: number; name: string; mountPath: string; subPath: string; readOnly: boolean }

export interface ConfigBuildResult { config?: AppConfig; errors: Record<string, string> }

const DNS_LABEL = /^[a-z0-9](?:[-a-z0-9]*[a-z0-9])?$/;
const ENV_NAME = /^[A-Za-z_][A-Za-z0-9_]*$/;
const QUANTITY = /^([0-9]+(?:\.[0-9]+)?)([EPTGMK]i?|m|u|n)?$/;

export function emptyAppConfigDraft(): AppConfigDraft {
  return {
    command: "", args: "", workingDir: "", imagePullPolicy: "", env: [], envFrom: [],
    requestCPU: "", requestMemory: "", limitCPU: "", limitMemory: "", ports: [],
    startupProbe: emptyProbe(), readinessProbe: emptyProbe(), livenessProbe: emptyProbe(),
    volumes: [], volumeMounts: [], runAsNonRoot: false, runAsUser: "", runAsGroup: "", fsGroup: "",
    readOnlyRootFilesystem: false, allowPrivilegeEscalation: false, dropCapabilities: "",
    seccompRuntimeDefault: false, terminationGracePeriodSeconds: "", imagePullSecrets: "",
  };
}

export function buildAppConfig(draft: AppConfigDraft): ConfigBuildResult {
  const errors: Record<string, string> = {};
  const config: AppConfig = {};
  const command = lines(draft.command, false);
  const args = lines(draft.args, false);
  if (command.length > 20) errors.command = "命令最多 20 项。";
  if (args.length > 100) errors.args = "参数最多 100 项。";
  if (command.length) config.command = command;
  if (args.length) config.args = args;
  if (draft.workingDir) {
    if (!validAbsolutePath(draft.workingDir)) errors.workingDir = "请输入绝对 POSIX 路径。";
    else config.working_dir = draft.workingDir;
  }
  if (draft.imagePullPolicy) config.image_pull_policy = draft.imagePullPolicy;

  if (draft.env.length + draft.envFrom.length > 100) errors.env = "环境变量与引用合计最多 100 项。";
  const env = draft.env.map((item, index) => buildEnv(item, index, errors)).filter((item): item is EnvVar => item !== null);
  const envFrom = draft.envFrom.map((item, index) => buildEnvFrom(item, index, errors)).filter((item): item is EnvFromSource => item !== null);
  if (env.length) config.env = env;
  if (envFrom.length) config.env_from = envFrom;

  const resources = buildResources(draft, errors);
  if (resources) config.resources = resources;
  if (draft.ports.length > 20) errors.ports = "端口最多 20 项。";
  const ports = draft.ports.map((item, index) => buildPort(item, index, errors)).filter((item): item is AppPort => item !== null);
  if (ports.length) config.ports = ports;

  for (const [key, value] of [["startupProbe", draft.startupProbe], ["readinessProbe", draft.readinessProbe], ["livenessProbe", draft.livenessProbe]] as const) {
    const probe = buildProbe(value, key, errors);
    if (probe) config[key === "startupProbe" ? "startup_probe" : key === "readinessProbe" ? "readiness_probe" : "liveness_probe"] = probe;
  }

  if (draft.volumes.length > 20 || draft.volumeMounts.length > 20) errors.volumes = "卷与挂载分别最多 20 项。";
  const volumes = draft.volumes.map((item, index) => buildVolume(item, index, errors)).filter((item): item is Volume => item !== null);
  const volumeNames = new Set(volumes.map((volume) => volume.name));
  const volumeMounts = draft.volumeMounts.map((item, index) => buildVolumeMount(item, index, volumeNames, errors)).filter((item): item is VolumeMount => item !== null);
  if (volumes.length) config.volumes = volumes;
  if (volumeMounts.length) config.volume_mounts = volumeMounts;

  const security = buildSecurity(draft, errors);
  if (security) config.security_context = security;
  if (draft.terminationGracePeriodSeconds) {
    const seconds = boundedInteger(draft.terminationGracePeriodSeconds, 0, 300);
    if (seconds === null) errors.terminationGracePeriodSeconds = "终止宽限期必须是 0-300 的整数。";
    else config.termination_grace_period_seconds = seconds;
  }
  const secrets = words(draft.imagePullSecrets);
  if (secrets.length > 10) errors.imagePullSecrets = "imagePullSecret 最多 10 项。";
  else if (secrets.some((name) => !validResourceName(name))) errors.imagePullSecrets = "imagePullSecret 名称无效。";
  else if (secrets.length) config.image_pull_secrets = secrets;

  return { config: Object.keys(config).length ? config : undefined, errors };
}

export function parseAppConfig(value: unknown): AppConfig {
  if (value == null) return {};
  if (!isRecord(value)) throw new Error("config 不是对象");
  const config: AppConfig = {};
  if (value.command !== undefined) config.command = stringArray(value.command);
  if (value.args !== undefined) config.args = stringArray(value.args);
  if (value.working_dir !== undefined) config.working_dir = stringValue(value.working_dir);
  if (value.image_pull_policy !== undefined) config.image_pull_policy = enumValue(value.image_pull_policy, ["Always", "IfNotPresent", "Never"]);
  if (value.env !== undefined) config.env = arrayValue(value.env).map(parseEnv);
  if (value.env_from !== undefined) config.env_from = arrayValue(value.env_from).map(parseEnvFrom);
  if (value.resources !== undefined) config.resources = parseResources(value.resources);
  if (value.ports !== undefined) config.ports = arrayValue(value.ports).map(parsePort);
  if (value.startup_probe !== undefined) config.startup_probe = parseProbe(value.startup_probe);
  if (value.readiness_probe !== undefined) config.readiness_probe = parseProbe(value.readiness_probe);
  if (value.liveness_probe !== undefined) config.liveness_probe = parseProbe(value.liveness_probe);
  if (value.volumes !== undefined) config.volumes = arrayValue(value.volumes).map(parseVolume);
  if (value.volume_mounts !== undefined) config.volume_mounts = arrayValue(value.volume_mounts).map(parseVolumeMount);
  if (value.security_context !== undefined) config.security_context = parseSecurity(value.security_context);
  if (value.termination_grace_period_seconds !== undefined) config.termination_grace_period_seconds = numberValue(value.termination_grace_period_seconds);
  if (value.image_pull_secrets !== undefined) config.image_pull_secrets = stringArray(value.image_pull_secrets);
  return config;
}

export function hasAppConfig(config: AppConfig | undefined): boolean {
  return Boolean(config && Object.keys(config).length);
}

function emptyProbe(): ProbeDraft { return { type: "", path: "/", port: "", scheme: "HTTP", headers: "", command: "", initialDelay: "", period: "", timeout: "", success: "", failure: "" }; }
function lines(value: string, trim = true): string[] { return value === "" ? [] : value.split("\n").map((item) => trim ? item.trim() : item).filter((item) => !trim || item !== ""); }
function words(value: string): string[] { return value.split(/[\s,]+/).map((item) => item.trim()).filter(Boolean); }
function validAbsolutePath(value: string): boolean { return value.startsWith("/") && value.length <= 4096; }
function validResourceName(value: string): boolean { return value.length > 0 && value.length <= 253 && value.split(".").every((part) => DNS_LABEL.test(part) && part.length <= 63); }
function boundedInteger(value: string, minimum: number, maximum: number): number | null { const number = Number(value); return Number.isSafeInteger(number) && number >= minimum && number <= maximum ? number : null; }

function buildEnv(item: EnvDraft, index: number, errors: Record<string, string>): EnvVar | null {
  const key = `env.${index}`;
  if (!ENV_NAME.test(item.name)) { errors[key] = "环境变量名称无效。"; return null; }
  if (item.source === "value") return { name: item.name, value: item.value };
  if (!validResourceName(item.refName) || !item.refKey) { errors[key] = "请填写有效的资源名称和键。"; return null; }
  return { name: item.name, value_from: item.source === "secret" ? { secret_key_ref: { name: item.refName, key: item.refKey } } : { config_map_key_ref: { name: item.refName, key: item.refKey } } };
}

function buildEnvFrom(item: EnvFromDraft, index: number, errors: Record<string, string>): EnvFromSource | null {
  const key = `envFrom.${index}`;
  if (!validResourceName(item.name) || (item.prefix && !ENV_NAME.test(item.prefix))) { errors[key] = "引用名称或前缀无效。"; return null; }
  return item.kind === "secret" ? { prefix: item.prefix || undefined, secret_ref: { name: item.name } } : { prefix: item.prefix || undefined, config_map_ref: { name: item.name } };
}

function buildResources(draft: AppConfigDraft, errors: Record<string, string>): ResourceRequirements | undefined {
  for (const [key, value] of [["requestCPU", draft.requestCPU], ["requestMemory", draft.requestMemory], ["limitCPU", draft.limitCPU], ["limitMemory", draft.limitMemory]] as const) {
    if (value && !QUANTITY.test(value)) errors[key] = "请输入正数 Kubernetes Quantity。";
  }
  if (Object.keys(errors).some((key) => ["requestCPU", "requestMemory", "limitCPU", "limitMemory"].includes(key))) return undefined;
  const requests = { cpu: draft.requestCPU || undefined, memory: draft.requestMemory || undefined };
  const limits = { cpu: draft.limitCPU || undefined, memory: draft.limitMemory || undefined };
  return draft.requestCPU || draft.requestMemory || draft.limitCPU || draft.limitMemory ? { requests, limits } : undefined;
}

function buildPort(item: PortDraft, index: number, errors: Record<string, string>): AppPort | null {
  const containerPort = boundedInteger(item.containerPort, 1, 65535);
  const servicePort = item.servicePort ? boundedInteger(item.servicePort, 1, 65535) : undefined;
  if (!item.name || item.name.length > 15 || !DNS_LABEL.test(item.name) || containerPort === null || servicePort === null) { errors[`ports.${index}`] = "请填写有效的端口名称和 1-65535 端口。"; return null; }
  return { name: item.name, container_port: containerPort, protocol: item.protocol, service_port: servicePort };
}

function buildProbe(item: ProbeDraft, key: string, errors: Record<string, string>): Probe | undefined {
  if (!item.type) return undefined;
  const probe: Probe = {};
  if (item.type === "exec") { const command = lines(item.command, false); if (!command.length || command.length > 20) errors[key] = "Exec 命令必须包含 1-20 行。"; else probe.exec = { command }; }
  else { const port = boundedInteger(item.port, 1, 65535); if (port === null) errors[key] = "探针端口必须是 1-65535 的整数。"; else if (item.type === "tcp") probe.tcp_socket = { port }; else if (!validAbsolutePath(item.path)) errors[key] = "HTTP 探针路径必须是绝对路径。"; else { const headers = buildHeaders(item.headers, key, errors); probe.http_get = { path: item.path, port, scheme: item.scheme, http_headers: headers.length ? headers : undefined }; } }
  for (const [draftKey, apiKey, minimum, maximum] of [["initialDelay", "initial_delay_seconds", 0, 3600], ["period", "period_seconds", 0, 3600], ["timeout", "timeout_seconds", 0, 3600], ["success", "success_threshold", 0, 10], ["failure", "failure_threshold", 0, 60]] as const) {
    if (!item[draftKey]) continue;
    const value = boundedInteger(item[draftKey], minimum, maximum);
    if (value === null) errors[key] = "探针时间或阈值超出范围。"; else probe[apiKey] = value;
  }
  if ((key === "startupProbe" || key === "livenessProbe") && probe.success_threshold !== undefined && probe.success_threshold > 1) errors[key] = "启动与存活探针的成功阈值只能是 0 或 1。";
  return probe;
}

function buildHeaders(value: string, key: string, errors: Record<string, string>) {
  const headers = lines(value).map((line) => { const separator = line.indexOf(":"); return separator < 1 ? null : { name: line.slice(0, separator).trim(), value: line.slice(separator + 1).trim() }; });
  if (headers.length > 20 || headers.some((header) => header === null)) errors[key] = "HTTP Header 每行使用“名称: 值”，最多 20 行。";
  return headers.filter((header): header is { name: string; value: string } => header !== null);
}

function buildVolume(item: VolumeDraft, index: number, errors: Record<string, string>): Volume | null {
  if (!DNS_LABEL.test(item.name)) { errors[`volumes.${index}`] = "卷名称无效。"; return null; }
  if (item.type === "empty_dir") return { name: item.name, empty_dir: { medium: item.medium || undefined, size_limit: item.sizeLimit || undefined } };
  if (!validResourceName(item.refName)) { errors[`volumes.${index}`] = "引用资源名称无效。"; return null; }
  if (item.type === "persistent_volume_claim") return { name: item.name, persistent_volume_claim: { claim_name: item.refName, read_only: item.readOnly || undefined } };
  return item.type === "config_map" ? { name: item.name, config_map: { name: item.refName } } : { name: item.name, secret: { name: item.refName } };
}

function buildVolumeMount(item: VolumeMountDraft, index: number, volumeNames: Set<string>, errors: Record<string, string>): VolumeMount | null {
  if (!volumeNames.has(item.name) || !validAbsolutePath(item.mountPath) || item.subPath.startsWith("/") || item.subPath.split("/").includes("..")) { errors[`volumeMounts.${index}`] = "挂载必须引用已有卷，并使用有效路径。"; return null; }
  return { name: item.name, mount_path: item.mountPath, sub_path: item.subPath || undefined, read_only: item.readOnly || undefined };
}

function buildSecurity(draft: AppConfigDraft, errors: Record<string, string>): SecurityContext | undefined {
  const result: SecurityContext = {};
  for (const [key, value, output] of [["runAsUser", draft.runAsUser, "run_as_user"], ["runAsGroup", draft.runAsGroup, "run_as_group"], ["fsGroup", draft.fsGroup, "fs_group"]] as const) {
    if (!value) continue;
    const id = boundedInteger(value, 0, 4294967295);
    if (id === null) errors[key] = "ID 必须是 0-4294967295 的整数。"; else result[output] = id;
  }
  if (draft.runAsNonRoot) result.run_as_non_root = true;
  if (draft.readOnlyRootFilesystem) result.read_only_root_filesystem = true;
  if (draft.allowPrivilegeEscalation) result.allow_privilege_escalation = false;
  const capabilities = words(draft.dropCapabilities).map((value) => value.toUpperCase());
  if (capabilities.length > 20) errors.dropCapabilities = "capability 最多 20 项。"; else if (capabilities.length) result.drop_capabilities = capabilities;
  if (draft.seccompRuntimeDefault) result.seccomp_profile = "RuntimeDefault";
  return Object.keys(result).length ? result : undefined;
}

function isRecord(value: unknown): value is RecordValue { return typeof value === "object" && value !== null && !Array.isArray(value); }
function recordValue(value: unknown): RecordValue { if (!isRecord(value)) throw new Error("字段不是对象"); return value; }
function arrayValue(value: unknown): unknown[] { if (!Array.isArray(value)) throw new Error("字段不是数组"); return value; }
function stringValue(value: unknown): string { if (typeof value !== "string") throw new Error("字段不是字符串"); return value; }
function numberValue(value: unknown): number { if (typeof value !== "number") throw new Error("字段不是数字"); return value; }
function booleanValue(value: unknown): boolean { if (typeof value !== "boolean") throw new Error("字段不是布尔值"); return value; }
function stringArray(value: unknown): string[] { return arrayValue(value).map(stringValue); }
function enumValue<const T extends readonly string[]>(value: unknown, values: T): T[number] { const text = stringValue(value); if (!values.includes(text as T[number])) throw new Error("枚举值无效"); return text as T[number]; }
function optionalString(record: RecordValue, key: string): string | undefined { return record[key] === undefined ? undefined : stringValue(record[key]); }
function optionalNumber(record: RecordValue, key: string): number | undefined { return record[key] === undefined ? undefined : numberValue(record[key]); }
function optionalBoolean(record: RecordValue, key: string): boolean | undefined { return record[key] === undefined ? undefined : booleanValue(record[key]); }

function parseEnv(value: unknown): EnvVar { const item = recordValue(value); const result: EnvVar = { name: stringValue(item.name) }; if (item.value !== undefined) result.value = stringValue(item.value); if (item.value_from !== undefined) { const source = recordValue(item.value_from); result.value_from = {}; if (source.config_map_key_ref !== undefined) result.value_from.config_map_key_ref = parseKeyReference(source.config_map_key_ref); if (source.secret_key_ref !== undefined) result.value_from.secret_key_ref = parseKeyReference(source.secret_key_ref); } return result; }
function parseKeyReference(value: unknown) { const item = recordValue(value); return { name: stringValue(item.name), key: stringValue(item.key) }; }
function parseEnvFrom(value: unknown): EnvFromSource { const item = recordValue(value); const result: EnvFromSource = { prefix: optionalString(item, "prefix") }; if (item.config_map_ref !== undefined) result.config_map_ref = { name: stringValue(recordValue(item.config_map_ref).name) }; if (item.secret_ref !== undefined) result.secret_ref = { name: stringValue(recordValue(item.secret_ref).name) }; return result; }
function parseResources(value: unknown): ResourceRequirements { const item = recordValue(value); const result: ResourceRequirements = {}; if (item.requests !== undefined) result.requests = parseResourceValues(item.requests); if (item.limits !== undefined) result.limits = parseResourceValues(item.limits); return result; }
function parseResourceValues(value: unknown) { const item = recordValue(value); return { cpu: optionalString(item, "cpu"), memory: optionalString(item, "memory") }; }
function parsePort(value: unknown): AppPort { const item = recordValue(value); return { name: stringValue(item.name), container_port: numberValue(item.container_port), protocol: item.protocol === undefined ? undefined : enumValue(item.protocol, ["TCP", "UDP"]), service_port: optionalNumber(item, "service_port") }; }
function parseProbe(value: unknown): Probe { const item = recordValue(value); const result: Probe = { initial_delay_seconds: optionalNumber(item, "initial_delay_seconds"), period_seconds: optionalNumber(item, "period_seconds"), timeout_seconds: optionalNumber(item, "timeout_seconds"), success_threshold: optionalNumber(item, "success_threshold"), failure_threshold: optionalNumber(item, "failure_threshold") }; if (item.http_get !== undefined) { const action = recordValue(item.http_get); result.http_get = { path: stringValue(action.path), port: numberValue(action.port), scheme: action.scheme === undefined ? undefined : enumValue(action.scheme, ["HTTP", "HTTPS"]), http_headers: action.http_headers === undefined ? undefined : arrayValue(action.http_headers).map((header) => { const record = recordValue(header); return { name: stringValue(record.name), value: stringValue(record.value) }; }) }; } if (item.tcp_socket !== undefined) result.tcp_socket = { port: numberValue(recordValue(item.tcp_socket).port) }; if (item.exec !== undefined) result.exec = { command: stringArray(recordValue(item.exec).command) }; return result; }
function parseVolume(value: unknown): Volume { const item = recordValue(value); const result: Volume = { name: stringValue(item.name) }; if (item.empty_dir !== undefined) { const source = recordValue(item.empty_dir); result.empty_dir = { medium: source.medium === undefined ? undefined : enumValue(source.medium, ["Memory"]), size_limit: optionalString(source, "size_limit") }; } if (item.persistent_volume_claim !== undefined) { const source = recordValue(item.persistent_volume_claim); result.persistent_volume_claim = { claim_name: stringValue(source.claim_name), read_only: optionalBoolean(source, "read_only") }; } if (item.config_map !== undefined) result.config_map = { name: stringValue(recordValue(item.config_map).name) }; if (item.secret !== undefined) result.secret = { name: stringValue(recordValue(item.secret).name) }; return result; }
function parseVolumeMount(value: unknown): VolumeMount { const item = recordValue(value); return { name: stringValue(item.name), mount_path: stringValue(item.mount_path), sub_path: optionalString(item, "sub_path"), read_only: optionalBoolean(item, "read_only") }; }
function parseSecurity(value: unknown): SecurityContext { const item = recordValue(value); const allowPrivilegeEscalation = item.allow_privilege_escalation === undefined ? undefined : booleanValue(item.allow_privilege_escalation); if (allowPrivilegeEscalation) throw new Error("权限提升配置无效"); return { run_as_non_root: optionalBoolean(item, "run_as_non_root"), run_as_user: optionalNumber(item, "run_as_user"), run_as_group: optionalNumber(item, "run_as_group"), fs_group: optionalNumber(item, "fs_group"), read_only_root_filesystem: optionalBoolean(item, "read_only_root_filesystem"), allow_privilege_escalation: allowPrivilegeEscalation, drop_capabilities: item.drop_capabilities === undefined ? undefined : stringArray(item.drop_capabilities), seccomp_profile: item.seccomp_profile === undefined ? undefined : enumValue(item.seccomp_profile, ["RuntimeDefault"]) }; }
