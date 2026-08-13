import { LoginWorkspace } from "./login";
import { useLoginController } from "./use-login-controller";

export function Login({ redirectTo }: { redirectTo?: string }) {
  const login = useLoginController(redirectTo);

  return <LoginWorkspace login={login} />;
}
