import { useEffect, useRef, type ReactNode } from "react";

interface DialogProps {
  open: boolean;
  labelledBy: string;
  onClose: () => void;
  children: ReactNode;
}

export function Dialog({ open, labelledBy, onClose, children }: DialogProps) {
  const ref = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const dialog = ref.current;
    if (dialog === null) {
      return;
    }
    if (open && !dialog.open) {
      dialog.showModal();
      dialog.querySelector<HTMLElement>("[data-autofocus]")?.focus();
    } else if (!open && dialog.open) {
      dialog.close();
    }
  }, [open]);

  return (
    <dialog
      ref={ref}
      aria-labelledby={labelledBy}
      onClose={onClose}
      onCancel={(event) => {
        event.preventDefault();
        onClose();
      }}
      onClick={(event) => {
        if (event.target === event.currentTarget) {
          onClose();
        }
      }}
      className="m-0 h-full max-h-none w-full max-w-none flex-col overflow-hidden border-0 bg-surface p-0 text-text backdrop:bg-overlay open:flex desk:m-auto desk:h-[min(640px,calc(100%-48px))] desk:w-[520px] desk:rounded-xl desk:border desk:border-border desk:shadow-pop"
    >
      {open && children}
    </dialog>
  );
}
