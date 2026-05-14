import Link from "next/link";
import { redirect } from "next/navigation";
import { SignupForm } from "@/features/auth/components/SignupForm";
import { ROUTES } from "@/lib/constants/routes";
import { getServerSessionUser } from "@/lib/server/session";

export default async function SignupPage() {
    const user = await getServerSessionUser();

    if (user) {
        redirect(ROUTES.todos);
    }

    return (
        <main className="grid min-h-screen bg-[radial-gradient(circle_at_top_right,_rgba(244,114,182,0.16),_transparent_28%),linear-gradient(180deg,_#fff7ed_0%,_#f8fafc_100%)] px-6 py-12">
            <div className="mx-auto grid w-full max-w-6xl gap-10 lg:grid-cols-[0.9fr_1.1fr]">
                <section className="flex items-center justify-center rounded-[32px] border border-black/5 bg-white/95 p-6 shadow-[0_24px_80px_rgba(15,23,42,0.08)] sm:p-8">
                    <div className="w-full max-w-md">
                        <div className="mb-8">
                            <p className="text-sm font-medium text-rose-700">新規登録</p>
                            <h1 className="mt-2 text-3xl font-semibold text-slate-950">学習用アカウントを作成</h1>
                            <p className="mt-3 text-sm leading-7 text-slate-600">
                                名前、メールアドレス、パスワードを登録して Todo 管理画面へ進みます。
                            </p>
                        </div>
                        <SignupForm />
                        <p className="mt-6 text-sm text-slate-600">
                            すでに登録済みの場合は
                            <Link className="ml-1 font-semibold text-rose-700 transition hover:text-rose-900" href={ROUTES.login}>
                                ログイン
                            </Link>
                        </p>
                    </div>
                </section>

                <section className="rounded-[32px] border border-white/60 bg-rose-950 px-8 py-10 text-white shadow-[0_40px_120px_rgba(15,23,42,0.35)] sm:px-12 sm:py-14">
                    <p className="text-sm uppercase tracking-[0.28em] text-rose-200">Practical Setup</p>
                    <h2 className="mt-6 max-w-xl text-4xl font-semibold leading-tight sm:text-5xl">
                        BFF とサービス分離を前提に、フロントも API 境界を意識した構成へ寄せています。
                    </h2>
                    <ul className="mt-8 space-y-4 text-base leading-8 text-rose-100">
                        <li>機能単位のディレクトリ構成</li>
                        <li>zod と react-hook-form のフォーム実装</li>
                        <li>cookie セッションとサーバー側認証ガード</li>
                    </ul>
                </section>
            </div>
        </main>
    );
}