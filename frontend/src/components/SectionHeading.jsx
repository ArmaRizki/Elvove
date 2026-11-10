export default function SectionHeading({ frame, title, subtitle, light, as: Heading = "h2" }) {
  return (
    <div className="mb-12 max-w-2xl">
      {frame && <p className="frame-number text-xs" data-frame={frame} />}
      <Heading className={`font-display text-3xl md:text-4xl ${light ? "text-ink" : "text-gray-900"}`}>
        {title}
      </Heading>
      {subtitle && (
        <p className={`mt-3 text-sm leading-relaxed ${light ? "text-ink/70" : "text-gray-700"}`}>
          {subtitle}
        </p>
      )}
    </div>
  );
}
