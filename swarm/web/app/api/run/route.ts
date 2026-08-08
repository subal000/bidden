import { NextResponse } from "next/server";
import { canRun, getState, startRun } from "@/lib/runner";

export const dynamic = "force-dynamic";

export async function GET() {
  return NextResponse.json({ ...getState(), canRun: canRun() });
}

export async function POST(req: Request) {
  // The runner spawns the Go driver and agents, which do not exist on a hosted
  // deploy. Fail with something actionable rather than a spawn ENOENT.
  if (!canRun()) {
    return NextResponse.json(
      {
        error:
          "Running an auction needs the Go driver and agents, so it only works locally. " +
          "Clone the repo and run `npm run dev` from swarm/web with NEXT_PUBLIC_LIVE=true.",
      },
      { status: 501 },
    );
  }

  // The server picks a free job; the client does not need to know which.
  const body = await req.json().catch(() => ({}));
  const from = typeof body?.jobId === "number" ? body.jobId : 1;
  return NextResponse.json(startRun(from));
}
