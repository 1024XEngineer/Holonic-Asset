import type {
  EditorCharacterAnimation,
  EditorCharacterAnimationClip,
} from "./editor-record";

export function getEditorCharacterAnimationClips(
  animations: EditorCharacterAnimation[],
): EditorCharacterAnimationClip[] {
  return animations.flatMap((animation) =>
    animation.kind === "group" ? animation.directions : [animation],
  );
}
