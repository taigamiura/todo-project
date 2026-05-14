import { describe, expect, it } from "vitest";
import { loginSchema, signupSchema } from "@/features/auth/utils/authValidation";

describe("auth validation", () => {
  it("validates login input", () => {
    expect(loginSchema.safeParse({ email: "a@example.com", password: "secret" }).success).toBe(true);
    expect(loginSchema.safeParse({ email: "bad", password: "" }).success).toBe(false);
  });

  it("validates signup input", () => {
    expect(signupSchema.safeParse({ name: "Hanako", email: "a@example.com", password: "12345678" }).success).toBe(true);
    expect(signupSchema.safeParse({ name: "", email: "bad", password: "123" }).success).toBe(false);
  });
});