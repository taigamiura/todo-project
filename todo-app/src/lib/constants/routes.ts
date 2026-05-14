export const ROUTES = {
  home: "/",
  login: "/login",
  signup: "/signup",
  todos: "/todos",
  todoDetail: (id: string) => `/todos/${id}`,
} as const;