import { redirect } from "next/navigation";
import { getModule } from "@/lib/api";

export default async function ModuleRedirect({ params }: { params: { moduleKey: string } }) {
  const module = await getModule(params.moduleKey);
  redirect(`/docs/${params.moduleKey}/${module.default_version}`);
}
