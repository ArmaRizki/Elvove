// Content of the About story and the Terms page is HTML written by the admin
// rich-text editor. Rows saved before that editor still hold Markdown, so both
// formats are accepted and Markdown is upgraded the next time it is saved.
//
// Everything that reaches dangerouslySetInnerHTML goes through sanitizeHtml:
// DOMPurify removes tags and attributes the editor never produces, and the
// style hook below keeps only the handful of CSS properties it does produce.

import DOMPurify from "dompurify";

const HTML_PATTERN = /<(?:p|h[1-6]|ul|ol|li|blockquote|strong|em|u|s|span|br|a)\b/i;

// Inline styles the editor can set: font size and alignment.
const ALLOWED_STYLES = ["font-size", "text-align", "font-weight", "font-style", "text-decoration"];

const SANITIZE_CONFIG = {
  ALLOWED_TAGS: [
    "p", "br", "strong", "b", "em", "i", "u", "s",
    "h1", "h2", "h3", "h4", "h5", "h6",
    "ul", "ol", "li", "blockquote", "a", "span", "hr",
  ],
  ALLOWED_ATTR: ["style", "href", "target", "rel"],
  ALLOW_DATA_ATTR: false,
};

DOMPurify.addHook("afterSanitizeAttributes", (node) => {
  if (node.nodeType !== 1) return;

  if (node.hasAttribute("style")) {
    const kept = Array.from(node.style).filter((name) => ALLOWED_STYLES.includes(name));
    if (kept.length === 0) {
      node.removeAttribute("style");
    } else {
      node.setAttribute("style", kept.map((name) => `${name}: ${node.style.getPropertyValue(name)}`).join("; "));
    }
  }

  if (node.tagName === "A") {
    node.setAttribute("target", "_blank");
    node.setAttribute("rel", "noopener noreferrer");
  }
});

const escapeHtml = (text) =>
  text.replace(/[&<>"]/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[char]));

// The Markdown subset the old textareas documented: "#" / "##" / "###" headings,
// "- " or "* " bullets, "1. " numbered items, **bold**, and blank-line breaks.
const inline = (text) => escapeHtml(text).replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");

export function markdownToHtml(source, { softBreaks = false } = {}) {
  const lines = String(source || "").replace(/\r\n?/g, "\n").split("\n");
  const html = [];
  let paragraph = [];
  let list = null;

  const flushParagraph = () => {
    if (paragraph.length > 0) html.push(`<p>${paragraph.map(inline).join("<br>")}</p>`);
    paragraph = [];
  };
  const flushList = () => {
    if (list) html.push(`<${list.tag}>${list.items.map((item) => `<li>${inline(item)}</li>`).join("")}</${list.tag}>`);
    list = null;
  };

  lines.forEach((rawLine) => {
    const line = rawLine.trim();

    if (!line) {
      flushParagraph();
      flushList();
      return;
    }

    const heading = /^(#{1,3})\s+(.+)$/.exec(line);
    if (heading) {
      flushParagraph();
      flushList();
      const level = heading[1].length;
      html.push(`<h${level}>${inline(heading[2])}</h${level}>`);
      return;
    }

    const bullet = /^[-*]\s+(.+)$/.exec(line);
    const ordered = /^\d+\.\s+(.+)$/.exec(line);
    if (bullet || ordered) {
      flushParagraph();
      const tag = bullet ? "ul" : "ol";
      if (!list || list.tag !== tag) {
        flushList();
        list = { tag, items: [] };
      }
      list.items.push((bullet || ordered)[1]);
      return;
    }

    flushList();
    paragraph.push(line);
    // Terms render each line as its own paragraph; the About story keeps the
    // line breaks of one block together, matching its old whitespace-pre-line.
    if (!softBreaks) flushParagraph();
  });

  flushParagraph();
  flushList();
  return html.join("");
}

export function sanitizeHtml(html) {
  return DOMPurify.sanitize(html, SANITIZE_CONFIG);
}

// Stored value -> safe HTML ready to render.
export function toHtml(value, options) {
  const text = String(value || "").trim();
  if (!text) return "";
  return sanitizeHtml(HTML_PATTERN.test(text) ? text : markdownToHtml(text, options));
}

// Splits rendered HTML into the blocks the Terms page lays out in columns.
// `boundary` picks where a new block starts: at each top-level <h1> (the
// original behaviour) or at each <hr>, which the editor inserts through the
// "Pisah kolom" button and which is then dropped, because it marks the break
// instead of drawing a rule. Content before the first break is its own block.
export function splitHtmlSections(html, boundary = "heading") {
  const body = new DOMParser().parseFromString(html, "text/html").body;
  const sections = [];
  let current = [];

  const isBreak = (node) =>
    node.nodeType === 1 && (boundary === "marker" ? node.tagName === "HR" : node.tagName === "H1");

  const flush = () => {
    if (current.length > 0) {
      sections.push(current.map((node) => (node.nodeType === 1 ? node.outerHTML : escapeHtml(node.textContent || ""))).join(""));
    }
    current = [];
  };

  Array.from(body.childNodes).forEach((node) => {
    if (isBreak(node)) {
      flush();
      // A marker only divides the content; a heading also opens the new block.
      if (boundary === "marker") return;
    }
    current.push(node);
  });
  flush();

  return sections;
}
