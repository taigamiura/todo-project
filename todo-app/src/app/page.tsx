import { redirect } from "next/navigation";
import { ROUTES } from "@/lib/constants/routes";
import { getServerSessionUser } from "@/lib/server/session";

export default async function Home() {
  const user = await getServerSessionUser();

  redirect(user ? ROUTES.todos : ROUTES.login);
}
