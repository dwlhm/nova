"use strict";

const { describe, it } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

function loadRenderer() {
  const source = fs.readFileSync(path.join(__dirname, "..", "src", "nova-renderer.js"), "utf8");
  const context = { window: {}, module: { exports: {} }, exports: {} };
  vm.createContext(context);
  vm.runInContext(source, context);
  return { browser: context.window.NovaRenderer, commonjs: context.module.exports };
}

describe("NovaRenderer", () => {
  it("exports a CommonJS module for Node tests", () => {
    delete require.cache[require.resolve("../src/nova-renderer.js")];
    const renderer = require("../src/nova-renderer.js");
    renderer.clear();
    renderer.definePrimitive("sparkline", { mount: () => "mounted" });
    assert.equal(renderer.primitive("sparkline").mount(), "mounted");
  });

  it("attaches the same registry to window in browser-like contexts", () => {
    const renderer = loadRenderer();
    renderer.browser.clear();
    renderer.browser.definePrimitive("badge", { tag: "span" });
    assert.equal(renderer.commonjs.primitive("badge").tag, "span");
  });

  it("ignores empty primitive names", () => {
    const { browser } = loadRenderer();
    browser.clear();
    browser.definePrimitive("", { mount: () => null });
    assert.equal(browser.primitive(""), null);
  });
});
