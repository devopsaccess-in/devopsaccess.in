"use client";

import { useEffect, useRef } from "react";

// ConfirmDialog is the product's own confirmation, not the browser's.
// Built on <dialog> so focus trapping, Esc-to-dismiss, and inert background
// come from the platform rather than hand-rolled listeners.
export function ConfirmDialog({
  open,
  title,
  body,
  confirmLabel,
  destructive = false,
  pending = false,
  error,
  onConfirm,
  onCancel,
}: {
  open: boolean;
  title: string;
  body: React.ReactNode;
  confirmLabel: string;
  destructive?: boolean;
  pending?: boolean;
  error?: string | null;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  const ref = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    if (open && !el.open) el.showModal();
    if (!open && el.open) el.close();
  }, [open]);

  return (
    <dialog
      ref={ref}
      aria-labelledby="confirm-title"
      // Esc and backdrop-dismiss both route through onCancel so the parent
      // stays the single source of truth for whether the dialog is open.
      onCancel={(e) => {
        e.preventDefault();
        if (!pending) onCancel();
      }}
      onClick={(e) => {
        if (e.target === ref.current && !pending) onCancel();
      }}
      className="max-w-md rounded-lg border border-ink-line bg-ink-card p-0 text-mist backdrop:bg-black/70"
    >
      <div className="space-y-4 p-6">
        <h2 id="confirm-title" className="text-lg font-semibold text-white">
          {title}
        </h2>
        <div className="text-sm text-mist-dim">{body}</div>

        {error && <p className="text-sm text-alert">{error}</p>}

        <div className="flex justify-end gap-3 pt-2">
          <button type="button" className="btn-ghost" onClick={onCancel} disabled={pending}>
            Cancel
          </button>
          <button
            type="button"
            className={destructive ? "btn-danger" : "btn-primary"}
            onClick={onConfirm}
            disabled={pending}
            autoFocus={false}
          >
            {pending ? "Working…" : confirmLabel}
          </button>
        </div>
      </div>
    </dialog>
  );
}
