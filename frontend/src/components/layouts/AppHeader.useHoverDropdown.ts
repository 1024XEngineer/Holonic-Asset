import { useState } from "react";

import { useTimeout } from "@/hooks/use-timeout";

export function useHoverDropdown() {
  const [isOpen, setIsOpen] = useState(false);
  const [isPinned, setIsPinned] = useState(false);
  const { cancel: cancelClose, schedule: scheduleClose } = useTimeout();

  function openFromHover() {
    cancelClose();
    setIsOpen(true);
  }

  function closeFromHover() {
    if (isPinned) return;
    scheduleClose(() => setIsOpen(false), 120);
  }

  function onOpenChange(nextOpen: boolean) {
    setIsOpen(nextOpen);
    if (!nextOpen) setIsPinned(false);
  }

  function togglePinned() {
    cancelClose();
    setIsPinned((current) => {
      const next = !current;
      setIsOpen((isOpen) => next || !isOpen);
      return next;
    });
  }

  function releaseMenu() {
    cancelClose();
    setIsPinned(false);
    setIsOpen(false);
  }

  return {
    closeFromHover,
    isOpen,
    onOpenChange,
    openFromHover,
    releaseMenu,
    togglePinned,
  };
}
