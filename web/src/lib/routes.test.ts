import { describe, expect, it } from "vitest";
import { appDetailPath, createAppPath, projectAppsPath, projectsPath } from "./routes";

describe("项目应用路由", () => {
  it("始终保留项目上下文并编码路径参数", () => {
    expect(projectsPath).toBe("/projects");
    expect(projectAppsPath("a/b")).toBe("/projects/a%2Fb/apps");
    expect(createAppPath(3)).toBe("/projects/3/apps/new");
    expect(appDetailPath(3, 9)).toBe("/projects/3/apps/9");
  });
});
