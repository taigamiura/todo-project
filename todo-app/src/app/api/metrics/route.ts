import { NextResponse } from "next/server";
import { collectDefaultMetrics, register } from "prom-client";

export const dynamic = "force-dynamic";

const globalMetricsState = globalThis as typeof globalThis & {
  __todoAppMetricsInitialized?: boolean;
};

if (!globalMetricsState.__todoAppMetricsInitialized) {
  collectDefaultMetrics({
    prefix: "todo_app_",
    register,
  });
  globalMetricsState.__todoAppMetricsInitialized = true;
}

export async function GET() {
  return new NextResponse(await register.metrics(), {
    headers: {
      "Content-Type": register.contentType,
      "Cache-Control": "no-store, max-age=0",
    },
  });
}