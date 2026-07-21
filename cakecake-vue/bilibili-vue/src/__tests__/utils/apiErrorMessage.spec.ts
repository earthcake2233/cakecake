import { describe, it, expect } from "vitest";
import { extractApiErrorMessage } from "@/utils/apiErrorMessage";

describe("extractApiErrorMessage", () => {
  it("extracts msg from response data", () => {
    const error = {
      response: { data: { msg: "用户名已存在" } }
    };
    expect(extractApiErrorMessage(error)).toBe("用户名已存在");
  });

  it("falls back to message field", () => {
    const error = {
      response: { data: { message: "服务器错误" } }
    };
    expect(extractApiErrorMessage(error)).toBe("服务器错误");
  });

  it("prefers msg over message", () => {
    const error = {
      response: { data: { msg: "业务错误", message: "系统错误" } }
    };
    expect(extractApiErrorMessage(error)).toBe("业务错误");
  });

  it("ignores generic axios status messages", () => {
    const error = {
      message: "Request failed with status code 500"
    };
    expect(extractApiErrorMessage(error)).toBe("加载失败");
  });

  it("uses fallback for unknown errors", () => {
    expect(extractApiErrorMessage(null)).toBe("加载失败");
    expect(extractApiErrorMessage({})).toBe("加载失败");
  });

  it("supports custom fallback", () => {
    expect(extractApiErrorMessage(null, "自定义错误")).toBe("自定义错误");
  });

  it("handles string response data", () => {
    const error = {
      response: { data: "原始错误字符串" }
    };
    expect(extractApiErrorMessage(error)).toBe("原始错误字符串");
  });
});
