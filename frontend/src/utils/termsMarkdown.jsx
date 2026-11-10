// Renders the Terms & Conditions Markdown subset as React elements.
//
// Supported: "# " / "## " / "### " headings, "- " or "* " bullet lists,
// "1. " numbered lists, **bold** inline text, and blank-line paragraphs.
// Content is never passed to dangerouslySetInnerHTML: every piece of text is a
// React text node, so admin-entered Markdown cannot inject markup.

import { Fragment } from "react";

// Sizes follow the site's heading scale: section titles use the SectionHeading
// size, sub-headings step down. The section grid supplies the separators.
const HEADING_CLASSES = {
  1: "mb-4 font-display text-xl md:text-2xl text-gray-900",
  2: "mb-3 mt-8 font-display text-lg text-gray-900",
  3: "mb-2 mt-6 font-display text-base text-gray-900",
};

// Splits Markdown into sections at each top-level "# " heading. The text before
// the first heading (if any) is its own section. Each section keeps its exact
// source lines, so rendering a section is identical to rendering the whole text.
export function splitTermsSections(source) {
  const sections = [];
  let current = [];

  String(source || "").split(/\r?\n/).forEach((rawLine) => {
    if (/^#\s+/.test(rawLine.trim()) && current.length > 0) {
      sections.push(current);
      current = [];
    }
    current.push(rawLine);
  });

  if (current.length > 0) sections.push(current);
  return sections.map((lines) => lines.join("\n"));
}

const BOLD_PATTERN = /\*\*([^*]+)\*\*/g;

export function renderInline(text) {
  const parts = [];
  let last = 0;
  let match;
  BOLD_PATTERN.lastIndex = 0;

  while ((match = BOLD_PATTERN.exec(text)) !== null) {
    if (match.index > last) parts.push(text.slice(last, match.index));
    parts.push(<strong key={`b-${match.index}`} className="font-semibold text-gray-900">{match[1]}</strong>);
    last = match.index + match[0].length;
  }

  if (last < text.length) parts.push(text.slice(last));
  return parts.map((part, index) => <Fragment key={index}>{part}</Fragment>);
}

export function renderTerms(source, { paragraphClass, listClass, orderedClass } = {}) {
  const paragraph = paragraphClass ?? "mb-4 text-sm leading-7 text-gray-600";
  const bullets = listClass ?? "list-disc space-y-2 pl-7 text-sm leading-7 text-gray-600 mb-8";
  const numbered = orderedClass ?? "list-decimal space-y-2 pl-7 text-sm leading-7 text-gray-600 mb-8";

  const lines = String(source || "").split(/\r?\n/);
  const elements = [];
  let items = [];
  let listType = null;
  let key = 0;

  const flushList = () => {
    if (items.length === 0) return;
    const ListTag = listType === "ol" ? "ol" : "ul";
    elements.push(
      <ListTag key={`list-${key++}`} className={listType === "ol" ? numbered : bullets}>
        {items.map((item, index) => (
          <li key={index}>{renderInline(item)}</li>
        ))}
      </ListTag>
    );
    items = [];
    listType = null;
  };

  lines.forEach((rawLine) => {
    const line = rawLine.trim();

    if (!line) {
      flushList();
      return;
    }

    const heading = /^(#{1,3})\s+(.*)$/.exec(line);
    if (heading) {
      flushList();
      const level = heading[1].length;
      const Tag = `h${level + 1}`;
      // "#" -> h2, "##" -> h3, "###" -> h4, so the Markdown level is kept.
      elements.push(
        <Tag key={`heading-${key++}`} className={HEADING_CLASSES[level]}>
          {renderInline(heading[2])}
        </Tag>
      );
      return;
    }

    const bullet = /^[-*]\s+(.*)$/.exec(line);
    if (bullet) {
      if (listType !== "ul") flushList();
      listType = "ul";
      items.push(bullet[1]);
      return;
    }

    const ordered = /^\d+\.\s+(.*)$/.exec(line);
    if (ordered) {
      if (listType !== "ol") flushList();
      listType = "ol";
      items.push(ordered[1]);
      return;
    }

    flushList();
    elements.push(
      <p key={`p-${key++}`} className={paragraph}>
        {renderInline(line)}
      </p>
    );
  });

  flushList();
  return elements;
}
