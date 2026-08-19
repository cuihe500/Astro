import { describe, expect, it } from "vitest";
import { isAuthenticationError, unwrapApiResponse } from "./api";

describe("unwrapApiResponse", () => {
  it("returns data only when the business code is zero", () => {
    expect(unwrapApiResponse<{ token: string }>({ code: 0, message: "成功", data: { token: "jwt" } })).toEqual({
      token: "jwt",
    });
  });

  it("accepts successful responses without data", () => {
    expect(unwrapApiResponse<void>({ code: 0, message: "成功" })).toBeUndefined();
  });

  it("throws the backend business error even when HTTP would be successful", () => {
    expect(() => unwrapApiResponse({ code: 21001, message: "应用不存在" })).toThrowError(
      expect.objectContaining({ code: 21001, message: "应用不存在" }),
    );
  });

  it("rejects malformed envelopes", () => {
    expect(() => unwrapApiResponse({ message: "成功" })).toThrow("服务器返回了无法识别的响应");
  });
});

describe("isAuthenticationError", () => {
  it.each([10002, 20011, 20012])("recognizes code %i", (code) => {
    expect(isAuthenticationError(code)).toBe(true);
  });

  it("does not classify ordinary business errors as session failures", () => {
    expect(isAuthenticationError(20003)).toBe(false);
  });
});
