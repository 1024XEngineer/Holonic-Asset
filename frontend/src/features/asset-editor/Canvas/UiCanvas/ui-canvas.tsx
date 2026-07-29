import type { EditorUiComponent } from "@/model";

type UiCanvasProps = {
  model: {
    components: EditorUiComponent[];
    selectedComponentIds: string[];
  };
  onEvent: (event: {
    type: "component.selection.toggled";
    componentId: string;
  }) => void;
};

export function UiCanvas({ model, onEvent }: UiCanvasProps) {
  const selectedComponentIds = new Set(model.selectedComponentIds);

  return (
    <main className="min-h-0 min-w-0 flex-1 overflow-hidden bg-[#eeece7] p-6 lg:p-8">
      <section
        aria-label="UI canvas"
        className="flex h-full min-h-[36rem] items-center justify-center"
      >
        <div className="relative aspect-video w-full max-w-[72rem] overflow-hidden border border-[#51493f]/25 bg-[#1f343a] shadow-[0_16px_40px_rgb(45_41_35/0.18)]">
          <div
            aria-hidden="true"
            className="absolute inset-0 opacity-20 [background-image:linear-gradient(90deg,transparent_49.75%,#d9f2ec_50%,transparent_50.25%),linear-gradient(transparent_49.75%,#d9f2ec_50%,transparent_50.25%)] [background-size:10%_10%]"
          />
          {model.components.map((component) => {
            const selected = selectedComponentIds.has(component.id);
            const zIndex = component.kind === "panel" ? 10 : 20;

            return (
              <button
                key={component.id}
                type="button"
                aria-label={`Select ${component.label}`}
                aria-pressed={selected}
                onClick={() =>
                  onEvent({
                    type: "component.selection.toggled",
                    componentId: component.id,
                  })
                }
                className={`absolute overflow-hidden border text-left transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#d99096] ${getUiComponentClassName(component.kind, selected)}`}
                style={{
                  left: `${component.bounds.x}%`,
                  top: `${component.bounds.y}%`,
                  width: `${component.bounds.width}%`,
                  height: `${component.bounds.height}%`,
                  zIndex,
                }}
              >
                <span className="pointer-events-none block truncate px-3 text-xs font-semibold">
                  {component.label}
                </span>
              </button>
            );
          })}
        </div>
      </section>
    </main>
  );
}

function getUiComponentClassName(
  kind: EditorUiComponent["kind"],
  selected: boolean,
) {
  const selectionClassName = selected
    ? "border-[#d99096] ring-2 ring-[#d99096]/60"
    : "border-transparent hover:border-[#d99096]/70";

  switch (kind) {
    case "panel":
      return `flex items-start rounded-md bg-[#f7f5f0] pt-3 text-[#2d2923] shadow-lg ${selectionClassName}`;
    case "label":
      return `flex items-center bg-transparent text-lg text-[#2d2923] ${selectionClassName}`;
    case "button":
      return `flex items-center justify-center rounded-sm bg-[#b86b70] text-center text-white shadow-sm ${selectionClassName}`;
  }
}
