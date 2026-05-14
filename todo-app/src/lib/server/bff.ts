import type { operations, paths } from "@/lib/api/generated/bff";
import { ApiError, defaultErrorCode, parseApiErrorPayload } from "@/lib/apiError";

type RequestMethod = "GET" | "POST" | "PATCH" | "DELETE";
type BffPath = keyof paths;

type JsonContent<T> = T extends { content: { "application/json": infer Payload } } ? Payload : null;

type OperationFor<PathKey extends BffPath, Method extends RequestMethod> = Extract<
  paths[PathKey][Lowercase<Method>],
  operations[keyof operations]
>;

type MethodForPath<PathKey extends BffPath> = {
  [Method in RequestMethod]: paths[PathKey][Lowercase<Method>] extends never | undefined ? never : Method;
}[RequestMethod];

type SuccessResponse<PathKey extends BffPath, Method extends RequestMethod> = OperationFor<PathKey, Method> extends {
  responses: infer Responses;
}
  ? 200 extends keyof Responses
  ? JsonContent<Responses[200]>
  : 201 extends keyof Responses
  ? JsonContent<Responses[201]>
  : 204 extends keyof Responses
  ? null
  : never
  : never;

type RequestBody<PathKey extends BffPath, Method extends RequestMethod> = OperationFor<PathKey, Method> extends {
  requestBody: { content: { "application/json": infer Payload } };
}
  ? Payload
  : never;

type PathParams<PathKey extends BffPath> = paths[PathKey] extends { parameters: { path: infer Params } } ? Params : never;

type RequestBffOptions<PathKey extends BffPath, Method extends MethodForPath<PathKey>> = {
  method?: Method;
  token?: string;
  body?: RequestBody<PathKey, Method>;
  params?: PathParams<PathKey>;
};

function resolvePath<PathKey extends BffPath>(path: PathKey, params?: PathParams<PathKey>) {
  if (!params) {
    return path;
  }

  let resolvedPath = path as string;

  for (const [key, value] of Object.entries(params)) {
    resolvedPath = resolvedPath.replace(`{${key}}`, encodeURIComponent(String(value)));
  }

  return resolvedPath;
}

export function getBffBaseUrl() {
  return process.env.BFF_BASE_URL ?? "http://localhost:8080";
}

function getBffTimeoutMs() {
  const rawValue = process.env.BFF_REQUEST_TIMEOUT_MS ?? "4000";
  const parsedValue = Number(rawValue);

  if (!Number.isFinite(parsedValue) || parsedValue <= 0) {
    return 4000;
  }

  return parsedValue;
}

async function getTracePropagationHeaders() {
  try {
    const { headers } = await import("next/headers");
    const requestHeaders = await headers();
    const traceparent = requestHeaders.get("traceparent");
    const tracestate = requestHeaders.get("tracestate");
    const baggage = requestHeaders.get("baggage");

    return {
      ...(traceparent ? { traceparent } : {}),
      ...(tracestate ? { tracestate } : {}),
      ...(baggage ? { baggage } : {}),
    };
  } catch {
    return {};
  }
}

export async function requestBff<
  PathKey extends BffPath,
  Method extends MethodForPath<PathKey> = Extract<MethodForPath<PathKey>, "GET">,
>(
  path: PathKey,
  options = {} as RequestBffOptions<PathKey, Method>,
): Promise<SuccessResponse<PathKey, Method>> {
  let response: Response;
  const traceHeaders = await getTracePropagationHeaders();

  try {
    response = await fetch(`${getBffBaseUrl()}${resolvePath(path, options.params)}`, {
      method: options.method ?? "GET",
      headers: {
        ...traceHeaders,
        ...(options.token ? { Authorization: `Bearer ${options.token}` } : {}),
        ...(options.body ? { "Content-Type": "application/json" } : {}),
      },
      body: options.body ? JSON.stringify(options.body) : undefined,
      cache: "no-store",
      signal: AbortSignal.timeout(getBffTimeoutMs()),
    });
  } catch (error) {
    if (error instanceof Error && error.name === "TimeoutError") {
      throw new ApiError({
        status: 504,
        code: "BFF_REQUEST_TIMEOUT",
        message: "バックエンドの応答がタイムアウトしました。",
      });
    }

    throw error;
  }

  if (!response.ok) {
    const fallback = {
      status: response.status,
      code: defaultErrorCode(response.status),
      message: response.statusText || "バックエンドとの通信に失敗しました。",
    };

    try {
      throw parseApiErrorPayload(await response.json(), fallback);
    } catch (error) {
      if (error instanceof ApiError) {
        throw error;
      }

      throw new ApiError(fallback);
    }
  }

  if (response.status === 204) {
    return null as SuccessResponse<PathKey, Method>;
  }

  return (await response.json()) as SuccessResponse<PathKey, Method>;
}