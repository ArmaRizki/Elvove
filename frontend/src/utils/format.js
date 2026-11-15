// Formatters shared by the public pages and the admin panel, so a price reads
// the same way on every screen.
//
// One implementation only: the public pages used to carry their own
// Intl.NumberFormat (which writes "Rp 1.500.000" with a non-breaking space) and
// admin pages called toLocaleString directly, so the same package showed two
// different prices.

/** 18500000 → "Rp18.500.000". Returns null when the amount is unknown. */
export function formatPrice(value) {
  if (value === null || value === undefined || value === "") return null;
  const number = Number(value);
  if (Number.isNaN(number)) return null;
  return `Rp${number.toLocaleString("id-ID", { maximumFractionDigits: 0 })}`;
}
