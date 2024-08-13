import { NextResponse } from "next/server";

// biome-ignore lint/suspicious/useAwait: its fiiiiiine
export async function GET(): Promise<NextResponse> {
  return NextResponse.json({ status: "OK" }, { status: 200 });
}
