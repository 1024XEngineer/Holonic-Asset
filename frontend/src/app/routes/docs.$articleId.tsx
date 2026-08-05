import { createFileRoute, notFound } from "@tanstack/react-router";

import { Docs, isArticleId } from "@/features/docs";

export const Route = createFileRoute("/docs/$articleId")({
  component: DocsArticleRoute,
});

function DocsArticleRoute() {
  const { articleId } = Route.useParams();

  if (!isArticleId(articleId)) {
    throw notFound();
  }

  return <Docs articleId={articleId} />;
}
