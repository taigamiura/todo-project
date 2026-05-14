import type { components } from "@/lib/api/generated/bff";

export type User = components["schemas"]["User"];
export type LoginInput = components["schemas"]["LoginRequest"];
export type SignupInput = components["schemas"]["SignupRequest"];