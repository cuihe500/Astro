import { ApiError, request } from "../../lib/api";
import { projectAppsPath } from "../../lib/routes";
import type { App, CreateAppInput, LifecycleAction } from "./types";

export function parseApp(value: unknown): App {
  if (typeof value !== "object" || value === null) {
    throw new ApiError(-1, "服务器返回的应用数据无效，请重试。");
  }
  const app = value as Record<string, unknown>;
  if (!(
    typeof app.id === "number" &&
    typeof app.name === "string" &&
    typeof app.image === "string" &&
    typeof app.replicas === "number" &&
    typeof app.status === "string" &&
    typeof app.project_id === "number" &&
    typeof app.created_at === "string" &&
    typeof app.updated_at === "string"
  )) {
    throw new ApiError(-1, "服务器返回的应用数据无效，请重试。");
  }
  return value as unknown as App;
}

export function parseApps(data: unknown): App[] {
  if (data == null) return [];
  if (!Array.isArray(data)) {
    throw new ApiError(-1, "服务器返回的应用列表无效，请重试。");
  }
  return data.map(parseApp);
}

export async function getApps(projectId: string | number): Promise<App[]> {
  return parseApps(await request<unknown>(projectAppsPath(projectId), { auth: true }));
}

export async function getApp(projectId: string | number, id: string | number): Promise<App> {
  return parseApp(await request<unknown>(`${projectAppsPath(projectId)}/${encodeURIComponent(String(id))}`, { auth: true }));
}

export async function createApp(projectId: string | number, input: CreateAppInput): Promise<App> {
  return parseApp(
    await request<unknown>(projectAppsPath(projectId), {
      method: "POST",
      auth: true,
      body: JSON.stringify(input),
    }),
  );
}

export function deleteApp(projectId: string | number, id: number): Promise<void> {
  return request<void>(`${projectAppsPath(projectId)}/${id}`, { method: "DELETE", auth: true });
}

export function runLifecycleAction(projectId: string | number, id: number, action: LifecycleAction): Promise<void> {
  return request<void>(`${projectAppsPath(projectId)}/${id}/${action}`, { method: "POST", auth: true });
}

export async function getAppLogs(projectId: string | number, id: string | number): Promise<string> {
  const data = await request<unknown>(
    `${projectAppsPath(projectId)}/${encodeURIComponent(String(id))}/logs?lines=100`,
    { auth: true },
  );
  if (typeof data !== "object" || data === null || typeof (data as Record<string, unknown>).logs !== "string") {
    throw new ApiError(-1, "服务器返回的日志数据无效，请重试。");
  }
  return (data as { logs: string }).logs;
}
