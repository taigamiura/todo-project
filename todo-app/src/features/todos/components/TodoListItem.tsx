import Link from "next/link";
import { Todo } from "@/features/todos/types/todo";
import { ROUTES } from "@/lib/constants/routes";

type TodoListItemProps = {
    todo: Todo;
    onToggle: (id: string) => Promise<void>;
};

export function TodoListItem({ todo, onToggle }: TodoListItemProps) {
    return (
        <article className="rounded-[24px] border border-slate-200 bg-slate-50 p-5 transition hover:border-slate-300 hover:bg-white">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-3">
                        <span
                            className={[
                                "inline-flex rounded-full px-3 py-1 text-xs font-semibold",
                                todo.completed ? "bg-emerald-100 text-emerald-700" : "bg-amber-100 text-amber-700",
                            ].join(" ")}
                        >
                            {todo.completed ? "完了" : "進行中"}
                        </span>
                        <p className="text-xs text-slate-500">更新: {new Date(todo.updatedAt).toLocaleDateString("ja-JP")}</p>
                    </div>

                    <h3 className="mt-3 text-xl font-semibold text-slate-950">{todo.title}</h3>
                    <p className="mt-2 line-clamp-2 text-sm leading-7 text-slate-600">{todo.description || "説明は未入力です。"}</p>
                </div>

                <div className="flex shrink-0 items-center gap-3">
                    <button
                        type="button"
                        onClick={() => onToggle(todo.id)}
                        className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-medium text-slate-700 transition hover:border-slate-300 hover:text-slate-950"
                    >
                        {todo.completed ? "未完了に戻す" : "完了にする"}
                    </button>
                    <Link
                        href={ROUTES.todoDetail(todo.id)}
                        className="rounded-full bg-slate-950 px-4 py-2 text-sm font-medium text-white transition hover:bg-slate-800"
                    >
                        詳細を見る
                    </Link>
                </div>
            </div>
        </article>
    );
}