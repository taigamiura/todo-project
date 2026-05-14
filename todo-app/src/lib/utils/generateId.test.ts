import { describe, expect, it } from "vitest";
import { generateId } from "@/lib/utils/generateId";

describe("generateId", () => {
  it("generates unique-looking ids", () => {
    const first = generateId();
    const second = generateId();
    expect(first).toContain("-");
    expect(first).not.toBe(second);
  });
});