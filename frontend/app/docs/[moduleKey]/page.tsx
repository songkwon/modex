import { redirect } from "next/navigation";
import { getModule } from "@/lib/api";

export default async function ModuleRedirect({ params }: { params: Promise<{ moduleKey: string }> }) {
  const { moduleKey } = await params;
  const module = await getModule(moduleKey);
  redirect(`/docs/${moduleKey}/${module.default_version}`);
}
