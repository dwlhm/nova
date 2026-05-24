# @nova/renderer

Production renderer primitive registry used by Nova web artifacts.

This package intentionally exposes a small CommonJS-compatible browser module so the registry can
be tested directly with Node while still attaching `window.NovaRenderer` in generated web output.
