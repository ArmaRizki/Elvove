// Shared class strings for the admin panel.
// Keeping them in one place is what makes every admin page use the same
// borders, radii, spacing and button styles as the rest of the dashboard.

export const adminLabel =
  "block text-xs text-gray-700 tracking-widest";

export const adminInput =
  "mt-1 w-full min-w-0 bg-transparent border border-gray-200 focus:border-[#E11D48] outline-none px-3 py-2.5 text-sm text-gray-900";

export const adminCard = "border border-gray-200 bg-white p-4 sm:p-6";

export const adminPrimaryButton =
  "inline-flex items-center justify-center gap-2 bg-[#E11D48] text-ink px-4 py-2.5 text-sm tracking-wide transition-colors hover:bg-[#BE123C] disabled:opacity-50 disabled:cursor-not-allowed";

export const adminSecondaryButton =
  "inline-flex items-center justify-center gap-2 border border-gray-200 px-4 py-2.5 text-sm text-gray-700 transition-colors hover:border-gray-300 hover:text-gray-900 disabled:opacity-50 disabled:cursor-not-allowed";

export const adminTableWrap = "w-full overflow-x-auto border border-gray-200";

export const adminTable = "w-full min-w-[36rem] text-sm";

// Bookings carries seven columns, so it keeps a wider minimum before it
// starts scrolling inside its container.
export const adminTableWide = "w-full min-w-[44rem] text-sm";

export const adminIconButton =
  "inline-flex h-9 w-9 items-center justify-center text-gray-700 transition-colors hover:text-[#E11D48]";

export const adminNotice = (type) =>
  `border p-3 text-sm ${
    type === "success"
      ? "border-green-200 bg-green-50 text-green-800"
      : "border-red-200 bg-red-50 text-red-700"
  }`;
