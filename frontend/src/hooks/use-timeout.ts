import { useCallback, useEffect, useRef } from "react";

type TimeoutCallback = () => void;

export function useTimeout() {
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const cancel = useCallback(() => {
    if (timeoutRef.current !== null) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
  }, []);

  const schedule = useCallback(
    (callback: TimeoutCallback, delay: number) => {
      cancel();
      timeoutRef.current = setTimeout(() => {
        timeoutRef.current = null;
        callback();
      }, delay);
    },
    [cancel],
  );

  useEffect(() => cancel, [cancel]);

  return { cancel, schedule };
}
