import { expect, test } from "@playwright/test";

test("update todo flow from the list", async ({ page }) => {
  const email = `playwright-update-${Date.now()}-${Math.random().toString(36).slice(2, 8)}@example.com`;
  const createdTitle = `Todo Update ${Date.now()}`;
  const updatedTitle = `${createdTitle} Updated`;
  const createdDescription = "Created from Playwright E2E";
  const updatedDescription = "Updated from Playwright E2E";

  await page.goto("/signup");
  await page.getByLabel("名前").fill("Playwright Update User");
  await page.getByLabel("メールアドレス").fill(email);
  await page.getByLabel("パスワード").fill("password123");
  await page.getByRole("button", { name: "会員登録" }).click();

  await expect(page).toHaveURL(/\/todos$/);
  await page.getByRole("button", { name: "作成する" }).click();
  await page.getByLabel("タイトル").fill(createdTitle);
  await page.getByLabel("説明").fill(createdDescription);
  const createResponsePromise = page.waitForResponse(
    (response) => response.url().endsWith("/api/todos") && response.request().method() === "POST",
  );
  await page.getByRole("button", { name: "Todoを作成" }).click();
  const createResponse = await createResponsePromise;
  const createdTodoId = ((await createResponse.json()) as { todo: { id: string } }).todo.id;

  await expect(page.getByText(createdTitle)).toBeVisible();

  const updateResponse = await page.context().request.patch(`/api/todos/${createdTodoId}`, {
    data: {
      title: updatedTitle,
      description: updatedDescription,
      completed: true,
    },
  });

  expect(updateResponse.ok()).toBe(true);
  const updatedTodo = ((await updateResponse.json()) as {
    todo: { title: string; description: string; completed: boolean };
  }).todo;

  expect(updatedTodo.title).toBe(updatedTitle);
  expect(updatedTodo.description).toBe(updatedDescription);
  expect(updatedTodo.completed).toBe(true);

  await expect
    .poll(async () => {
      const response = await page.context().request.get(`/api/todos/${createdTodoId}`);
      if (!response.ok()) {
        return null;
      }

      return (await response.json()) as {
        todo: { title: string; description: string; completed: boolean };
      };
    })
    .toMatchObject({
      todo: {
        title: updatedTitle,
        description: updatedDescription,
        completed: true,
      },
    });
});