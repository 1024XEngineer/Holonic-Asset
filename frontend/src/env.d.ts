interface ImportMetaEnv {
  readonly VITE_CORE_API_BASE_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

declare module "*.css";

declare module "*.mdx" {
  import type { ComponentType } from "react";

  export const metadata: {
    title: string;
  };

  const MDXContent: ComponentType<{
    components?: Record<string, ComponentType>;
  }>;
  export default MDXContent;
}
