import { LoginInput, SignupInput, User } from "@/features/auth/types/auth";
import { parseApiErrorPayload } from "@/lib/apiError";

const AUTH_STORAGE_EVENT = "todo-app:auth-storage";

async function requestAuth<T>(path: string, options: RequestInit = {}) {
  const response = await fetch(path, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(options.headers ?? {}),
    },
    cache: "no-store",
  });
  const payload = (await response.json().catch(() => null)) as { error?: unknown; user?: T } | null;

  if (!response.ok) {
    throw parseApiErrorPayload(payload, {
      status: response.status,
      code: response.status === 401 ? "AUTH_INVALID_CREDENTIALS" : "AUTH_REQUEST_FAILED",
      message: "認証処理に失敗しました。",
    });
  }

  if (typeof window !== "undefined") {
    window.dispatchEvent(new Event(AUTH_STORAGE_EVENT));
  }

  return payload?.user as T;
}

async function requestAuthWithoutPayload(path: string, options: RequestInit = {}) {
  const response = await fetch(path, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(options.headers ?? {}),
    },
    cache: "no-store",
  });
  const payload = (await response.json().catch(() => null)) as { error?: unknown } | null;

  if (!response.ok) {
    throw parseApiErrorPayload(payload, {
      status: response.status,
      code: "AUTH_REQUEST_FAILED",
      message: "認証処理に失敗しました。",
    });
  }

  if (typeof window !== "undefined") {
    window.dispatchEvent(new Event(AUTH_STORAGE_EVENT));
  }
}

export function subscribeAuthStorage(onStoreChange: () => void) {
  if (typeof window === "undefined") {
    return () => undefined;
  }

  window.addEventListener(AUTH_STORAGE_EVENT, onStoreChange);

  return () => {
    window.removeEventListener(AUTH_STORAGE_EVENT, onStoreChange);
  };
}

export async function getCurrentUser(): Promise<User | null> {
  const response = await fetch("/api/auth/session", {
    method: "GET",
    cache: "no-store",
  });

  if (!response.ok) {
    return null;
  }

  const payload = (await response.json()) as { user: User | null };
  return payload.user;
}

export async function clearCurrentSession() {
  await requestAuthWithoutPayload("/api/auth/logout", {
    method: "POST",
  });
}

export function createUser(input: SignupInput) {
  return requestAuth<User>("/api/auth/signup", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function authenticateUser(input: LoginInput) {
  return requestAuth<User>("/api/auth/login", {
    method: "POST",
    body: JSON.stringify(input),
  });
}