export function snapToStep(value: number, step: number) {
  return step > 0 ? Math.round(value / step) * step : value;
}
