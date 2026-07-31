import type { QueryClient } from "@tanstack/react-query";
import {
  HeadContent,
  Link,
  Outlet,
  createRootRouteWithContext,
  useRouter,
} from "@tanstack/react-router";
import { AlertTriangle, LoaderCircle, MapPinOff } from "lucide-react";

import { Button } from "@/components/ui/button";

function RouteStatus({
  action,
  description,
  icon,
  title,
}: {
  action?: React.ReactNode;
  description: string;
  icon: React.ReactNode;
  title: string;
}) {
  return (
    <main className="grid min-h-screen place-items-center bg-muted/30 p-6 text-foreground">
      <section className="w-full max-w-md border border-border bg-background p-6 shadow-sm">
        <div className="grid size-10 place-items-center border border-border bg-muted text-muted-foreground">
          {icon}
        </div>
        <h1 className="mt-5 text-xl font-semibold">{title}</h1>
        <p className="mt-2 text-sm leading-6 text-muted-foreground">
          {description}
        </p>
        {action ? (
          <div className="mt-6 flex flex-wrap gap-2">{action}</div>
        ) : null}
      </section>
    </main>
  );
}

function ProjectLibraryLink() {
  return (
    <Button
      nativeButton={false}
      render={<Link to="/projects" search={{ project: undefined, q: "" }} />}
    >
      Go to project library
    </Button>
  );
}

function RoutePending() {
  return (
    <RouteStatus
      title="Loading workspace"
      description="Preparing the next workspace."
      icon={<LoaderCircle className="size-5 animate-spin" />}
    />
  );
}

function RouteError({ error }: { error: Error }) {
  const router = useRouter();

  return (
    <RouteStatus
      title="Unable to open this workspace"
      description={
        error.message || "An unexpected error occurred while loading this page."
      }
      icon={<AlertTriangle className="size-5" />}
      action={
        <>
          <Button variant="outline" onClick={() => void router.invalidate()}>
            Try again
          </Button>
          <ProjectLibraryLink />
        </>
      }
    />
  );
}

function RouteNotFound() {
  return (
    <RouteStatus
      title="Page not found"
      description="The address does not match a workspace in this application."
      icon={<MapPinOff className="size-5" />}
      action={<ProjectLibraryLink />}
    />
  );
}

function RootComponent() {
  return (
    <>
      <HeadContent />
      <Outlet />
    </>
  );
}

type AppRouterContext = {
  queryClient: QueryClient;
};

export const Route = createRootRouteWithContext<AppRouterContext>()({
  component: RootComponent,
  pendingComponent: RoutePending,
  errorComponent: RouteError,
  notFoundComponent: RouteNotFound,
});
