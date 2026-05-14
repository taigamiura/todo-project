import { describe, expect, it } from "vitest";
import { ROUTES } from "@/lib/constants/routes";

describe("routes", () => {
  it("exposes route helpers", () => {
    expect(ROUTES.home).toBe("/");
    expect(ROUTES.todoDetail("123")).toBe("/todos/123");
  });
});