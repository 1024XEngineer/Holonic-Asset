import { Eye, EyeOff, LoaderCircle } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

import type { LoginController } from "./use-login-controller";

export function LoginForm({ login }: { login: LoginController }) {
  return (
    <form
      className="mt-8 grid gap-5"
      noValidate
      onSubmit={(event) => {
        event.preventDefault();
        void login.form.handleSubmit();
      }}
    >
      <UsernameField login={login} />
      <PasswordField login={login} />
      <LoginErrorMessage login={login} />
      <LoginSubmitButton login={login} />
    </form>
  );
}

function UsernameField({ login }: { login: LoginController }) {
  const { t } = useTranslation("auth");

  return (
    <login.form.Field name="username">
      {(field) => (
        <label className="grid gap-2 text-sm font-medium">
          {t("username")}
          <Input
            autoComplete="username"
            autoFocus
            className="h-11 rounded-md border-black/20 bg-white px-3 dark:border-white/20 dark:bg-white/5"
            maxLength={64}
            name="username"
            required
            value={field.state.value}
            onBlur={field.handleBlur}
            onChange={(event) => {
              login.clearSubmitError();
              field.handleChange(event.target.value);
            }}
          />
        </label>
      )}
    </login.form.Field>
  );
}

function PasswordField({ login }: { login: LoginController }) {
  const { t } = useTranslation("auth");

  return (
    <login.form.Field name="password">
      {(field) => (
        <label className="grid gap-2 text-sm font-medium">
          {t("password")}
          <span className="relative">
            <Input
              autoComplete="current-password"
              className="h-11 rounded-md border-black/20 bg-white px-3 pr-11 dark:border-white/20 dark:bg-white/5"
              maxLength={72}
              name="password"
              required
              type={login.showPassword ? "text" : "password"}
              value={field.state.value}
              onBlur={field.handleBlur}
              onChange={(event) => {
                login.clearSubmitError();
                field.handleChange(event.target.value);
              }}
            />
            <PasswordVisibilityButton login={login} />
          </span>
        </label>
      )}
    </login.form.Field>
  );
}

function PasswordVisibilityButton({ login }: { login: LoginController }) {
  const { t } = useTranslation("auth");
  const label = login.showPassword ? t("hidePassword") : t("showPassword");

  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      className="absolute top-1/2 right-1 grid size-9 -translate-y-1/2 place-items-center rounded-md text-black/48 transition-colors hover:bg-black/5 hover:text-black focus-visible:ring-3 focus-visible:ring-black/20 focus-visible:outline-none dark:text-white/48 dark:hover:bg-white/10 dark:hover:text-white dark:focus-visible:ring-white/25"
      onClick={login.togglePassword}
    >
      {login.showPassword ? (
        <EyeOff className="size-4" />
      ) : (
        <Eye className="size-4" />
      )}
    </button>
  );
}

function LoginErrorMessage({ login }: { login: LoginController }) {
  const { t } = useTranslation("auth");

  if (!login.submitError) return null;

  return (
    <p
      role="alert"
      className="border-l-2 border-destructive bg-destructive/8 px-3 py-2.5 text-sm text-destructive"
    >
      {t(login.submitError)}
    </p>
  );
}

function LoginSubmitButton({ login }: { login: LoginController }) {
  const { t } = useTranslation("auth");

  return (
    <login.form.Subscribe
      selector={(state) => ({
        canSubmit:
          state.values.username.trim().length > 0 &&
          state.values.password.length > 0,
        isSubmitting: state.isSubmitting,
      })}
    >
      {({ canSubmit, isSubmitting }) => (
        <Button
          type="submit"
          className="mt-1 h-11 rounded-md bg-[#171814] text-white hover:bg-[#30322c] dark:bg-[#f4f4ee] dark:text-[#171814] dark:hover:bg-white"
          disabled={!canSubmit || isSubmitting}
        >
          {isSubmitting ? <LoaderCircle className="animate-spin" /> : null}
          {isSubmitting ? t("submitting") : t("submit")}
        </Button>
      )}
    </login.form.Subscribe>
  );
}
