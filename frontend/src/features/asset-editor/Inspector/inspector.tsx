import { ScrollArea } from "@/components/ui/scroll-area";
import { useTranslation } from "react-i18next";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

import { InspectorEdit } from "./edit";
import { InspectorHistory } from "./history";
import { SceneryInspectorContent } from "./scenery";
import type { InspectorProps } from "./inspector.types";

export type { InspectorSubmitRequest } from "./inspector.types";

export function Inspector(props: InspectorProps) {
  const { t } = useTranslation("editor");
  const primaryTab = props.kind === "scenery" ? "inspect" : "edit";
  return (
    <aside className="flex min-h-0 w-full shrink-0 flex-col border-t bg-background lg:w-80 lg:border-t-0 lg:border-l">
      <Tabs
        defaultValue={primaryTab}
        className="flex min-h-0 flex-1 flex-col gap-0"
      >
        <div className="border-b px-3 py-2">
          <TabsList className="grid h-8 w-full grid-cols-2">
            <TabsTrigger value={primaryTab} className="text-xs">
              {t(primaryTab)}
            </TabsTrigger>
            <TabsTrigger value="history" className="text-xs">
              {t("history")}
            </TabsTrigger>
          </TabsList>
        </div>
        <ScrollArea className="max-h-80 min-h-0 flex-1 lg:max-h-none">
          <TabsContent value={primaryTab} className="m-0 p-3">
            {props.kind === "scenery" ? (
              <SceneryInspectorContent
                layer={props.layer}
                dimensions={props.dimensions}
                visible={props.visible}
                onToggleVisibility={props.onToggleVisibility}
              />
            ) : (
              <InspectorEdit
                selectedNodes={props.selectedNodes}
                selectedFrames={props.selectedFrames}
                prompt={props.prompt}
                onPromptChange={props.onPromptChange}
                animations={props.animations}
                prototype={props.prototype}
                onSubmit={props.onSubmit}
                onClearSelection={props.onClearSelection}
                isSubmitting={props.isSubmitting}
              />
            )}
          </TabsContent>
          <TabsContent value="history" className="m-0 p-4">
            <InspectorHistory entries={props.history} />
          </TabsContent>
        </ScrollArea>
      </Tabs>
    </aside>
  );
}
