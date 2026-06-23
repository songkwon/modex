import { NextRequest, NextResponse } from "next/server";

type SkillEntry = {
  url?: string;
  [key: string]: unknown;
};

type SkillDiscovery = {
  skills?: SkillEntry[];
  [key: string]: unknown;
};

const DISCOVERY_PATH = "/.well-known/agent-skills/index.json";

function internalApiBaseURL() {
  return (process.env.INTERNAL_API_BASE_URL || process.env.MODEX_PUBLIC_API_BASE_URL || "http://localhost:8671").replace(
    /\/+$/,
    ""
  );
}

function absoluteURL(req: NextRequest, path: string) {
  if (/^https?:\/\//i.test(path)) return path;
  return new URL(path, req.nextUrl.origin).toString();
}

export async function skillDiscoveryGET(req: NextRequest) {
  const res = await fetch(`${internalApiBaseURL()}${DISCOVERY_PATH}`, { cache: "no-store" });
  if (!res.ok) {
    return NextResponse.json(
      {
        error: "skill_discovery_unavailable",
        message: await res.text()
      },
      { status: res.status }
    );
  }

  const body = (await res.json()) as SkillDiscovery;
  const skills = Array.isArray(body.skills)
    ? body.skills.map((skill) => ({
        ...skill,
        url: typeof skill.url === "string" ? absoluteURL(req, skill.url) : skill.url
      }))
    : body.skills;

  return NextResponse.json(
    {
      ...body,
      skills
    },
    {
      headers: {
        "Cache-Control": "public, max-age=300"
      }
    }
  );
}
