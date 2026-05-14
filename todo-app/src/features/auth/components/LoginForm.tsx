"use client";

import { useState, useSyncExternalStore } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/Button";
import { Field } from "@/components/ui/Field";
import { Input } from "@/components/ui/Input";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { LoginInput } from "@/features/auth/types/auth";
import { loginSchema } from "@/features/auth/utils/authValidation";
import { ROUTES } from "@/lib/constants/routes";

function subscribeHydration() {
    return () => undefined;
}

export function LoginForm() {
    const router = useRouter();
    const { login } = useAuth();
    const [submitError, setSubmitError] = useState("");
    const isHydrated = useSyncExternalStore(subscribeHydration, () => true, () => false);
    const {
        register,
        handleSubmit,
        formState: { errors, isSubmitting },
    } = useForm<LoginInput>({
        resolver: zodResolver(loginSchema),
        defaultValues: {
            email: "",
            password: "",
        },
    });

    const onSubmit = handleSubmit(async (value) => {
        setSubmitError("");

        try {
            await login(value);
            router.replace(ROUTES.todos);
        } catch (error) {
            setSubmitError(error instanceof Error ? error.message : "ログインに失敗しました。");
        }
    });

    return (
        <form className="space-y-5" onSubmit={onSubmit}>
            <Field label="メールアドレス" htmlFor="email" error={errors.email?.message}>
                <Input
                    id="email"
                    type="email"
                    autoComplete="email"
                    placeholder="name@example.com"
                    {...register("email")}
                />
            </Field>

            <Field label="パスワード" htmlFor="password" error={errors.password?.message}>
                <Input
                    id="password"
                    type="password"
                    autoComplete="current-password"
                    placeholder="8文字以上で入力"
                    {...register("password")}
                />
            </Field>

            {submitError ? <p className="text-sm text-rose-600">{submitError}</p> : null}

            <Button fullWidth disabled={isSubmitting || !isHydrated} type="submit">
                {isSubmitting ? "ログイン中..." : "ログイン"}
            </Button>
        </form>
    );
}