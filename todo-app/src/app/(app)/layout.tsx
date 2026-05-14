import { redirect } from "next/navigation";
import { ROUTES } from "@/lib/constants/routes";
import { getServerSessionUser } from "@/lib/server/session";

export default async function AppLayout({ children }: Readonly<{ children: React.ReactNode }>) {
    const user = await getServerSessionUser();

    if (!user) {
        redirect(ROUTES.login);
    }

    return <>{children}</>;
}