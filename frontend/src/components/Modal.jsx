import { useEffect } from "react";
import { X } from "lucide-react";

/**
 * A dialog over the page, the one shape every admin form uses: a dimmed cover,
 * a white box that scrolls, a title and a close button.
 *
 * It is not rendered at all when `open` is false, so the caller keeps the
 * `{open && <Modal …>}` shape it had before and no state outlives the dialog.
 *
 * Props:
 *   open       whether the dialog is shown
 *   onClose    called by the cover, the close button and the Escape key
 *   title      the heading; also the accessible name when `label` is left out
 *   eyebrow    small red line above the heading, for a detail pane's context
 *   label      accessible name when the title alone is not enough
 *   children   the dialog body, usually a <form>
 *   footer     optional row under the body, outside its scroll area
 *   size       "md" (one column form) or "lg" (wider detail panes)
 *   tone       "white" for forms, "dim" for a darker cover behind details
 */
export default function Modal({
  open,
  onClose,
  title,
  eyebrow,
  label,
  children,
  footer,
  size = "md",
  tone = "white",
}) {
  // Escape closes it, and the page behind does not scroll while it is open.
  useEffect(() => {
    if (!open) return;
    const onKeyDown = (event) => {
      if (event.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKeyDown);
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", onKeyDown);
      document.body.style.overflow = previous;
    };
  }, [open, onClose]);

  if (!open) return null;

  const width = size === "lg" ? "max-w-3xl" : "max-w-lg";
  const cover = tone === "dim" ? "bg-black/40" : "bg-white/80";

  return (
    <div
      className={`fixed inset-0 z-50 flex items-center justify-center ${cover} p-4 sm:p-6`}
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label={label || title}
        className={`flex max-h-[90vh] w-full ${width} flex-col overflow-hidden border border-gray-200 bg-white`}
      >
        {(title || onClose) && (
          <div className="flex items-center justify-between gap-3 border-b border-gray-200 px-4 py-4 sm:px-6">
            <div className="min-w-0">
              {eyebrow && (
                <p className="text-xs tracking-widest text-[#E11D48]">{eyebrow}</p>
              )}
              <h2
                className={`min-w-0 break-words font-display text-gray-900 ${
                  eyebrow ? "mt-1 text-xl" : "text-lg"
                }`}
              >
                {title}
              </h2>
            </div>
            <button
              type="button"
              onClick={onClose}
              className="p-2 text-gray-700 hover:text-gray-900"
              aria-label="Tutup"
            >
              <X size={18} />
            </button>
          </div>
        )}
        <div className="min-h-0 flex-1 overflow-y-auto">{children}</div>
        {footer && <div className="border-t border-gray-200 px-4 py-3 sm:px-6">{footer}</div>}
      </div>
    </div>
  );
}
