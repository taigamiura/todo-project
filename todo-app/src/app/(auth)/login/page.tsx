import Link from "next/link";
import { redirect } from "next/navigation";
import { LoginForm } from "@/features/auth/components/LoginForm";
import { ROUTES } from "@/lib/constants/routes";
import { getServerSessionUser } from "@/lib/server/session";

export default async function LoginPage() {
    const user = await getServerSessionUser();

    if (user) {
        redirect(ROUTES.todos);
    }

    return (
        <main className="grid min-h-screen bg-[radial-gradient(circle_at_top_left,_rgba(14,165,233,0.18),_transparent_32%),linear-gradient(180deg,_#f8fafc_0%,_#e2e8f0_100%)] px-6 py-12">
            <div className="mx-auto grid w-full max-w-6xl gap-10 lg:grid-cols-[1.15fr_0.85fr]">
                <section className="rounded-[32px] border border-white/60 bg-slate-950 px-8 py-10 text-white shadow-[0_40px_120px_rgba(15,23,42,0.35)] sm:px-12 sm:py-14">
                    <p className="text-sm uppercase tracking-[0.28em] text-cyan-300">Todo Workspace</p>
                    <h1 className="mt-6 max-w-xl text-4xl font-semibold leading-tight sm:text-5xl">
                        会員ログインして、作業を継続できる Todo クライアント。
                    </h1>
                    <p className="mt-6 max-w-lg text-base leading-8 text-slate-300 sm:text-lg">
                        一覧、詳細、更新、削除までをひと通り備えた構成です。認証は cookie セッション、データ操作は API 経由に寄せています。
                    </p>
                    <div className="mt-10 grid gap-4 sm:grid-cols-3">
                        <div className="rounded-2xl border border-white/10 bg-white/5 p-5">
                            <p className="text-sm text-slate-300">認証導線</p>
                            <p className="mt-2 text-lg font-medium">ログイン / 登録</p>
                        </div>
                        <div className="rounded-2xl border border-white/10 bg-white/5 p-5">
                            <p className="text-sm text-slate-300">一覧画面</p>
                            <p className="mt-2 text-lg font-medium">新規作成と状態確認</p>
                        </div>
                        <div className="rounded-2xl border border-white/10 bg-white/5 p-5">
                            <p className="text-sm text-slate-300">詳細画面</p>
                            <p className="mt-2 text-lg font-medium">編集と削除</p>
                        </div>
                    </div>
                </section>

                <section className="flex items-center justify-center rounded-[32px] border border-black/5 bg-white/90 p-6 shadow-[0_24px_80px_rgba(15,23,42,0.08)] backdrop-blur sm:p-8">
                    <div className="w-full max-w-md">
                        <div className="mb-8">
                            <p className="text-sm font-medium text-sky-700">ログイン</p>
                            <h2 className="mt-2 text-3xl font-semibold text-slate-950">アカウントにアクセス</h2>
                            <p className="mt-3 text-sm leading-7 text-slate-600">
                                登録済みユーザーでログインしてください。まだ登録していない場合は新規作成へ進みます。
                            </p>
                        </div>
                        <LoginForm />
                        <p className="mt-6 text-sm text-slate-600">
                            アカウント未作成の場合は
                            <Link className="ml-1 font-semibold text-sky-700 transition hover:text-sky-900" href={ROUTES.signup}>
                                新規登録
                            </Link>
                        </p>
                    </div>
                </section>
            </div>
        </main>
    );
}