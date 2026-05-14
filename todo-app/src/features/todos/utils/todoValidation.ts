import { z } from "zod";

export const todoSchema = z.object({
  title: z.string().trim().min(1, "タイトルは必須です。").max(50, "タイトルは50文字以内で入力してください。"),
  description: z.string().trim().max(300, "説明は300文字以内で入力してください。"),
  completed: z.boolean(),
});