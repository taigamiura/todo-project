"use client";

import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/Button";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { ROUTES } from "@/lib/constants/routes";

export function LogoutButton() {
    const router = useRouter();
    const { logout } = useAuth();

    return (
        <Button
            variant="secondary"
            onClick={async () => {
                await logout();
                router.replace(ROUTES.login);
            }}
        >
            ログアウト
        </Button>
    );
}