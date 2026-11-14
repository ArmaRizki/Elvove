import { useMemo } from "react";
import { toHtml } from "../utils/richText";

/**
 * Renders admin rich text on a public page. The stored value is HTML, except on
 * rows written before the editor, which still hold Markdown. toHtml converts
 * the latter and sanitizes both before anything reaches the DOM.
 */
export default function RichTextContent({ value, softBreaks = false, className = "" }) {
  const html = useMemo(() => toHtml(value, { softBreaks }), [value, softBreaks]);
  if (!html) return null;
  return <div className={`rich-text ${className}`} dangerouslySetInnerHTML={{ __html: html }} />;
}
