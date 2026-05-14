import { z } from "zod";

export const loginSchema = z.object({
  email: z.email("メールアドレスの形式が不正です。").trim(),
  password: z.string().trim().min(1, "パスワードは必須です。"),
});

export const signupSchema = z.object({
  name: z.string().trim().min(1, "名前は必須です。"),
  email: z.email("メールアドレスの形式が不正です。").trim(),
  password: z.string().trim().min(8, "パスワードは8文字以上で入力してください。"),
});