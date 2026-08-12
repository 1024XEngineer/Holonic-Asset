import { useState } from "react";
import { useTranslation } from "react-i18next";

const workflow = [
  { number: "01", key: "projectStyle" },
  { number: "02", key: "prototypes" },
  { number: "03", key: "animations" },
  { number: "04", key: "export" },
] as const;

export function HomeWorkflow() {
  const { t } = useTranslation("home");
  const [activeStep, setActiveStep] = useState("01");

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
            {t("workflowTitle")}
          </h2>
          <p className="max-w-md text-sm leading-7 text-neutral-700 lg:justify-self-end">
            {t("workflowDescription")}
          </p>
        </div>
        <ol className="mt-16 grid border-t border-neutral-950/30 sm:grid-cols-2 lg:grid-cols-4">
          {workflow.map(({ key, number }) => (
            <li
              key={number}
              className="border-b border-neutral-950/30 py-8 lg:border-r lg:border-b-0 lg:px-6 lg:first:pl-0 lg:last:border-r-0"
            >
              <button
                type="button"
                onClick={() => setActiveStep(number)}
                aria-pressed={activeStep === number}
                className="group w-full text-left focus-visible:ring-3 focus-visible:ring-neutral-950/50 focus-visible:outline-none"
              >
                <span
                  className={`font-mono text-xs font-bold transition-colors ${activeStep === number ? "text-neutral-950" : "text-neutral-600"}`}
                >
                  {number}
                </span>
                <h3 className="mt-10 text-xl font-semibold tracking-tight">
                  {t(`workflow.${key}.title`)}
                </h3>
                <p
                  className={`mt-3 text-sm leading-6 transition-opacity ${activeStep === number ? "text-neutral-950 opacity-100" : "text-neutral-700 opacity-65 group-hover:opacity-100"}`}
                >
                  {t(`workflow.${key}.description`)}
                </p>
                <span
                  className={`mt-5 block h-1 origin-left bg-neutral-950 transition-transform duration-300 ease-[cubic-bezier(0.22,1,0.36,1)] ${activeStep === number ? "scale-x-100" : "scale-x-0 group-hover:scale-x-50"}`}
                />
              </button>
            </li>
          ))}
        </ol>
      </div>
    </section>
  );
}
