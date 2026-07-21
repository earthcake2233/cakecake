import { describe, it, expect } from "vitest";
import {
  validateMinibiliUsername,
  validateMinibiliRegisterPassword,
  minibiliErrorMessage,
  mapMinibiliLoginFailureMessage
} from "@/utils/minibiliAuthRules";

describe("validateMinibiliUsername", () => {
  it("accepts valid usernames", () => {
    expect(validateMinibiliUsername("测试用户")).toBe("");
    expect(validateMinibiliUsername("cake_user")).toBe("");
    expect(validateMinibiliUsername("abc123")).toBe("");
    expect(validateMinibiliUsername("用户123")).toBe("");
  });

  it("rejects too short names", () => {
    const err = validateMinibiliUsername("ab");
    expect(err).not.toBe("");
  });

  it("rejects too long names", () => {
    const err = validateMinibiliUsername("a".repeat(33));
    expect(err).not.toBe("");
  });

  it("rejects special characters", () => {
    const err = validateMinibiliUsername("user@name");
    expect(err).not.toBe("");
    const err2 = validateMinibiliUsername("user name");
    expect(err2).not.toBe("");
  });

  it("handles empty input", () => {
    const err = validateMinibiliUsername("");
    expect(err).not.toBe("");
  });
});

describe("validateMinibiliRegisterPassword", () => {
  it("accepts passwords >= 8 chars", () => {
    expect(validateMinibiliRegisterPassword("12345678")).toBe("");
    expect(validateMinibiliRegisterPassword("a".repeat(20))).toBe("");
  });

  it("rejects short passwords", () => {
    const err = validateMinibiliRegisterPassword("1234567");
    expect(err).not.toBe("");
  });

  it("handles empty input", () => {
    const err = validateMinibiliRegisterPassword("");
    expect(err).not.toBe("");
  });
});

describe("minibiliErrorMessage", () => {
  it("extracts msg from response data", () => {
    const err = { response: { data: { msg: "业务错误" } } };
    expect(minibiliErrorMessage(err)).toBe("业务错误");
  });

  it("falls back to error.message", () => {
    const err = { message: "网络超时" };
    expect(minibiliErrorMessage(err)).toBe("网络超时");
  });

  it("uses fallback as last resort", () => {
    expect(minibiliErrorMessage({})).toBe("请求失败");
    expect(minibiliErrorMessage(null, "自定义")).toBe("自定义");
  });
});

describe("mapMinibiliLoginFailureMessage", () => {
  it("masks 401 errors as generic message", () => {
    const err = { response: { data: { code: 40100 } } };
    expect(mapMinibiliLoginFailureMessage(err)).toBe("用户名或密码错误");
  });

  it("filters token-related messages", () => {
    const err = { response: { data: { msg: "Token已过期" } } };
    expect(mapMinibiliLoginFailureMessage(err)).toBe("用户名或密码错误");
  });

  it("passes through other messages", () => {
    const err = { response: { data: { msg: "账号已被封禁" } } };
    expect(mapMinibiliLoginFailureMessage(err)).toBe("账号已被封禁");
  });

  it("defaults for unknown errors", () => {
    expect(mapMinibiliLoginFailureMessage({})).toBe("用户名或密码错误");
  });
});
