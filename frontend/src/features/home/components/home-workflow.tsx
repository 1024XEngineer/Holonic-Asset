const workflow = [
  {
    number: "01",
    title: "Project & Visual Direction",
    description:
      "Define game type, visual style, and target platform. Generate a master Visual Direction Sheet to seed unified AI context for all assets.",
  },
  {
    number: "02",
    title: "Multi-View Prototypes",
    description:
      "Generate character and object prototypes with 1, 4, or 8-directional views (top-down, side-on, isometric) maintaining consistent style.",
  },
  {
    number: "03",
    title: "Animations & Bound SFX",
    description:
      "Expand prototypes into 4–16 frame action sequences (Idle, Walk, Attack) and attach generated sound effects directly to animations.",
  },
  {
    number: "04",
    title: "Engine-Ready Export",
    description:
      "Export structured Spritesheets, Parallax Scenery Layers, Tilemaps, UI components, and audio files ready for Unity or Godot.",
  },
];

export function HomeWorkflow() {
  return (
    <section
      aria-labelledby="workflow-heading"
      className="bg-lime-300 text-neutral-950"
    >
      <div className="mx-auto max-w-[100rem] px-5 py-20 sm:px-8 sm:py-28 lg:px-10 lg:py-32">
        <div className="grid gap-8 lg:grid-cols-[minmax(0,1fr)_minmax(18rem,.45fr)] lg:items-end">
          <h2
            id="workflow-heading"
            className="max-w-4xl text-5xl leading-[0.94] font-semibold tracking-[-0.06em] sm:text-6xl lg:text-7xl"
          >
            One workflow from vision to game-ready pipeline.
          </h2>
          <p className="max-w-md text-sm leading-7 text-neutral-700 lg:justify-self-end">
            Holonic Asset connects project visual context, asset prototyping,
            animation frame generation, and sound effect binding into one
            structured workflow.
          </p>
        </div>

        <ol className="mt-16 grid border-t border-neutral-950/30 sm:grid-cols-2 lg:grid-cols-4">
          {workflow.map(({ description, number, title }) => (
            <li
              key={number}
              className="border-b border-neutral-950/30 py-8 lg:border-r lg:border-b-0 lg:px-6 lg:first:pl-0 lg:last:border-r-0"
            >
              <span className="font-mono text-xs font-bold">{number}</span>
              <h3 className="mt-10 text-xl font-semibold tracking-tight">
                {title}
              </h3>
              <p className="mt-3 text-sm leading-6 text-neutral-700">
                {description}
              </p>
            </li>
          ))}
        </ol>
      </div>
    </section>
  );
}
