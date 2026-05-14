import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

const authenticateUser = vi.fn();
const clearCurrentSession = vi.fn();
const createUser = vi.fn();
const getCurrentUser = vi.fn();
const subscribeAuthStorage = vi.fn();

vi.mock("@/lib/storage/authStorage", () => ({
    authenticateUser,
    clearCurrentSession,
    createUser,
    getCurrentUser,
    subscribeAuthStorage,
}));

describe("useAuth", () => {
    it("loads, refreshes and mutates user state", async () => {
        const user = { id: "u1", name: "Hanako", email: "a@example.com" };
        let callback: (() => void) | undefined;
        getCurrentUser.mockResolvedValue(user);
        subscribeAuthStorage.mockImplementation((cb: () => void) => {
            callback = cb;
            return vi.fn();
        });
        authenticateUser.mockResolvedValue(user);
        createUser.mockResolvedValue(user);

        const mod = await import("@/features/auth/hooks/useAuth");
        const { result } = renderHook(() => mod.useAuth());

        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.user).toEqual(user);

        await act(async () => {
            await result.current.login({ email: user.email, password: "password123" });
            await result.current.signup({ name: user.name, email: user.email, password: "password123" });
            await result.current.logout();
        });

        expect(clearCurrentSession).toHaveBeenCalled();
        await act(async () => {
            callback?.();
        });
    });
});