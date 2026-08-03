import { Link } from "@tanstack/react-router";

const footerLinks = [
  { label: "Home", to: "/" },
  { label: "Image", to: "/generate" },
  { label: "Projects", to: "/projects" },
] as const;

export function HomeFooter() {
  return (
    <footer className="bg-background">
      <div className="mx-auto grid max-w-[100rem] gap-10 px-5 py-10 sm:px-8 md:grid-cols-[1fr_auto] md:items-end lg:px-10">
        <div>
          <p className="text-2xl font-semibold tracking-[-0.04em]">
            Holonic Asset
          </p>
          <p className="mt-2 max-w-md text-sm leading-6 text-muted-foreground">
            AI-powered game asset creation for characters, environments, UI, and
            consistent project libraries.
          </p>
        </div>
        <nav aria-label="Footer navigation" className="flex flex-wrap gap-5">
          {footerLinks.map(({ label, to }) => (
            <Link
              key={to}
              to={to}
              search={
                to === "/projects" ? { project: undefined, q: "" } : undefined
              }
              className="text-sm text-muted-foreground transition-colors hover:text-foreground focus-visible:rounded-sm focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
            >
              {label}
            </Link>
          ))}
        </nav>
      </div>
    </footer>
  );
}
