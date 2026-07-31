import { AppHeader } from "@/components/layouts/app-header";

import { HomeCapabilities } from "./home-capabilities";
import { HomeClosingCta } from "./home-closing-cta";
import { HomeFooter } from "./home-footer";
import { HomeHero } from "./home-hero";
import { HomeProjectStory } from "./home-project-story";
import { HomeWorkflow } from "./home-workflow";

export function Home() {
  return (
    <div className="min-h-screen bg-background text-foreground">
      <AppHeader />
      <main>
        <HomeHero />
        <HomeCapabilities />
        <HomeProjectStory />
        <HomeWorkflow />
        <HomeClosingCta />
      </main>
      <HomeFooter />
    </div>
  );
}
