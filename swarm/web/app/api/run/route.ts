import { NextResponse } from "next/server";
import { getState, startRun } from "@/lib/runner";

export const dynamic = "force-dynamic";

export async function GET() {
  return NextResponse.json(getState());
}

export async function POST(req: Request) {
  // The server picks a free job; the client does not need to know which.
  const body = await req.json().catch(() => ({}));
  const from = typeof body?.jobId === "number" ? body.jobId : 1;
  return NextResponse.json(startRun(from));
}
