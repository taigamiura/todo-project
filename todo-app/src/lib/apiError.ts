export type ApiErrorData = {
  code: string;
  message: string;
  status: number;
};

type ApiErrorObject = {
  error?: unknown;
};

export class ApiError extends Error {
  code: string;
  status: number;

  constructor(data: ApiErrorData) {
    super(data.message);
    this.name = "ApiError";
    this.code = data.code;
    this.status = data.status;
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

export function defaultErrorCode(status: number) {
  switch (status) {
    case 400:
      return "BAD_REQUEST";
    case 401:
      return "UNAUTHORIZED";
    case 404:
      return "NOT_FOUND";
    case 409:
      return "CONFLICT";
    case 502:
    case 504:
      return "BAD_GATEWAY";
    case 503:
      return "SERVICE_UNAVAILABLE";
    default:
      return "INTERNAL_SERVER_ERROR";
  }
}

export function createApiError(data: ApiErrorData) {
  return new ApiError(data);
}

export function parseApiErrorPayload(payload: unknown, fallback: ApiErrorData) {
  if (isRecord(payload)) {
    const typedPayload = payload as ApiErrorObject;
    if (typeof typedPayload.error === "string" && typedPayload.error.trim() !== "") {
      return new ApiError({ ...fallback, message: typedPayload.error });
    }

    if (isRecord(typedPayload.error) && typeof typedPayload.error.message === "string") {
      return new ApiError({
        code: typeof typedPayload.error.code === "string" ? typedPayload.error.code : fallback.code,
        message: typedPayload.error.message,
        status: typeof typedPayload.error.status === "number" ? typedPayload.error.status : fallback.status,
      });
    }
  }

  return new ApiError(fallback);
}

export function normalizeApiError(error: unknown, fallback: ApiErrorData) {
  if (error instanceof ApiError) {
    return error;
  }

  if (error instanceof Error && error.message) {
    return new ApiError({ ...fallback, message: error.message });
  }

  return new ApiError(fallback);
}

export function toApiErrorPayload(error: ApiError | ApiErrorData) {
  if (error instanceof ApiError) {
    return {
      error: {
        code: error.code,
        message: error.message,
        status: error.status,
      },
    };
  }

  return { error };
}