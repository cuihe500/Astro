import { ApiError, request } from "../../lib/api";
import type { App, CreateAppInput, LifecycleAction } from "./types";

function isApp(value: unknown): value is App {
  if (typeof value !== "object" || value === null) return false;
  const app = value as Record<string, unknown>;
  return (
    typeof app.id === "number" &&
    typeof app.name === "string" &&
    typeof app.image === "string" &&
    typeof app.replicas === "number" &&
    typeof app.status === "string" &&
    typeof app.user_id === "number" &&
    typeof app.namespace === "string" &&
    typeof app.created_at === "string" &&
    typeof app.updated_at === "string"
  );
}

function requireApp(value: unknown): App {
  if (!isApp(value)) {
    throw new ApiError(-1, "服务器返回的应用数据无效，请重试。");
  }
  return value;
}

export async function getApps(): Promise<App[]> {
  const data = await request<unknown>("/apps", { auth: true });
  if (data == null) return [];
  if (!Array.isArray(data)) {
    throw new ApiError(-1, "服务器返回的应用列表无效，请重试。");
  }
  return data.map(requireApp);
}

export async function getApp(id: string): Promise<App> {
  return requireApp(await request<unknown>(`/apps/${encodeURIComponent(id)}`, { auth: true }));
}

export async function createApp(input: CreateAppInput): Promise<App> {
  return requireApp(
    await request<unknown>("/apps", {
      method: "POST",
      auth: true,
      body: JSON.stringify(input),
    }),
  );
}

export function deleteApp(id: number): Promise<void> {
  return request<void>(`/apps/${id}`, { method: "DELETE", auth: true });
}

export function runLifecycleAction(id: number, action: LifecycleAction): Promise<void> {
  return request<void>(`/apps/${id}/${action}`, { method: "POST", auth: true });
}

export async function getAppLogs(id: string): Promise<string> {
  const data = await request<unknown>(`/apps/${encodeURIComponent(id)}/logs?lines=100`, { auth: true });
  if (typeof data !== "object" || data === null || typeof (data as Record<string, unknown>).logs !== "string") {
    throw new ApiError(-1, "服务器返回的日志数据无效，请重试。");
  }
  return (data as { logs: string }).logs;
}
