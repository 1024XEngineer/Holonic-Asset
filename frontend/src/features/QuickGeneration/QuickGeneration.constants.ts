const seededPreviewClassNames: Record<string, string> = {
  "mushroom-courier":
    "bg-[radial-gradient(circle_at_35%_28%,#f4c665_0,transparent_25%),linear-gradient(145deg,#243847,#708d78)]",
  "moonlit-lantern":
    "bg-[radial-gradient(circle_at_55%_40%,#a9d5ff_0,transparent_24%),linear-gradient(145deg,#151d3a,#58608d)]",
  "forest-gate":
    "bg-[radial-gradient(circle_at_65%_30%,#d2e893_0,transparent_22%),linear-gradient(145deg,#20372b,#66835e)]",
};

const generatedPreviewClassName =
  "bg-[radial-gradient(circle_at_34%_28%,#f7d98c_0,transparent_24%),linear-gradient(145deg,#243847,#68808a)]";

export function getQuickAssetPreviewClassName(assetId: string) {
  return seededPreviewClassNames[assetId] ?? generatedPreviewClassName;
}
