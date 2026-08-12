import { useAuthSession } from "./use-auth-session";

export function useAuthenticatedUserId() {
  const session = useAuthSession();
  if (!session) throw new Error("Authentication is required.");
  return session.user.id;
}
