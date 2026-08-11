import { useEffect, useState } from "react";
import type { ComponentPropsWithoutRef, ComponentType } from "react";
import { useScrollSpy } from "@mantine/hooks";
import { Link } from "@tanstack/react-router";
import { z } from "zod";

import DirectionsEn, {
  metadata as directionsEn,
} from "./content/directions.mdx";
import PerspectiveEn, {
  metadata as perspectiveEn,
} from "./content/perspective.mdx";
import ReferenceEn, { metadata as referenceEn } from "./content/reference.mdx";
import TilesetsEn, { metadata as tilesetsEn } from "./content/tilesets.mdx";
import { AppHeader } from "@/components/layouts/app-header";
import { cn } from "@/lib/utils";

type Article = {
  Content: ComponentType<{ components?: Record<string, ComponentType> }>;
  metadata: { title: string };
};

export const articleOrder = [
  "reference",
  "perspective",
  "directions",
  "tilesets",
] as const;

const articleIdSchema = z.enum(articleOrder);

export type ArticleId = z.infer<typeof articleIdSchema>;

const articles: Record<ArticleId, Article> = {
  reference: { Content: ReferenceEn, metadata: referenceEn },
  perspective: { Content: PerspectiveEn, metadata: perspectiveEn },
  directions: { Content: DirectionsEn, metadata: directionsEn },
  tilesets: { Content: TilesetsEn, metadata: tilesetsEn },
};

export function isArticleId(value: unknown): value is ArticleId {
  return articleIdSchema.safeParse(value).success;
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

type DocsProps = {
  articleId: ArticleId;
};

export function Docs({ articleId }: DocsProps) {
  const article = articles[articleId];
  const outline = useScrollSpy({
    selector: `#${articleId}-panel :is(h2, h3)`,
    offset: 80,
  });
  const [selectedOutlineId, setSelectedOutlineId] = useState<string | null>(
    null,
  );
  const activeOutlineId = selectedOutlineId ?? outline.data[outline.active]?.id;

  useEffect(() => {
    setSelectedOutlineId(null);
  }, [articleId, outline.active]);

  return (
    <div className="min-h-screen bg-white text-neutral-950">
      <AppHeader />
      <main>
        <div className="mx-auto grid max-w-[110rem] lg:grid-cols-[17rem_minmax(0,1fr)_14rem]">
          <aside className="border-b border-neutral-950/10 bg-white px-5 py-8 sm:px-8 lg:sticky lg:top-14 lg:h-[calc(100vh-3.5rem)] lg:self-start lg:overflow-y-auto lg:border-r lg:border-b-0 lg:px-10 lg:py-12">
            <nav
              aria-label="Documentation topics"
              className="flex gap-1 overflow-x-auto pb-1 lg:block lg:space-y-1.5 lg:overflow-visible"
            >
              {articleOrder.map((id) => {
                const active = articleId === id;
                return (
                  <Link
                    key={id}
                    to="/docs/$articleId"
                    params={{ articleId: id }}
                    aria-current={active ? "page" : undefined}
                    className={cn(
                      "block shrink-0 px-1 py-1 text-left text-sm text-neutral-500 transition-colors hover:text-neutral-950 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-cyan-700 lg:w-full",
                      active && "font-semibold text-neutral-950",
                    )}
                  >
                    {articles[id].metadata.title}
                  </Link>
                );
              })}
            </nav>
          </aside>

          <article
            id={`${articleId}-panel`}
            className="min-w-0 px-5 py-16 sm:px-8 sm:py-24 lg:px-12"
          >
            <div className="mx-auto max-w-3xl">
              <article.Content components={mdxComponents} />
            </div>
          </article>

          <aside className="hidden border-l border-neutral-950/10 bg-white px-7 py-12 lg:sticky lg:top-14 lg:block lg:h-[calc(100vh-3.5rem)] lg:self-start lg:overflow-y-auto">
            <nav aria-label="Table of contents">
              <ol className="mt-5 space-y-3 border-l border-neutral-950/15 pl-4">
                {outline.data.map(({ id, value, depth }) => (
                  <li key={id}>
                    <a
                      href={`#${id}`}
                      onClick={() => setSelectedOutlineId(id)}
                      aria-current={
                        activeOutlineId === id ? "location" : undefined
                      }
                      className={cn(
                        "block text-xs leading-5 transition-colors hover:text-cyan-700",
                        depth === 3 && "pl-3",
                        activeOutlineId === id
                          ? "font-semibold text-neutral-950"
                          : "text-neutral-500",
                      )}
                    >
                      {value}
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
