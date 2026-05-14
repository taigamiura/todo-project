"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { TodoDetailActions } from "@/features/todos/components/TodoDetailActions";
import { TodoForm } from "@/features/todos/components/TodoForm";
import { useTodoDetail } from "@/features/todos/hooks/useTodoDetail";
import { ROUTES } from "@/lib/constants/routes";

export default function TodoDetailPage() {
    const params = useParams<{ id: string }>();
    const router = useRouter();
    const { user, loading: authLoading } = useAuth();
    const { todo, loading, updateTodo, deleteTodo } = useTodoDetail(user?.id, params.id);

    if (authLoading || !user || loading) {
        return (
            <main className="flex min-h-screen items-center justify-center bg-[var(--color-surface)] px-6 py-10">
                <Card className="max-w-md text-center text-sm text-slate-600">詳細データを読み込んでいます...</Card>
            </main>
        );
    }

    if (!todo) {
        return (
            <main className="flex min-h-screen items-center justify-center bg-[var(--color-surface)] px-6 py-10">
                <Card className="max-w-lg text-center">
                    <h1 className="text-2xl font-semibold text-slate-950">Todo が見つかりません</h1>
                    <p className="mt-4 text-sm leading-7 text-slate-600">
                        削除済み、または不正な ID の可能性があります。一覧に戻って再度選択してください。
                    </p>
                    <Link
                        href={ROUTES.todos}
                        className="mt-6 inline-flex rounded-full bg-slate-950 px-5 py-3 text-sm font-semibold text-white transition hover:bg-slate-800"
                    >
                        一覧へ戻る
                    </Link>
                </Card>
            </main>
        );
    }

    return (
        <main className="min-h-screen bg-[var(--color-surface)] px-6 py-8">
            <div className="mx-auto flex w-full max-w-4xl flex-col gap-8">
                <PageHeader
                    eyebrow="Todo Detail"
                    title={todo.title}
                    description="詳細画面から更新と削除を行えます。作成日時や更新日時も確認できます。"
                    actions={
                        <Link
                            href={ROUTES.todos}
                            className="rounded-full border border-slate-200 bg-white px-5 py-3 text-sm font-semibold text-slate-700 transition hover:border-slate-300 hover:text-slate-950"
                        >
                            一覧へ戻る
                        </Link>
                    }
                />

                <section className="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
                    <Card>
                        <p className="text-sm font-medium text-sky-700">更新</p>
                        <h2 className="mt-1 text-2xl font-semibold text-slate-950">Todo を編集</h2>
                        <div className="mt-6">
                            <TodoForm
                                key={`${todo.id}:${todo.updatedAt}`}
                                mode="edit"
                                initialValue={{
                                    title: todo.title,
                                    description: todo.description,
                                    completed: todo.completed,
                                }}
                                submitLabel="変更を保存"
                                onSubmit={async (input) => {
                                    await updateTodo(input);
                                }}
                            />
                        </div>
                    </Card>

                    <Card>
                        <p className="text-sm font-medium text-slate-500">詳細情報</p>
                        <dl className="mt-6 space-y-4 text-sm text-slate-600">
                            <div>
                                <dt className="font-medium text-slate-950">状態</dt>
                                <dd className="mt-1">{todo.completed ? "完了" : "未完了"}</dd>
                            </div>
                            <div>
                                <dt className="font-medium text-slate-950">作成日時</dt>
                                <dd className="mt-1">{new Date(todo.createdAt).toLocaleString("ja-JP")}</dd>
                            </div>
                            <div>
                                <dt className="font-medium text-slate-950">更新日時</dt>
                                <dd className="mt-1">{new Date(todo.updatedAt).toLocaleString("ja-JP")}</dd>
                            </div>
                        </dl>

                        <div className="mt-8 border-t border-slate-200 pt-6">
                            <TodoDetailActions
                                onDelete={async () => {
                                    await deleteTodo();
                                    router.replace(ROUTES.todos);
                                }}
                            />
                        </div>
                    </Card>
                </section>
            </div>
        </main>
    );
}