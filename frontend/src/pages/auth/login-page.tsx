import { useSearch } from "@tanstack/react-router";

import { Login } from "@/features/auth";

export function LoginPage() {
  const { redirect } = useSearch({ from: "/login" });

  return <Login redirectTo={redirect} />;
}
