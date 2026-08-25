import { ApiError, request } from "../../lib/api";
import type { CreateProjectInput, Project } from "./types";

export function parseProject(value: unknown): Project {
  if (typeof value !== "object" || value === null) {
    throw new ApiError(-1, "服务器返回的项目数据无效，请重试。");
  }
  const project = value as Record<string, unknown>;
  if (
    typeof project.id !== "number" ||
    typeof project.name !== "string" ||
    typeof project.user_id !== "number" ||
    typeof project.namespace !== "string" ||
    typeof project.created_at !== "string" ||
    typeof project.updated_at !== "string"
  ) {
    throw new ApiError(-1, "服务器返回的项目数据无效，请重试。");
  }
  return project as unknown as Project;
}

export function parseProjects(value: unknown): Project[] {
  if (value == null) return [];
  if (!Array.isArray(value)) {
    throw new ApiError(-1, "服务器返回的项目列表无效，请重试。");
  }
  return value.map(parseProject);
}

export async function getProjects(): Promise<Project[]> {
  return parseProjects(await request<unknown>("/projects", { auth: true }));
}

export async function getProject(id: string | number): Promise<Project> {
  return parseProject(await request<unknown>(`/projects/${encodeURIComponent(String(id))}`, { auth: true }));
}

export async function createProject(input: CreateProjectInput): Promise<Project> {
  return parseProject(
    await request<unknown>("/projects", {
      method: "POST",
      auth: true,
      body: JSON.stringify(input),
    }),
  );
}

export function deleteProject(id: number): Promise<void> {
  return request<void>(`/projects/${id}`, { method: "DELETE", auth: true });
}
