import { Music2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import type { EditorModeProps } from "./types";

export function AudioEditorMode({ renderHeader }: EditorModeProps) {
  return (
    <>
      {renderHeader("Sound studio")}
      <main className="flex min-h-0 flex-1 items-center justify-center bg-[#eeece7] p-6">
        <section className="max-w-sm border border-black/10 bg-white p-7 text-center shadow-sm">
          <Music2 className="mx-auto size-8 text-[#b86b70]" />
          <h2 className="mt-4 font-serif text-xl text-[#2d2923]">
            Audio is edited in Sound Studio
          </h2>
          <p className="mt-2 text-sm leading-6 text-[#51493f]">
            Arrange tracks and generate variations from the dedicated audio
            workspace.
          </p>
          <Button className="mt-5" render={<a href="/audio" />}>
            Open Sound Studio
          </Button>
        </section>
      </main>
    </>
  );
}
