// @vitest-environment jsdom

import { act, type ReactNode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AppDetailPage } from "../../apps/pages/AppDetailPage";
import { AppsListPage } from "../../apps/pages/AppsListPage";
import type { App } from "../../apps/types";
import { CreateProjectPage } from "./CreateProjectPage";
import { ProjectsListPage } from "./ProjectsListPage";
import type { Project } from "../types";
import { createProject, deleteProject, getProject, getProjects } from "../api";
import { deleteApp, getApp, getAppLogs, getApps, runLifecycleAction } from "../../apps/api";

vi.mock("../api", () => ({
  createProject: vi.fn(),
  deleteProject: vi.fn(),
  getProject: vi.fn(),
  getProjects: vi.fn(),
}));
vi.mock("../../apps/api", () => ({
  deleteApp: vi.fn(),
  getApp: vi.fn(),
  getAppLogs: vi.fn(),
  getApps: vi.fn(),
  runLifecycleAction: vi.fn(),
}));

const project: Project = {
  id: 1,
  name: "项目一",
  user_id: 7,
  namespace: "astro-project-one",
  created_at: "2026-08-25T00:00:00Z",
  updated_at: "2026-08-25T00:00:00Z",
};

let container: HTMLDivElement;
let root: Root | null;

beforeEach(() => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  Object.defineProperty(HTMLDialogElement.prototype, "showModal", {
    configurable: true,
    value() { this.setAttribute("open", ""); },
  });
  Object.defineProperty(HTMLDialogElement.prototype, "close", {
    configurable: true,
    value() { this.removeAttribute("open"); },
  });
  container = document.createElement("div");
  document.body.append(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => root?.unmount());
  root = null;
  container.remove();
  vi.resetAllMocks();
});

async function renderRoute(path: string, routePath: string, element: ReactNode) {
  const router = createMemoryRouter([{ path: routePath, element }], { initialEntries: [path] });
  await act(async () => {
    root?.render(<RouterProvider router={router} />);
  });
  return router;
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((resolvePromise) => { resolve = resolvePromise; });
  return { promise, resolve };
}

describe("项目页面", () => {
  it("无项目时只引导创建第一个项目", async () => {
    vi.mocked(getProjects).mockResolvedValue([]);
    await renderRoute("/projects", "/projects", <ProjectsListPage />);

    expect(container.textContent).toContain("还没有项目");
    expect(container.textContent).toContain("先创建一个项目，再在其中部署应用。");
    expect(container.querySelector('a[href="/projects/new"]')).not.toBeNull();
  });

  it("创建项目后进入新项目的应用页", async () => {
    vi.mocked(createProject).mockResolvedValue(project);
    const router = createMemoryRouter([
      { path: "/projects/new", element: <CreateProjectPage /> },
      { path: "/projects/:projectId/apps", element: <div>项目应用页</div> },
    ], { initialEntries: ["/projects/new"] });
    await act(async () => { root?.render(<RouterProvider router={router} />); });

    const input = container.querySelector<HTMLInputElement>("#project-name");
    const form = container.querySelector<HTMLFormElement>("form");
    if (!input || !form) throw new Error("未渲染项目创建表单");
    await act(async () => {
      const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set;
      setter?.call(input, "  项目一  ");
      input.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () => { form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })); });

    expect(createProject).toHaveBeenCalledWith({ name: "项目一" });
    expect(router.state.location.pathname).toBe("/projects/1/apps");
  });

  it("确认后删除空项目并更新列表", async () => {
    vi.mocked(getProjects).mockResolvedValue([project]);
    vi.mocked(deleteProject).mockResolvedValue(undefined);
    await renderRoute("/projects", "/projects", <ProjectsListPage />);

    const deleteButton = container.querySelector<HTMLButtonElement>('[aria-label="删除项目 项目一"]');
    if (!deleteButton) throw new Error("未渲染删除按钮");
    await act(async () => { deleteButton.click(); });
    const form = container.querySelector<HTMLFormElement>("dialog form");
    if (!form) throw new Error("未渲染删除确认表单");
    await act(async () => { form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })); });

    expect(deleteProject).toHaveBeenCalledWith(1);
    expect(container.textContent).toContain("项目“项目一”已删除。");
  });
});

describe("项目应用页面", () => {
  it("切换项目后忽略较晚返回的旧项目请求", async () => {
    const oldProject = deferred<Project>();
    const oldApps = deferred<App[]>();
    const newProject = deferred<Project>();
    const newApps = deferred<App[]>();
    const projectTwo = { ...project, id: 2, name: "项目二", namespace: "astro-project-two" };
    const appOne: App = { id: 1, name: "app-one", image: "nginx:alpine", replicas: 1, status: "running", project_id: 1, created_at: project.created_at, updated_at: project.updated_at };
    const appTwo: App = { ...appOne, id: 2, name: "app-two", project_id: 2 };
    vi.mocked(getProject).mockImplementation((id) => String(id) === "1" ? oldProject.promise : newProject.promise);
    vi.mocked(getApps).mockImplementation((id) => String(id) === "1" ? oldApps.promise : newApps.promise);

    const router = await renderRoute("/projects/1/apps", "/projects/:projectId/apps", <AppsListPage />);
    await act(async () => { await router.navigate("/projects/2/apps"); });
    await act(async () => { newProject.resolve(projectTwo); newApps.resolve([appTwo]); });
    expect(container.textContent).toContain("项目二");
    expect(container.textContent).toContain("app-two");

    await act(async () => { oldProject.resolve(project); oldApps.resolve([appOne]); });
    expect(container.textContent).toContain("项目二");
    expect(container.textContent).not.toContain("app-one");
  });

  it("切换应用后忽略旧生命周期请求的完成结果", async () => {
    const lifecycle = deferred<void>();
    const appOne: App = { id: 1, name: "app-one", image: "nginx:alpine", replicas: 1, status: "running", project_id: 1, created_at: project.created_at, updated_at: project.updated_at };
    const appTwo: App = { ...appOne, id: 2, name: "app-two", project_id: 2 };
    vi.mocked(getApp).mockImplementation((_, id) => Promise.resolve(String(id) === "1" ? appOne : appTwo));
    vi.mocked(getAppLogs).mockResolvedValue("");
    vi.mocked(runLifecycleAction).mockReturnValue(lifecycle.promise);

    const router = await renderRoute("/projects/1/apps/1", "/projects/:projectId/apps/:id", <AppDetailPage />);
    const stopButton = Array.from(container.querySelectorAll("button")).find((button) => button.textContent?.includes("停止"));
    if (!stopButton) throw new Error("未渲染停止按钮");
    await act(async () => { stopButton.click(); });
    await act(async () => { await router.navigate("/projects/2/apps/2"); });
    lifecycle.resolve();
    await act(async () => { await lifecycle.promise; });

    expect(router.state.location.pathname).toBe("/projects/2/apps/2");
    expect(container.textContent).toContain("app-two");
    expect(container.textContent).not.toContain("停止请求已提交");
  });

  it("切换应用后旧删除请求不会跳回原项目", async () => {
    const deletion = deferred<void>();
    const appOne: App = { id: 1, name: "app-one", image: "nginx:alpine", replicas: 1, status: "running", project_id: 1, created_at: project.created_at, updated_at: project.updated_at };
    const appTwo: App = { ...appOne, id: 2, name: "app-two", project_id: 2 };
    vi.mocked(getApp).mockImplementation((_, id) => Promise.resolve(String(id) === "1" ? appOne : appTwo));
    vi.mocked(getAppLogs).mockResolvedValue("");
    vi.mocked(deleteApp).mockReturnValue(deletion.promise);

    const router = await renderRoute("/projects/1/apps/1", "/projects/:projectId/apps/:id", <AppDetailPage />);
    const deleteButton = Array.from(container.querySelectorAll("button")).find((button) => button.textContent?.trim() === "删除应用");
    if (!deleteButton) throw new Error("未渲染删除应用按钮");
    await act(async () => { deleteButton.click(); });
    const form = container.querySelector<HTMLFormElement>("dialog form");
    if (!form) throw new Error("未渲染删除应用确认表单");
    await act(async () => { form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })); });
    await act(async () => { await router.navigate("/projects/2/apps/2"); });
    deletion.resolve();
    await act(async () => { await deletion.promise; });

    expect(router.state.location.pathname).toBe("/projects/2/apps/2");
    expect(container.textContent).toContain("app-two");
  });
});
