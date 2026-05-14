import { expect, test } from "@playwright/test";

test("delete flow removes a todo from the list", async ({ page }) => {
  const email = `playwright-delete-${Date.now()}-${Math.random().toString(36).slice(2, 8)}@example.com`;
  const todoTitle = `Todo Delete ${Date.now()}`;

  await page.goto("/signup");
  await page.getByLabel("名前").fill("Playwright Delete User");
  await page.getByLabel("メールアドレス").fill(email);
  await page.getByLabel("パスワード").fill("password123");
  await page.getByRole("button", { name: "会員登録" }).click();

  await expect(page).toHaveURL(/\/todos$/);

  await expect
    .poll(async () => {
      const response = await page.context().request.get("/api/auth/session");
      return response.ok();
    })
    .toBe(true);

  await page.getByRole("button", { name: "作成する" }).click();
  await page.getByLabel("タイトル").fill(todoTitle);
  await page.getByLabel("説明").fill("Delete me");
  const createResponsePromise = page.waitForResponse(
    (response) => response.url().endsWith("/api/todos") && response.request().method() === "POST",
  );
  await page.getByRole("button", { name: "Todoを作成" }).click();
  const createResponse = await createResponsePromise;
  const createdTodoId = ((await createResponse.json()) as { todo: { id: string } }).todo.id;

  await expect
    .poll(async () => {
      const response = await page.context().request.get(`/api/todos/${createdTodoId}`);
      return response.ok();
    })
    .toBe(true);

  const deleteResponse = await page.context().request.delete(`/api/todos/${createdTodoId}`);

  expect(deleteResponse.status()).toBe(204);
  await page.reload();
  await expect(page.getByText(todoTitle)).toHaveCount(0);
});