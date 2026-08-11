import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, useRouterState } from "@tanstack/react-router";
import { ChevronDown, CreditCard, LogOut, Settings } from "lucide-react";
import { useHoverDropdown } from "./use-hover-dropdown";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  accountProfileUpdatedEvent,
  defaultAccountProfile,
  readAccountProfile,
  type AccountProfile,
} from "@/model/account";

function isActivePath(pathname: string, to: string) {
  return to === "/"
    ? pathname === "/"
    : pathname === to || pathname.startsWith(`${to}/`);
}

function AccountMenu() {
  const { t } = useTranslation("navigation");
  const [profile, setProfile] = useState<AccountProfile>(defaultAccountProfile);
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  });
  const {
    closeFromHover,
    isOpen,
    onOpenChange,
    openFromHover,
    releaseMenu,
    togglePinned,
  } = useHoverDropdown();
  const isSettingsActive =
    pathname === "/settings" || pathname.startsWith("/settings/");

  useEffect(() => {
    const syncProfile = () => setProfile(readAccountProfile());
    syncProfile();
    window.addEventListener(accountProfileUpdatedEvent, syncProfile);
    window.addEventListener("storage", syncProfile);
    return () => {
      window.removeEventListener(accountProfileUpdatedEvent, syncProfile);
      window.removeEventListener("storage", syncProfile);
    };
  }, []);

  return (
    <DropdownMenu open={isOpen} onOpenChange={onOpenChange}>
      <DropdownMenuTrigger
        onMouseEnter={openFromHover}
        onMouseLeave={closeFromHover}
        onClick={togglePinned}
        className={cn(
          "inline-flex h-9 items-center gap-1 rounded-md px-1.5 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none sm:px-3 sm:text-sm",
          (isOpen || isSettingsActive) &&
            "bg-foreground text-background hover:bg-foreground hover:text-background",
        )}
      >
        {t("account")}{" "}
        <ChevronDown
          className={cn(
            "size-3.5 transition-transform",
            isOpen && "rotate-180",
          )}
        />
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        sideOffset={8}
        className="w-72 p-2"
        onMouseEnter={openFromHover}
        onMouseLeave={closeFromHover}
      >
        <DropdownMenuGroup>
          <DropdownMenuLabel className="px-2 py-2">
            <div className="flex items-center gap-3">
              <div className="grid size-9 place-items-center overflow-hidden rounded-lg border border-border bg-transparent">
                <img
                  src={profile.avatarUrl ?? "/setting/images.jpg"}
                  alt={t("profileImageAlt")}
                  className="size-full object-cover"
                />
              </div>
              <div className="min-w-0">
                <p className="truncate text-sm font-semibold text-popover-foreground">
                  {profile.name}
                </p>
                <p className="truncate text-xs text-muted-foreground">
                  {profile.email}
                </p>
              </div>
            </div>
          </DropdownMenuLabel>
        </DropdownMenuGroup>
        <div className="grid grid-cols-2 gap-2 px-2 py-2">
          <div className="rounded-lg border bg-muted/50 p-2">
            <p className="text-xs text-muted-foreground">{t("credits")}</p>
            <p className="mt-1 text-lg font-semibold">1,280</p>
          </div>
          <div className="rounded-lg border bg-muted/50 p-2">
            <p className="text-xs text-muted-foreground">{t("plan")}</p>
            <Badge variant="secondary" className="mt-1">
              {t("starter")}
            </Badge>
          </div>
        </div>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuItem
            render={<Link to="/settings" onClick={releaseMenu} />}
          >
            <Settings />
            {t("settings")}
          </DropdownMenuItem>
          <DropdownMenuItem>
            <CreditCard />
            {t("billing")}
          </DropdownMenuItem>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuItem variant="destructive" onClick={releaseMenu}>
          <LogOut />
          {t("logout")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function AppHeader() {
  const { t } = useTranslation("navigation");
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  });

  return (
    <header className="sticky top-0 z-50 border-b bg-background/90 backdrop-blur">
      <nav
        aria-label="Main navigation"
        className="grid min-h-14 grid-cols-[1fr_auto_1fr] items-center gap-2 px-3 sm:px-6"
      >
        <div className="flex min-w-0 items-center">
          <Link
            to="/"
            aria-label="Holonic Asset home"
            className="inline-flex shrink-0 rounded-sm focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
          >
            <img
              src="/logos/logo-light-nobackground.svg"
              alt=""
              className="size-5 sm:hidden"
            />
            <img
              src="/logos/logo-light-with-name.svg"
              alt=""
              className="hidden h-6 w-auto sm:block"
            />
          </Link>
        </div>
        <div className="flex items-center justify-center gap-1 sm:gap-2">
          {[
            ["/", "home"],
            ["/generate", "image"],
            ["/projects", "project"],
            ["/docs", "docs"],
          ].map(([to, key]) => {
            const active = isActivePath(pathname, to);
            return (
              <Link
                key={to}
                to={to}
                aria-current={active ? "page" : undefined}
                className={cn(
                  "inline-flex h-9 items-center rounded-md px-1.5 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none sm:px-3 sm:text-sm",
                  active &&
                    "bg-foreground text-background hover:bg-foreground hover:text-background",
                )}
              >
                {t(key)}
              </Link>
            );
          })}
        </div>
        <div className="flex justify-end">
          <AccountMenu />
        </div>
      </nav>
    </header>
  );
}
