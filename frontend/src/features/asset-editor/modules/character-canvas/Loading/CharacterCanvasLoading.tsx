export function CharacterCanvasLoading() {
  return (
    <div
      className="absolute inset-0 grid place-items-center bg-[#eeece7]"
      role="status"
    >
      <span className="size-5 animate-spin rounded-full border-2 border-black/10 border-t-[#b86b70]" />
      <span className="sr-only">Loading canvas</span>
    </div>
  );
}
