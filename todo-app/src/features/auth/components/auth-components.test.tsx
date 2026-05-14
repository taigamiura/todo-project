import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { LoginForm } from "@/features/auth/components/LoginForm";
import { LogoutButton } from "@/features/auth/components/LogoutButton";
import { SignupForm } from "@/features/auth/components/SignupForm";

const replace = vi.fn();
const login = vi.fn();
const signup = vi.fn();
const logout = vi.fn();

vi.mock("next/navigation", () => ({
    useRouter: () => ({ replace }),
}));

vi.mock("@/features/auth/hooks/useAuth", () => ({
    useAuth: () => ({ user: null, loading: false, login, signup, logout }),
}));

describe("auth components", () => {
    it("submits login and signup forms and shows validation/errors", async () => {
        login.mockResolvedValue({});
        signup.mockRejectedValue(new Error("登録失敗"));
        const user = userEvent.setup();

        render(<LoginForm />);
        await user.click(screen.getByRole("button", { name: "ログイン" }));
        expect(await screen.findByText("メールアドレスの形式が不正です。")).toBeInTheDocument();

        await user.type(screen.getByPlaceholderText("name@example.com"), "a@example.com");
        await user.type(screen.getByPlaceholderText("8文字以上で入力"), "password123");
        await user.click(screen.getByRole("button", { name: "ログイン" }));
        await waitFor(() => expect(login).toHaveBeenCalled());
        expect(replace).toHaveBeenCalled();

        render(<SignupForm />);
        await user.type(screen.getByPlaceholderText("山田 花子"), "Hanako");
        await user.type(screen.getAllByPlaceholderText("name@example.com")[1], "b@example.com");
        await user.type(screen.getAllByPlaceholderText("8文字以上で入力")[1], "password123");
        await user.click(screen.getByRole("button", { name: "会員登録" }));
        expect(await screen.findByText("登録失敗")).toBeInTheDocument();
    });

    it("uses fallback submit errors and success redirects", async () => {
        login.mockRejectedValue("failed");
        signup.mockResolvedValue({});
        const user = userEvent.setup();

        render(<LoginForm />);
        await user.type(screen.getByPlaceholderText("name@example.com"), "a@example.com");
        await user.type(screen.getByPlaceholderText("8文字以上で入力"), "password123");
        await user.click(screen.getByRole("button", { name: "ログイン" }));
        expect(await screen.findByText("ログインに失敗しました。")).toBeInTheDocument();

        render(<SignupForm />);
        await user.type(screen.getByPlaceholderText("山田 花子"), "Hanako");
        await user.type(screen.getAllByPlaceholderText("name@example.com")[1], "b@example.com");
        await user.type(screen.getAllByPlaceholderText("8文字以上で入力")[1], "password123");
        await user.click(screen.getByRole("button", { name: "会員登録" }));
        await waitFor(() => expect(signup).toHaveBeenCalled());
    });

    it("logs out and redirects", async () => {
        logout.mockResolvedValue(undefined);
        const user = userEvent.setup();
        render(<LogoutButton />);
        await user.click(screen.getByRole("button", { name: "ログアウト" }));
        await waitFor(() => expect(logout).toHaveBeenCalled());
        expect(replace).toHaveBeenCalled();
    });
});