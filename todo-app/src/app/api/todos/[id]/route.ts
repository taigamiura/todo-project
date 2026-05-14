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

type RouteContext = {
  params: Promise<{ id: string }>;
};

export async function GET(_: Request, context: RouteContext) {
  try {
    const { id } = await context.params;
    const payload = await requestBff("/v1/todos/{id}", {
      token: await getToken(),
      params: { id },
    });

    return NextResponse.json(payload);
  } catch (error) {
    return createErrorResponse(error, {
      status: 500,
      code: "TODO_FETCH_FAILED",
      message: "Todo の取得に失敗しました。",
    });
  }
}

export async function PATCH(request: Request, context: RouteContext) {
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
    const { id } = await context.params;
    const payload = await requestBff("/v1/todos/{id}", {
      method: "PATCH",
      token: await getToken(),
      body: parsed.data,
      params: { id },
    });

    return NextResponse.json(payload);
  } catch (error) {
    return createErrorResponse(error, {
      status: 500,
      code: "TODO_UPDATE_FAILED",
      message: "Todo の更新に失敗しました。",
    });
  }
}

export async function DELETE(_: Request, context: RouteContext) {
  try {
    const { id } = await context.params;
    await requestBff("/v1/todos/{id}", {
      method: "DELETE",
      token: await getToken(),
      params: { id },
    });

    return new NextResponse(null, { status: 204 });
  } catch (error) {
    return createErrorResponse(error, {
      status: 500,
      code: "TODO_DELETE_FAILED",
      message: "Todo の削除に失敗しました。",
    });
  }
}