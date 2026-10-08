/** Lowercases text and joins its letter and digit runs with hyphens. */
export function slug(text: string): string {
  return text
    .toLowerCase()
    .split(/[^\p{L}\p{N}]+/u)
    .filter((part) => part.length > 0)
    .join("-");
}
