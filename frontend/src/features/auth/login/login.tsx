import { LoginArtwork } from "./login-artwork";
import { LoginFormPanel } from "./login-form-panel";
import type { LoginController } from "./use-login-controller";

export function LoginWorkspace({ login }: { login: LoginController }) {
  return (
    <main className="min-h-screen bg-[#f4f3ef] text-[#171814] lg:grid lg:grid-cols-[minmax(0,1.08fr)_minmax(30rem,0.92fr)] dark:bg-[#111310] dark:text-[#f4f4ee]">
      <LoginArtwork />
      <LoginFormPanel login={login} />
    </main>
  );
}
