import { useRef, useState, type ReactNode } from "react";
import {
  Camera,
  Check,
  CreditCard,
  IdCard,
  Monitor,
  Moon,
  Sun,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { AppHeader } from "@/components/layouts/app-header";
import { cn } from "@/lib/utils";
import { useTimeout } from "@/hooks/use-timeout";
import { readAccountProfile, saveAccountProfile } from "@/lib/account-profile";
import {
  readThemePreference,
  saveThemePreference,
  type ThemePreference,
} from "@/lib/theme-preference";

function isValidEmail(email: string) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

export function Settings() {
  const [profile] = useState(readAccountProfile);
  const [name, setName] = useState(profile.name);
  const [email, setEmail] = useState(profile.email);
  const [emailTouched, setEmailTouched] = useState(false);
  const [avatarUrl, setAvatarUrl] = useState<string>();
  const [profileSaved, setProfileSaved] = useState(false);
  const [theme, setTheme] = useState<ThemePreference>(readThemePreference);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { schedule } = useTimeout();

  return (
    <div className="min-h-screen bg-muted/30 text-foreground">
      <AppHeader />
      <main className="w-full px-4 py-8 sm:px-6 sm:py-10">
        <div className="mx-auto max-w-4xl space-y-5">
          <SettingsSection
            description="The details shown across your Holonic Asset workspace."
            icon={<IdCard />}
            title="Profile"
          >
            <form
              className="space-y-5"
              noValidate
              onSubmit={(event) => {
                event.preventDefault();
                setEmailTouched(true);
                if (!isValidEmail(email)) return;
                saveAccountProfile({ name: name.trim(), email, avatarUrl });
                setProfileSaved(true);
                schedule(() => setProfileSaved(false), 1800);
              }}
            >
              <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
                <div className="grid size-16 shrink-0 place-items-center overflow-hidden rounded-xl border border-border bg-transparent">
                  {avatarUrl ? (
                    <img
                      src={avatarUrl}
                      alt="Profile"
                      className="size-full object-cover"
                    />
                  ) : (
                    <img
                      src="/setting/images.jpg"
                      alt="Profile"
                      className="size-full object-cover"
                    />
                  )}
                </div>
                <input
                  ref={fileInputRef}
                  accept="image/jpeg,image/png,image/webp"
                  className="sr-only"
                  type="file"
                  onChange={(event) => {
                    const file = event.target.files?.[0];
                    if (!file) return;
                    const reader = new FileReader();
                    reader.addEventListener("load", () => {
                      if (typeof reader.result === "string") {
                        setAvatarUrl(reader.result);
                        setProfileSaved(false);
                      }
                    });
                    reader.readAsDataURL(file);
                  }}
                />
                <Button
                  type="button"
                  variant="outline"
                  className="sm:ml-auto"
                  onClick={() => fileInputRef.current?.click()}
                >
                  <Camera />
                  Change photo
                </Button>
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                <Field label="Display name" htmlFor="settings-display-name">
                  <Input
                    id="settings-display-name"
                    value={name}
                    onChange={(event) => {
                      setName(event.target.value);
                      setProfileSaved(false);
                    }}
                  />
                </Field>
                <Field label="Email" htmlFor="settings-email">
                  <Input
                    id="settings-email"
                    aria-describedby={
                      emailTouched && !isValidEmail(email)
                        ? "settings-email-error"
                        : undefined
                    }
                    aria-invalid={emailTouched && !isValidEmail(email)}
                    type="email"
                    value={email}
                    onBlur={() => setEmailTouched(true)}
                    onChange={(event) => {
                      setEmail(event.target.value);
                      setProfileSaved(false);
                    }}
                  />
                  {emailTouched && !isValidEmail(email) ? (
                    <p
                      id="settings-email-error"
                      className="text-sm text-destructive"
                    >
                      Enter a valid email address.
                    </p>
                  ) : null}
                </Field>
              </div>

              <div className="flex flex-wrap items-center justify-between gap-3 border-t pt-4">
                <p className="text-sm text-muted-foreground" aria-live="polite">
                  {profileSaved ? "Profile saved." : ""}
                </p>
                <Button
                  type="submit"
                  disabled={!name.trim() || !isValidEmail(email)}
                >
                  {profileSaved ? <Check /> : null}
                  Save profile
                </Button>
              </div>
            </form>
          </SettingsSection>

          <SettingsSection
            description="Choose how the workspace looks and opens for you."
            icon={<Monitor />}
            title="Preferences"
          >
            <div className="space-y-1">
              <PreferenceRow
                description="Use a light, dark, or system appearance."
                label="Appearance"
              >
                <div className="flex rounded-lg border bg-muted/30 p-1">
                  <ThemeButton
                    active={theme === "light"}
                    label="Light"
                    onClick={() => {
                      setTheme("light");
                      saveThemePreference("light");
                    }}
                  >
                    <Sun />
                  </ThemeButton>
                  <ThemeButton
                    active={theme === "dark"}
                    label="Dark"
                    onClick={() => {
                      setTheme("dark");
                      saveThemePreference("dark");
                    }}
                  >
                    <Moon />
                  </ThemeButton>
                </div>
              </PreferenceRow>
            </div>
          </SettingsSection>

          <SettingsSection
            description="A quick view of the generation resources available to this account."
            icon={<CreditCard />}
            title="Plan & credits"
          >
            <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] sm:items-center">
              <div className="border-l-2 border-primary pl-3">
                <p className="text-xs font-medium tracking-[0.08em] text-muted-foreground uppercase">
                  Current plan
                </p>
                <p className="mt-1 text-lg font-semibold">Starter</p>
              </div>
              <div className="border-l-2 border-primary pl-3">
                <p className="text-xs font-medium tracking-[0.08em] text-muted-foreground uppercase">
                  Available credits
                </p>
                <p className="mt-1 text-lg font-semibold">1,280</p>
              </div>
              <Button
                type="button"
                variant="outline"
                className="sm:justify-self-end"
              >
                Manage plan
              </Button>
            </div>
          </SettingsSection>
        </div>
      </main>
    </div>
  );
}

function SettingsSection({
  children,
  description,
  icon,
  title,
}: {
  children: ReactNode;
  description: string;
  icon: ReactNode;
  title: string;
}) {
  return (
    <section className="rounded-xl border bg-background p-5 shadow-sm sm:p-6">
      <div className="flex items-center gap-3 border-b pb-5">
        <div className="grid size-8 shrink-0 place-items-center rounded-lg border bg-muted text-muted-foreground [&>svg]:size-4">
          {icon}
        </div>
        <div>
          <h2 className="text-base font-semibold">{title}</h2>
          <p className="mt-1 text-sm leading-5 text-muted-foreground">
            {description}
          </p>
        </div>
      </div>
      <div className="pt-5">{children}</div>
    </section>
  );
}

function Field({
  children,
  htmlFor,
  label,
}: {
  children: ReactNode;
  htmlFor: string;
  label: string;
}) {
  return (
    <label className="grid gap-2 text-sm font-medium" htmlFor={htmlFor}>
      {label}
      {children}
    </label>
  );
}

function PreferenceRow({
  children,
  description,
  label,
}: {
  children: ReactNode;
  description: string;
  label: string;
}) {
  return (
    <div className="flex flex-col gap-3 border-b py-4 first:pt-0 last:border-b-0 last:pb-0 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <p className="text-sm font-medium">{label}</p>
        <p className="mt-1 text-sm leading-5 text-muted-foreground">
          {description}
        </p>
      </div>
      <div className="shrink-0">{children}</div>
    </div>
  );
}

function ThemeButton({
  active,
  children,
  label,
  onClick,
}: {
  active: boolean;
  children: ReactNode;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      aria-pressed={active}
      className={cn(
        "inline-flex h-7 items-center gap-1 rounded-md px-2 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none [&>svg]:size-3.5",
        active && "bg-background text-foreground shadow-xs",
      )}
      type="button"
      onClick={onClick}
    >
      {children}
      {label}
    </button>
  );
}
