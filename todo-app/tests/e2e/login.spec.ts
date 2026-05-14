import { expect, test } from "@playwright/test";

test("login flow redirects to todo dashboard", async ({ page, request }) => {
  const email = `playwright-login-${Date.now()}-${Math.random().toString(36).slice(2, 8)}@example.com`;
  const password = "password123";

  const signupResponse = await request.post("/api/auth/signup", {
    data: {
      name: "Playwright Login User",
      email,
      password,
    },
  });

  expect(signupResponse.ok()).toBe(true);

  await page.goto("/login");
  await expect(page.getByRole("heading", { name: "アカウントにアクセス" })).toBeVisible();

  await page.getByLabel("メールアドレス").fill(email);
  await page.getByLabel("パスワード").fill(password);
  await page.getByRole("button", { name: "ログイン" }).click();

  await expect(page).toHaveURL(/\/todos$/);
  await expect
    .poll(async () => {
      const response = await page.context().request.get("/api/auth/session");
      return response.ok();
    })
    .toBe(true);
});