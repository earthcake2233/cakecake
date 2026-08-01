import { describe, it, expect } from "vitest";
import {
  validateCakecakeUsername,
  validateCakecakeRegisterPassword,
  cakecakeErrorMessage,
  mapCakecakeLoginFailureMessage
} from "@/utils/cakecakeAuthRules";

describe("validateCakecakeUsername", () => {
  it("accepts valid usernames", () => {
    expect(validateCakecakeUsername("测试用户")).toBe("");
    expect(validateCakecakeUsername("cake_user")).toBe("");
    expect(validateCakecakeUsername("abc123")).toBe("");
    expect(validateCakecakeUsername("用户123")).toBe("");
  });

  it("rejects too short names", () => {
    const err = validateCakecakeUsername("ab");
    expect(err).not.toBe("");
  });

  it("rejects too long names", () => {
    const err = validateCakecakeUsername("a".repeat(33));
    expect(err).not.toBe("");
  });

  it("rejects special characters", () => {
    const err = validateCakecakeUsername("user@name");
    expect(err).not.toBe("");
    const err2 = validateCakecakeUsername("user name");
    expect(err2).not.toBe("");
  });

  it("handles empty input", () => {
    const err = validateCakecakeUsername("");
    expect(err).not.toBe("");
  });
});

describe("validateCakecakeRegisterPassword", () => {
  it("accepts passwords >= 8 chars", () => {
    expect(validateCakecakeRegisterPassword("12345678")).toBe("");
    expect(validateCakecakeRegisterPassword("a".repeat(20))).toBe("");
  });

  it("rejects short passwords", () => {
    const err = validateCakecakeRegisterPassword("1234567");
    expect(err).not.toBe("");
  });

  it("handles empty input", () => {
    const err = validateCakecakeRegisterPassword("");
    expect(err).not.toBe("");
  });
});

describe("cakecakeErrorMessage", () => {
  it("extracts msg from response data", () => {
    const err = { response: { data: { msg: "业务错误" } } };
    expect(cakecakeErrorMessage(err)).toBe("业务错误");
  });

  it("falls back to error.message", () => {
    const err = { message: "网络超时" };
    expect(cakecakeErrorMessage(err)).toBe("网络超时");
  });

  it("uses fallback as last resort", () => {
    expect(cakecakeErrorMessage({})).toBe("请求失败");
    expect(cakecakeErrorMessage(null, "自定义")).toBe("自定义");
  });
});

describe("mapCakecakeLoginFailureMessage", () => {
  it("masks 401 errors as generic message", () => {
    const err = { response: { data: { code: 40100 } } };
    expect(mapCakecakeLoginFailureMessage(err)).toBe("用户名或密码错误");
  });

  it("filters token-related messages", () => {
    const err = { response: { data: { msg: "Token已过期" } } };
    expect(mapCakecakeLoginFailureMessage(err)).toBe("用户名或密码错误");
  });

  it("passes through other messages", () => {
    const err = { response: { data: { msg: "账号已被封禁" } } };
    expect(mapCakecakeLoginFailureMessage(err)).toBe("账号已被封禁");
  });

  it("defaults for unknown errors", () => {
    expect(mapCakecakeLoginFailureMessage({})).toBe("用户名或密码错误");
  });
});
