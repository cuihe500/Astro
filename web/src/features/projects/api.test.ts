import { describe, expect, it } from "vitest";
import { parseProject, parseProjects } from "./api";

const project = {
  id: 1,
  name: "个人网站",
  user_id: 2,
  namespace: "astro-project-123",
  created_at: "2026-08-24T00:00:00Z",
  updated_at: "2026-08-24T00:00:00Z",
};

describe("项目响应解码", () => {
  it("接受完整项目并把空列表响应归一化", () => {
    expect(parseProject(project)).toEqual(project);
    expect(parseProjects(null)).toEqual([]);
  });

  it("拒绝缺少归属字段的项目", () => {
    expect(() => parseProject({ id: 1, name: "个人网站" })).toThrow("项目数据无效");
  });
});
