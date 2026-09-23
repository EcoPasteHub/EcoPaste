import assert from "node:assert/strict";
import { test } from "node:test";
import { resolveImageOcrRefresh } from "./clipboardOcrRefresh";

const search = {
  atTop: true,
  enabled: true,
  keyword: "invoice",
  visible: true,
};

test("OCR worker refresh ignores ordinary item filter restrictions", () => {
  for (const filter of [
    { category: "image" },
    { range: "favorite" },
    { groupId: "custom-group" },
    { category: "image", groupId: "custom-group", range: "favorite" },
  ]) {
    assert.equal(
      resolveImageOcrRefresh("updated", { ...search, ...filter }),
      "reload",
    );
  }
});

test("worker results defer while scrolling or hidden and resume at the top", () => {
  assert.equal(
    resolveImageOcrRefresh("updated", { ...search, atTop: false }),
    "defer",
  );
  assert.equal(
    resolveImageOcrRefresh("updated", { ...search, visible: false }),
    "defer",
  );
  assert.equal(resolveImageOcrRefresh("updated", search), "reload");
});

test("clear invalidates cached matches even when hidden, scrolled or disabled", () => {
  assert.equal(
    resolveImageOcrRefresh("cleared", {
      ...search,
      atTop: false,
      visible: false,
    }),
    "reset",
  );
  assert.equal(
    resolveImageOcrRefresh("cleared", { ...search, enabled: false }),
    "reset",
  );
});

test("worker progress does not refresh unsearched or disabled lists", () => {
  assert.equal(
    resolveImageOcrRefresh("updated", { ...search, keyword: "  " }),
    "ignore",
  );
  assert.equal(
    resolveImageOcrRefresh("updated", { ...search, enabled: false }),
    "ignore",
  );
  assert.equal(
    resolveImageOcrRefresh("cleared", { ...search, keyword: "" }),
    "ignore",
  );
});
