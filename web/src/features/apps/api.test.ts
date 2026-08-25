import { describe, expect, it } from "vitest";
import { parseApp, parseApps } from "./api";

const app = {
  id: 4,
  name: "demo",
  image: "nginx:latest",
  replicas: 1,
  status: "running",
  project_id: 3,
  created_at: "2026-08-24T00:00:00Z",
  updated_at: "2026-08-24T00:00:00Z",
};

describe("应用响应解码", () => {
  it("要求应用携带项目归属", () => {
    expect(parseApp(app)).toEqual(app);
    expect(() => parseApp({ ...app, project_id: undefined })).toThrow("应用数据无效");
  });

  it("把空列表响应归一化", () => {
    expect(parseApps(null)).toEqual([]);
  });
});
