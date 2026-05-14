"use client";

import { useState, useSyncExternalStore } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/Button";
import { Field } from "@/components/ui/Field";
import { Input } from "@/components/ui/Input";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { SignupInput } from "@/features/auth/types/auth";
import { signupSchema } from "@/features/auth/utils/authValidation";
import { ROUTES } from "@/lib/constants/routes";

function subscribeHydration() {
    return () => undefined;
}

export function SignupForm() {
    const router = useRouter();
    const { signup } = useAuth();
    const [submitError, setSubmitError] = useState("");
    const isHydrated = useSyncExternalStore(subscribeHydration, () => true, () => false);
    const {
        register,
        handleSubmit,
        formState: { errors, isSubmitting },
    } = useForm<SignupInput>({
        resolver: zodResolver(signupSchema),
        defaultValues: {
            name: "",
            email: "",
            password: "",
        },
    });

    const onSubmit = handleSubmit(async (value) => {
        setSubmitError("");

        try {
            await signup(value);
            router.replace(ROUTES.todos);
        } catch (error) {
            setSubmitError(error instanceof Error ? error.message : "登録に失敗しました。");
        }
    });

    return (
        <form className="space-y-5" onSubmit={onSubmit}>
            <Field label="名前" htmlFor="name" error={errors.name?.message}>
                <Input
                    id="name"
                    autoComplete="name"
                    placeholder="山田 花子"
                    {...register("name")}
                />
            </Field>

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
                    autoComplete="new-password"
                    placeholder="8文字以上で入力"
                    {...register("password")}
                />
            </Field>

            {submitError ? <p className="text-sm text-rose-600">{submitError}</p> : null}

            <Button fullWidth className="bg-rose-600 hover:bg-rose-700" disabled={isSubmitting || !isHydrated} type="submit">
                {isSubmitting ? "登録中..." : "会員登録"}
            </Button>
        </form>
    );
}