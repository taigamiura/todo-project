import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { loginSchema } from "@/features/auth/utils/authValidation";
import { createApiError, toApiErrorPayload } from "@/lib/apiError";
import { requestBff } from "@/lib/server/bff";
import { createErrorResponse } from "@/lib/server/errorResponse";
import { getSessionCookieOptions, SESSION_COOKIE_NAME } from "@/lib/server/session";

export async function POST(request: Request) {
  const payload = await request.json();
  const parsed = loginSchema.safeParse(payload);

  if (!parsed.success) {
    return NextResponse.json(
      toApiErrorPayload(createApiError({
        status: 400,
        code: "VALIDATION_FAILED",
        message: parsed.error.issues[0]?.message ?? "入力内容が不正です。",
      })),
      { status: 400 },
    );
  }

  try {
    const result = await requestBff("/v1/auth/login", {
      method: "POST",
      body: parsed.data,
    });
    const cookieStore = await cookies();

    cookieStore.set(SESSION_COOKIE_NAME, result.accessToken, getSessionCookieOptions());

    return NextResponse.json({ user: result.user });
  } catch (error) {
    return createErrorResponse(error, {
      status: 401,
      code: "AUTHENTICATION_FAILED",
      message: "ログインに失敗しました。",
    });
  }
}