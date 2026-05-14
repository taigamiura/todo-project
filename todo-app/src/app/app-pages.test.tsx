import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderAsyncComponent } from "@/test/test-helpers";

const redirect = vi.fn((value: string) => {
    throw new Error(`redirect:${value}`);
});
const getServerSessionUser = vi.fn();
const authState = {
    user: { id: "u1", name: "Hanako", email: "a@example.com" },
    loading: false,
};
const todosState = {
    todos: [] as Array<{ id: string; title: string; description: string; completed: boolean; createdAt: string; updatedAt: string }>,
    loading: false,
    createTodo: vi.fn(),
    toggleTodo: vi.fn(),
};
const todoDetailState = {
    todo: { id: "t1", title: "Todo", description: "desc", completed: false, createdAt: "2024-01-01T00:00:00.000Z", updatedAt: "2024-01-01T00:00:00.000Z" },
    loading: false,
    updateTodo: vi.fn(),
    deleteTodo: vi.fn(),
};

vi.mock("next/navigation", () => ({ redirect }));
vi.mock("@/lib/server/session", () => ({ getServerSessionUser }));
vi.mock("@/features/auth/components/LoginForm", () => ({ LoginForm: () => <div>LoginForm</div> }));
vi.mock("@/features/auth/components/SignupForm", () => ({ SignupForm: () => <div>SignupForm</div> }));
vi.mock("@/features/auth/components/LogoutButton", () => ({ LogoutButton: () => <button>Logout</button> }));
vi.mock("@/features/auth/hooks/useAuth", () => ({
    useAuth: () => authState,
}));
vi.mock("@/features/todos/hooks/useTodos", () => ({
    useTodos: () => todosState,
}));
vi.mock("@/features/todos/hooks/useTodoDetail", () => ({
    useTodoDetail: () => todoDetailState,
}));
vi.mock("next/navigation", async () => {
    const actual = await vi.importActual<typeof import("next/navigation")>("next/navigation");
    return { ...actual, redirect, useParams: () => ({ id: "t1" }), useRouter: () => ({ replace: vi.fn() }) };
});

describe("app pages", () => {
    it("redirects home and protected layout based on session", async () => {
        getServerSessionUser.mockResolvedValueOnce(null).mockResolvedValueOnce(null).mockResolvedValueOnce({ id: "u1" });
        const Home = (await import("@/app/page")).default;
        const AppLayout = (await import("@/app/(app)/layout")).default;

        await expect(Home()).rejects.toThrow("redirect:/login");
        await expect(AppLayout({ children: <div>child</div> })).rejects.toThrow("redirect:/login");
        await expect(Home()).rejects.toThrow("redirect:/todos");
    });

    it("renders auth pages when unauthenticated and redirects when authenticated", async () => {
        getServerSessionUser.mockResolvedValueOnce(null).mockResolvedValueOnce(null).mockResolvedValueOnce({ id: "u1" }).mockResolvedValueOnce({ id: "u1" });
        const LoginPage = (await import("@/app/(auth)/login/page")).default;
        const SignupPage = (await import("@/app/(auth)/signup/page")).default;

        await renderAsyncComponent(LoginPage);
        expect(screen.getByText("LoginForm")).toBeInTheDocument();

        await renderAsyncComponent(SignupPage);
        expect(screen.getByText("SignupForm")).toBeInTheDocument();

        const again = (await import("@/app/(auth)/login/page")).default;
        await expect(again()).rejects.toThrow("redirect:/todos");

        const againSignup = (await import("@/app/(auth)/signup/page")).default;
        await expect(againSignup()).rejects.toThrow("redirect:/todos");
    });

    it("renders root layout and client todo pages", async () => {
        const RootLayout = (await import("@/app/layout")).default;
        const TodosPage = (await import("@/app/(app)/todos/page")).default;
        const TodoDetailPage = (await import("@/app/(app)/todos/[id]/page")).default;

        render(<RootLayout><div>child</div></RootLayout>);
        expect(screen.getByText("child")).toBeInTheDocument();

        authState.loading = true;
        render(<TodosPage />);
        expect(screen.getByText("ユーザー情報を確認しています...")).toBeInTheDocument();

        authState.loading = false;
        todosState.loading = true;
        render(<TodosPage />);
        expect(screen.getByText("読み込み中")).toBeInTheDocument();

        todosState.loading = false;
        expect(screen.getByText("Todo がまだありません")).toBeInTheDocument();

        todosState.todos = [todoDetailState.todo];
        render(<TodosPage />);
        expect(screen.getByText("1 件")).toBeInTheDocument();

        todoDetailState.loading = true;
        render(<TodoDetailPage />);
        expect(screen.getByText("詳細データを読み込んでいます...")).toBeInTheDocument();

        todoDetailState.loading = false;
        todoDetailState.todo = null as never;
        render(<TodoDetailPage />);
        expect(screen.getByText("Todo が見つかりません")).toBeInTheDocument();

        todoDetailState.todo = { id: "t1", title: "Todo", description: "desc", completed: false, createdAt: "2024-01-01T00:00:00.000Z", updatedAt: "2024-01-01T00:00:00.000Z" };
        render(<TodoDetailPage />);
        expect(screen.getByText("Todo を編集")).toBeInTheDocument();
    });
});