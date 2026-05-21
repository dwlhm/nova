# Nova For Cursor

Cursor loads VS Code-compatible extensions from its local extension directory. For local
development, link the Nova extension into Cursor:

```bash
./editors/cursor/install-local.sh
```

Then reload Cursor and open a `.nova` file. The extension will start:

```bash
go run ./cmd/nova lsp
```

If you use Nova outside this checkout, set `nova.serverPath` in Cursor settings to a Nova
executable, or make `nova` available on `PATH`.

The default Cursor extension directory is:

```txt
~/.cursor/extensions
```

Set `CURSOR_EXTENSIONS_DIR` before running the script if your Cursor profile uses another
directory.
