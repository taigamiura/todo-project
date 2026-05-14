import { describe, expect, it } from "vitest";
import { todoSchema } from "@/features/todos/utils/todoValidation";

describe("todo validation", () => {
  it("accepts valid values", () => {
    expect(todoSchema.safeParse({ title: "title", description: "desc", completed: false }).success).toBe(true);
  });

  it("rejects invalid values", () => {
    expect(todoSchema.safeParse({ title: "", description: "x".repeat(301), completed: false }).success).toBe(false);
  });
});