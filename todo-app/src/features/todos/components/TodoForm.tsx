"use client";

import { useEffect, useRef, useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { Controller, useForm } from "react-hook-form";
import { Button } from "@/components/ui/Button";
import { Field } from "@/components/ui/Field";
import { Input } from "@/components/ui/Input";
import { TodoDraft } from "@/features/todos/types/todo";
import { todoSchema } from "@/features/todos/utils/todoValidation";

type TodoFormProps = {
    mode: "create" | "edit";
    submitLabel: string;
    initialValue?: TodoDraft;
    onSubmit: (input: TodoDraft) => Promise<void>;
};

const defaultValue: TodoDraft = {
    title: "",
    description: "",
    completed: false,
};

function isSameDraft(left: TodoDraft, right: TodoDraft) {
    return (
        left.title === right.title &&
        left.description === right.description &&
        left.completed === right.completed
    );
}

export function TodoForm({ mode, submitLabel, initialValue = defaultValue, onSubmit }: TodoFormProps) {
    const [submitError, setSubmitError] = useState("");
    const previousInitialValueRef = useRef(initialValue);
    const {
        control,
        reset,
        handleSubmit,
        formState: { errors, isSubmitting },
    } = useForm<TodoDraft>({
        resolver: zodResolver(todoSchema),
        defaultValues: initialValue,
    });

    useEffect(() => {
        if (isSameDraft(previousInitialValueRef.current, initialValue)) {
            return;
        }

        previousInitialValueRef.current = initialValue;
        reset(initialValue);
    }, [initialValue, reset]);

    const onValidSubmit = handleSubmit(async (value) => {
        setSubmitError("");

        try {
            await onSubmit(value);

            if (mode === "create") {
                reset(defaultValue);
            }
        } catch (error) {
            setSubmitError(error instanceof Error ? error.message : "保存に失敗しました。");
        }
    });

    return (
        <form className="space-y-5" onSubmit={onValidSubmit}>
            <Field label="タイトル" htmlFor="title" error={errors.title?.message}>
                <Controller
                    name="title"
                    control={control}
                    render={({ field }) => (
                        <Input
                            {...field}
                            id="title"
                            placeholder="例: 週次レポートを提出する"
                        />
                    )}
                />
            </Field>

            <Field label="説明" htmlFor="description" error={errors.description?.message}>
                <Controller
                    name="description"
                    control={control}
                    render={({ field }) => (
                        <textarea
                            {...field}
                            id="description"
                            rows={5}
                            className="w-full rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm text-slate-950 outline-none transition placeholder:text-slate-400 focus:border-sky-400 focus:ring-4 focus:ring-sky-100"
                            placeholder="詳細なメモや手順を入力"
                        />
                    )}
                />
            </Field>

            <Controller
                name="completed"
                control={control}
                render={({ field }) => (
                    <label className="flex items-center gap-3 rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-700">
                        <input
                            type="checkbox"
                            checked={field.value}
                            onBlur={field.onBlur}
                            onChange={(event) => field.onChange(event.target.checked)}
                            name={field.name}
                            ref={field.ref}
                        />
                        完了済みとして保存する
                    </label>
                )}
            />

            {submitError ? <p className="text-sm text-rose-600">{submitError}</p> : null}

            <Button disabled={isSubmitting} type="submit">
                {isSubmitting ? "保存中..." : submitLabel}
            </Button>
        </form>
    );
}