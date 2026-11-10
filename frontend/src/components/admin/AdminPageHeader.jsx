// One heading block for every admin page so titles, descriptions and actions
// always line up the same way, on desktop and on phones.
export default function AdminPageHeader({ title, description, meta, actions }) {
  return (
    <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div className="min-w-0">
        <h1 className="font-display text-2xl text-gray-900">{title}</h1>
        {description && (
          <p className="mt-1 text-sm text-gray-600">{description}</p>
        )}
        {meta && <div className="mt-2 text-xs text-gray-500">{meta}</div>}
      </div>

      {actions && (
        <div className="flex flex-wrap items-center gap-2">{actions}</div>
      )}
    </div>
  );
}
