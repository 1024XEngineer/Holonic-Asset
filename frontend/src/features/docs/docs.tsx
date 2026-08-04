import { useEffect, useState } from "react";
import type { ComponentPropsWithoutRef, ComponentType } from "react";

import DirectionsEn, {
  metadata as directionsEn,
} from "./content/directions.mdx";
import PerspectiveEn, {
  metadata as perspectiveEn,
} from "./content/perspective.mdx";
import ReferenceEn, { metadata as referenceEn } from "./content/reference.mdx";
import { AppHeader } from "@/components/layouts/app-header";
import { cn } from "@/lib/utils";

type ArticleId = "directions" | "perspective" | "reference";
type Article = {
  Content: ComponentType<{ components?: Record<string, ComponentType> }>;
  metadata: { title: string };
};
type OutlineItem = { id: string; label: string; level: 2 | 3 };

const articles: Record<ArticleId, Article> = {
  reference: { Content: ReferenceEn, metadata: referenceEn },
  perspective: { Content: PerspectiveEn, metadata: perspectiveEn },
  directions: { Content: DirectionsEn, metadata: directionsEn },
};

const articleOrder: ArticleId[] = ["reference", "perspective", "directions"];

function createAnchorId(text: string, usedIds: Set<string>) {
  const base =
    text
      .toLowerCase()
      .trim()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "") || "section";
  let id = base;
  let suffix = 2;
  while (usedIds.has(id)) id = `${base}-${suffix++}`;
  usedIds.add(id);
  return id;
}

const mdxComponents = {
  h2: ({ className, ...props }: ComponentPropsWithoutRef<"h2">) => (
    <h2
      className={cn(
        "mt-12 scroll-mt-20 text-3xl font-semibold leading-tight tracking-[-0.035em] sm:text-4xl",
        className,
      )}
      {...props}
    />
  ),
  h3: ({ className, ...props }: ComponentPropsWithoutRef<"h3">) => (
    <h3
      className={cn(
        "mt-9 scroll-mt-20 text-xl font-semibold tracking-tight",
        className,
      )}
      {...props}
    />
  ),
  p: ({ className, ...props }: ComponentPropsWithoutRef<"p">) => (
    <p
      className={cn("mt-4 text-base leading-7 text-neutral-600", className)}
      {...props}
    />
  ),
  ul: ({ className, ...props }: ComponentPropsWithoutRef<"ul">) => (
    <ul className={cn("mt-6 grid gap-3", className)} {...props} />
  ),
  li: ({ className, ...props }: ComponentPropsWithoutRef<"li">) => (
    <li
      className={cn(
        "border-t border-neutral-950/10 pt-3 text-sm leading-6 text-neutral-700",
        className,
      )}
      {...props}
    />
  ),
};

export function Docs() {
  const [activeArticle, setActiveArticle] = useState<ArticleId>("reference");
  const article = articles[activeArticle];
  const onThisPage = "On this page";
  const [outline, setOutline] = useState<OutlineItem[]>([]);
  const [activeOutlineId, setActiveOutlineId] = useState("");

  useEffect(() => {
    const panel = document.getElementById(`${activeArticle}-panel`);
    if (!panel) return;

    const usedIds = new Set<string>();
    const nextOutline = Array.from(panel.querySelectorAll("h2, h3")).map(
      (heading) => {
        const id =
          heading.id || createAnchorId(heading.textContent ?? "", usedIds);
        heading.id = id;
        usedIds.add(id);
        return {
          id,
          label: heading.textContent ?? "",
          level: heading.tagName === "H2" ? 2 : 3,
        } as OutlineItem;
      },
    );

    setOutline(nextOutline);
    setActiveOutlineId(nextOutline[0]?.id ?? "");
  }, [activeArticle]);

  useEffect(() => {
    if (outline.length === 0) return;
    const syncActiveOutline = () => {
      const lastOutlineId = outline[outline.length - 1].id;
      const scrollableHeight =
        document.documentElement.scrollHeight - window.innerHeight;
      const reachedDocumentEnd =
        scrollableHeight > 0 && window.scrollY >= scrollableHeight - 1;
      if (reachedDocumentEnd) {
        setActiveOutlineId(lastOutlineId);
        return;
      }

      const offset = 80;
      let nextActiveOutlineId = outline[0].id;
      let closestOutlineId = nextActiveOutlineId;
      let closestDistance = Number.POSITIVE_INFINITY;

      for (const { id } of outline) {
        const heading = document.getElementById(id);
        if (!heading) continue;

        const distance = Math.abs(heading.getBoundingClientRect().top - offset);
        if (distance < closestDistance) {
          closestOutlineId = id;
          closestDistance = distance;
        }

        if (heading.getBoundingClientRect().top <= offset) {
          nextActiveOutlineId = id;
        }
      }

      setActiveOutlineId(
        closestDistance <= 32 ? closestOutlineId : nextActiveOutlineId,
      );
    };

    syncActiveOutline();
    window.addEventListener("scroll", syncActiveOutline, { passive: true });
    window.addEventListener("resize", syncActiveOutline);

    return () => {
      window.removeEventListener("scroll", syncActiveOutline);
      window.removeEventListener("resize", syncActiveOutline);
    };
  }, [outline]);

  return (
    <div className="min-h-screen bg-white text-neutral-950">
      <AppHeader />
      <main>
        <div className="mx-auto grid max-w-[110rem] lg:grid-cols-[17rem_minmax(0,1fr)_14rem]">
          <aside className="border-b border-neutral-950/10 bg-white px-5 py-8 sm:px-8 lg:sticky lg:top-14 lg:h-[calc(100vh-3.5rem)] lg:self-start lg:overflow-y-auto lg:border-r lg:border-b-0 lg:px-10 lg:py-12">
            <nav
              aria-label={onThisPage}
              role="tablist"
              className="flex gap-1 overflow-x-auto pb-1 lg:block lg:space-y-1.5 lg:overflow-visible"
            >
              {articleOrder.map((id) => {
                const active = activeArticle === id;
                return (
                  <button
                    key={id}
                    id={`${id}-tab`}
                    role="tab"
                    type="button"
                    aria-controls={`${id}-panel`}
                    aria-selected={active}
                    onClick={() => setActiveArticle(id)}
                    className={cn(
                      "block shrink-0 px-1 py-1 text-left text-sm text-neutral-500 transition-colors hover:text-neutral-950 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-cyan-700 lg:w-full",
                      active && "font-semibold text-neutral-950",
                    )}
                  >
                    {articles[id].metadata.title}
                  </button>
                );
              })}
            </nav>
          </aside>

          <article
            id={`${activeArticle}-panel`}
            role="tabpanel"
            aria-labelledby={`${activeArticle}-tab`}
            className="min-w-0 px-5 py-16 sm:px-8 sm:py-24 lg:px-12"
          >
            <div className="mx-auto max-w-3xl">
              <article.Content components={mdxComponents} />
            </div>
          </article>

          <aside className="hidden border-l border-neutral-950/10 bg-white px-7 py-12 lg:sticky lg:top-14 lg:block lg:h-[calc(100vh-3.5rem)] lg:self-start lg:overflow-y-auto">
            <nav aria-label={onThisPage}>
              <ol className="mt-5 space-y-3 border-l border-neutral-950/15 pl-4">
                {outline.map(({ id, label, level }) => (
                  <li key={id}>
                    <a
                      href={`#${id}`}
                      onClick={() => setActiveOutlineId(id)}
                      aria-current={
                        activeOutlineId === id ? "location" : undefined
                      }
                      className={cn(
                        "block text-xs leading-5 transition-colors hover:text-cyan-700",
                        level === 3 && "pl-3",
                        activeOutlineId === id
                          ? "font-semibold text-neutral-950"
                          : "text-neutral-500",
                      )}
                    >
                      {label}
                    </a>
                  </li>
                ))}
              </ol>
            </nav>
          </aside>
        </div>
      </main>
    </div>
  );
}
