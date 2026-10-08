import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { slug } from "../src/slug.ts";

describe("slug", () => {
  it("joins words with hyphens", () => {
    assert.equal(slug("Hello, World"), "hello-world");
  });

  it("keeps non-ASCII letters", () => {
    assert.equal(slug("Café au lait"), "café-au-lait");
  });

  it("returns an empty string for punctuation only", () => {
    assert.equal(slug("!!!"), "");
  });
});
