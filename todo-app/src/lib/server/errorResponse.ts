import { NextResponse } from "next/server";
import { normalizeApiError, toApiErrorPayload, type ApiErrorData } from "@/lib/apiError";

export function createErrorResponse(error: unknown, fallback: ApiErrorData) {
  const normalized = normalizeApiError(error, fallback);

  return NextResponse.json(toApiErrorPayload(normalized), {
    status: normalized.status,
  });
}