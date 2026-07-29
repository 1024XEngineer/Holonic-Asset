import { Link, useRouterState } from "@tanstack/react-router";
import {
  ChevronDown,
  CreditCard,
  LogOut,
  Settings,
  UserRound,
} from "lucide-react";
import { useHoverDropdown } from "./AppHeader.useHoverDropdown";
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

const navItems = [
  { to: "/", label: "Home" },
  { to: "/generate", label: "Image" },
  { to: "/projects", label: "Project" },
] as const;

function isActivePath(pathname: string, to: string) {
  return to === "/"
    ? pathname === "/"
    : pathname === to || pathname.startsWith(`${to}/`);
}

function AccountMenu() {
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
        Account{" "}
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
              <div className="grid size-9 place-items-center rounded-lg bg-primary text-primary-foreground">
                <UserRound className="size-4" />
              </div>
              <div className="min-w-0">
                <p className="truncate text-sm font-semibold text-popover-foreground">
                  Demo User
                </p>
                <p className="truncate text-xs text-muted-foreground">
                  demo@holonicasset.app
                </p>
              </div>
            </div>
          </DropdownMenuLabel>
        </DropdownMenuGroup>
        <div className="grid grid-cols-2 gap-2 px-2 py-2">
          <div className="rounded-lg border bg-muted/50 p-2">
            <p className="text-xs text-muted-foreground">Credits</p>
            <p className="mt-1 text-lg font-semibold">1,280</p>
          </div>
          <div className="rounded-lg border bg-muted/50 p-2">
            <p className="text-xs text-muted-foreground">Plan</p>
            <Badge variant="secondary" className="mt-1">
              Starter
            </Badge>
          </div>
        </div>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuItem
            render={<Link to="/settings" onClick={releaseMenu} />}
          >
            <Settings />
            Settings
          </DropdownMenuItem>
          <DropdownMenuItem>
            <CreditCard />
            Billing &amp; credits
          </DropdownMenuItem>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuItem variant="destructive" onClick={releaseMenu}>
          <LogOut />
          Log out
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function AppHeader() {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  });

  return (
    <header className="sticky top-0 z-50 border-b bg-background/90 backdrop-blur">
      <nav
        aria-label="Main navigation"
        className="grid min-h-14 grid-cols-[1fr_auto_1fr] items-center gap-2 px-3 sm:px-6"
      >
        <div aria-hidden="true" />
        <div className="flex items-center justify-center gap-1 sm:gap-2">
          {navItems.map(({ to, label }) => {
            const active = isActivePath(pathname, to);
            return (
              <Link
                key={to}
                to={to}
                search={
                  to === "/projects" ? { project: undefined, q: "" } : undefined
                }
                aria-current={active ? "page" : undefined}
                className={cn(
                  "inline-flex h-9 items-center rounded-md px-1.5 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none sm:px-3 sm:text-sm",
                  active &&
                    "bg-foreground text-background hover:bg-foreground hover:text-background",
                )}
              >
                {label}
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
