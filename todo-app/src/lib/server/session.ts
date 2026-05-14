import { cookies } from "next/headers";
import { jwtVerify, type JWTPayload } from "jose";
import type { components } from "@/lib/api/generated/bff";

export const SESSION_COOKIE_NAME = "todo_session";

export type SessionUser = components["schemas"]["User"];
export type SessionPayload = components["schemas"]["SessionResponse"];

type SessionClaims = JWTPayload & {
  sub: string;
  name: string;
  email: string;
};

function getSessionSecret() {
  const secret = process.env.APP_SESSION_SECRET;

  if (!secret) {
    throw new Error("APP_SESSION_SECRET is not configured.");
  }

  return new TextEncoder().encode(secret);
}

export async function verifySessionToken(token: string): Promise<SessionUser | null> {
  try {
    const { payload } = await jwtVerify(token, getSessionSecret(), {
      algorithms: ["HS256"],
    });
    const claims = payload as SessionClaims;

    if (!claims.sub || !claims.email || !claims.name) {
      return null;
    }

    return {
      id: claims.sub,
      name: claims.name,
      email: claims.email,
    };
  } catch {
    return null;
  }
}

export async function getServerSessionUser() {
  const cookieStore = await cookies();
  const token = cookieStore.get(SESSION_COOKIE_NAME)?.value;

  if (!token) {
    return null;
  }

  return verifySessionToken(token);
}

export function getSessionCookieOptions() {
  return {
    httpOnly: true,
    sameSite: "lax" as const,
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: 60 * 60 * 12,
  };
}