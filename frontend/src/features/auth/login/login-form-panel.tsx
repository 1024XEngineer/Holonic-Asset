import { LockKeyhole } from "lucide-react";
import { useTranslation } from "react-i18next";

import { LoginForm } from "./login-form";
import type { LoginController } from "./use-login-controller";

export function LoginFormPanel({ login }: { login: LoginController }) {
  const { t } = useTranslation("auth");

  return (
    <section className="flex items-center px-5 py-10 sm:px-12 lg:px-[clamp(3rem,7vw,7rem)]">
      <div className="mx-auto w-full max-w-md">
        <div className="grid size-10 place-items-center rounded-md border border-black/15 bg-white/55 dark:border-white/15 dark:bg-white/5">
          <LockKeyhole className="size-4" />
        </div>
        <h1 className="mt-6 text-3xl font-semibold sm:text-4xl">
          {t("title")}
        </h1>
        <p className="mt-3 text-sm leading-6 text-black/58 dark:text-white/58">
          {t("description")}
        </p>
        <LoginForm login={login} />
        <p className="mt-7 border-t border-black/12 pt-5 text-xs leading-5 text-black/48 dark:border-white/12 dark:text-white/48">
          {t("accountHelp")}
        </p>
      </div>
    </section>
  );
}
