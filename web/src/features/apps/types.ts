export interface App {
  id: number;
  name: string;
  image: string;
  replicas: number;
  status: string;
  project_id: number;
  created_at: string;
  updated_at: string;
  config?: AppConfig;
}

export interface CreateAppInput {
  name: string;
  image: string;
  replicas: number;
  port?: number;
  config?: AppConfig;
}

export interface AppConfig {
  command?: string[];
  args?: string[];
  working_dir?: string;
  image_pull_policy?: "Always" | "IfNotPresent" | "Never";
  env?: EnvVar[];
  env_from?: EnvFromSource[];
  resources?: ResourceRequirements;
  ports?: AppPort[];
  startup_probe?: Probe;
  readiness_probe?: Probe;
  liveness_probe?: Probe;
  volumes?: Volume[];
  volume_mounts?: VolumeMount[];
  security_context?: SecurityContext;
  termination_grace_period_seconds?: number;
  image_pull_secrets?: string[];
}

export interface EnvVar {
  name: string;
  value?: string;
  value_from?: { config_map_key_ref?: KeyReference; secret_key_ref?: KeyReference };
}

export interface KeyReference { name: string; key: string }

export interface EnvFromSource {
  prefix?: string;
  config_map_ref?: { name: string };
  secret_ref?: { name: string };
}

export interface ResourceRequirements {
  requests?: ResourceValues;
  limits?: ResourceValues;
}

export interface ResourceValues { cpu?: string; memory?: string }

export interface AppPort {
  name: string;
  container_port: number;
  protocol?: "TCP" | "UDP";
  service_port?: number;
}

export interface Probe {
  http_get?: { path: string; port: number; scheme?: "HTTP" | "HTTPS"; http_headers?: HTTPHeader[] };
  tcp_socket?: { port: number };
  exec?: { command: string[] };
  initial_delay_seconds?: number;
  period_seconds?: number;
  timeout_seconds?: number;
  success_threshold?: number;
  failure_threshold?: number;
}

export interface HTTPHeader { name: string; value: string }

export interface Volume {
  name: string;
  empty_dir?: { medium?: "Memory"; size_limit?: string };
  persistent_volume_claim?: { claim_name: string; read_only?: boolean };
  config_map?: { name: string };
  secret?: { name: string };
}

export interface VolumeMount { name: string; mount_path: string; sub_path?: string; read_only?: boolean }

export interface SecurityContext {
  run_as_non_root?: boolean;
  run_as_user?: number;
  run_as_group?: number;
  fs_group?: number;
  read_only_root_filesystem?: boolean;
  allow_privilege_escalation?: false;
  drop_capabilities?: string[];
  seccomp_profile?: "RuntimeDefault";
}

export type LifecycleAction = "start" | "stop" | "restart";
