import { NextResponse } from "next/server";
import { createApiError, toApiErrorPayload } from "@/lib/apiError";
import { getServerSessionUser } from "@/lib/server/session";

export async function GET() {
  const user = await getServerSessionUser();

  if (!user) {
    return NextResponse.json(
      toApiErrorPayload(createApiError({
        status: 401,
        code: "AUTH_REQUIRED",
        message: "認証が必要です。",
      })),
      { status: 401 },
    );
  }

  return NextResponse.json({ user });
}