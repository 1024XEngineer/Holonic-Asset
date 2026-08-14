export const uiSetCanvasWidthOptions = [640, 768, 1024, 1280, 1440, 1920];
export const uiSetCanvasHeightOptions = [360, 480, 576, 720, 768, 1080];

export const defaultUISetCanvasDimensions = {
  width: 1024,
  height: 768,
} as const;

export function isUISetCanvasWidth(value: number) {
  return uiSetCanvasWidthOptions.includes(value);
}

export function isUISetCanvasHeight(value: number) {
  return uiSetCanvasHeightOptions.includes(value);
}
