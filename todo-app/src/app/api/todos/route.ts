import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { todoSchema } from "@/features/todos/utils/todoValidation";
import { createApiError, toApiErrorPayload } from "@/lib/apiError";
import { requestBff } from "@/lib/server/bff";
import { createErrorResponse } from "@/lib/server/errorResponse";
import { SESSION_COOKIE_NAME } from "@/lib/server/session";

async function getToken() {
  const cookieStore = await cookies();
  const token = cookieStore.get(SESSION_COOKIE_NAME)?.value;

  if (!token) {
    throw createApiError({
      status: 401,
      code: "AUTH_REQUIRED",
      message: "認証が必要です。",
    });
  }

  return token;
}

export async function GET() {
  try {
    const payload = await requestBff("/v1/todos", {
      token: await getToken(),
    });

    return NextResponse.json(payload);
  } catch (error) {
    return createErrorResponse(error, {
      status: 500,
      code: "TODO_LIST_FAILED",
      message: "Todo の取得に失敗しました。",
    });
  }
}

export async function POST(request: Request) {
  const payload = await request.json();
  const parsed = todoSchema.safeParse(payload);

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
    const payload = await requestBff("/v1/todos", {
      method: "POST",
      token: await getToken(),
      body: parsed.data,
    });

    return NextResponse.json(payload, { status: 201 });
  } catch (error) {
    return createErrorResponse(error, {
      status: 500,
      code: "TODO_CREATE_FAILED",
      message: "Todo の作成に失敗しました。",
    });
  }
}