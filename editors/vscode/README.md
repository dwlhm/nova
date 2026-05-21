# Nova VS Code Extension

This extension works in VS Code-compatible editors, including Cursor.

## Development Install

Open the Nova repository in VS Code or Cursor, then run the extension host from this
folder. No npm install is required. The extension starts the language server with:

```bash
go run ./cmd/nova lsp
```

When used outside the Nova repository, set `nova.serverPath` to a Nova executable or make
`nova` available on `PATH`.
