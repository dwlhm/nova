const cp = require("child_process");
const fs = require("fs");
const path = require("path");
const vscode = require("vscode");

let client;

function activate(context) {
  client = new NovaClient(context);
  context.subscriptions.push(client);
  context.subscriptions.push(client.diagnostics);
  context.subscriptions.push(vscode.languages.registerDocumentFormattingEditProvider("nova", client));
  context.subscriptions.push(vscode.languages.registerHoverProvider("nova", client));
  context.subscriptions.push(vscode.languages.registerDefinitionProvider("nova", client));
  context.subscriptions.push(vscode.languages.registerReferenceProvider("nova", client));
  context.subscriptions.push(vscode.languages.registerRenameProvider("nova", client));
  context.subscriptions.push(vscode.languages.registerDocumentSymbolProvider("nova", client));
  context.subscriptions.push(vscode.languages.registerCompletionItemProvider("nova", client, "<", "@", ":", "."));
  client.start();
}

function deactivate() {
  if (!client) {
    return undefined;
  }
  return client.stop();
}

class NovaClient {
  constructor(context) {
    this.context = context;
    this.output = vscode.window.createOutputChannel("Nova");
    this.diagnostics = vscode.languages.createDiagnosticCollection("nova");
    this.pending = new Map();
    this.nextID = 1;
    this.buffer = Buffer.alloc(0);
    this.ready = Promise.resolve();
  }

  start() {
    const server = resolveServer(this.context);
    this.output.appendLine(`Starting Nova language server: ${server.command} ${server.args.join(" ")}`);
    if (server.cwd) {
      this.output.appendLine(`Nova server cwd: ${server.cwd}`);
    }

    try {
      this.process = cp.spawn(server.command, server.args, {
        cwd: server.cwd,
        stdio: ["pipe", "pipe", "pipe"]
      });
    } catch (error) {
      this.showStartError(error);
      return;
    }

    this.process.stdout.on("data", chunk => this.readMessages(chunk));
    this.process.stderr.on("data", chunk => this.output.append(chunk.toString()));
    this.process.on("error", error => this.showStartError(error));
    this.process.on("exit", (code, signal) => {
      this.output.appendLine(`Nova language server exited with code ${code}, signal ${signal}`);
    });

    this.ready = this.request("initialize", initializeParams())
      .then(() => this.notify("initialized", {}))
      .then(() => this.openCurrentDocuments())
      .catch(error => this.showStartError(error));

    this.context.subscriptions.push(vscode.workspace.onDidOpenTextDocument(document => this.didOpen(document)));
    this.context.subscriptions.push(vscode.workspace.onDidChangeTextDocument(event => this.didChange(event.document)));
    this.context.subscriptions.push(vscode.workspace.onDidCloseTextDocument(document => this.didClose(document)));
  }

  async stop() {
    if (!this.process || this.process.killed) {
      return;
    }
    try {
      await this.request("shutdown", null);
    } catch (error) {
      this.output.appendLine(`Nova shutdown request failed: ${error.message}`);
    }
    this.notify("exit", null);
    this.process.kill();
  }

  dispose() {
    this.stop();
    this.output.dispose();
  }

  openCurrentDocuments() {
    for (const document of vscode.workspace.textDocuments) {
      this.didOpen(document);
    }
  }

  didOpen(document) {
    if (!isNovaDocument(document)) {
      return;
    }
    this.ready.then(() => {
      this.notify("textDocument/didOpen", {
        textDocument: {
          uri: document.uri.toString(),
          languageId: "nova",
          version: document.version,
          text: document.getText()
        }
      });
    });
  }

  didChange(document) {
    if (!isNovaDocument(document)) {
      return;
    }
    this.ready.then(() => {
      this.notify("textDocument/didChange", {
        textDocument: {
          uri: document.uri.toString(),
          version: document.version
        },
        contentChanges: [{ text: document.getText() }]
      });
    });
  }

  didClose(document) {
    if (!isNovaDocument(document)) {
      return;
    }
    this.ready.then(() => {
      this.notify("textDocument/didClose", {
        textDocument: { uri: document.uri.toString() }
      });
    });
  }

  async provideDocumentFormattingEdits(document) {
    const edits = await this.requestForDocument("textDocument/formatting", document, {
      textDocument: { uri: document.uri.toString() },
      options: {
        tabSize: 2,
        insertSpaces: true
      }
    });
    return (edits || []).map(edit => new vscode.TextEdit(toRange(edit.range), edit.newText));
  }

  async provideHover(document, position) {
    const hover = await this.requestForPosition("textDocument/hover", document, position);
    if (!hover || !hover.contents) {
      return undefined;
    }
    const contents = hover.contents.kind === "markdown"
      ? new vscode.MarkdownString(hover.contents.value)
      : hover.contents.value;
    return new vscode.Hover(contents, hover.range ? toRange(hover.range) : undefined);
  }

  async provideDefinition(document, position) {
    const response = await this.requestForPosition("textDocument/definition", document, position);
    if (!response) {
      return undefined;
    }
    if (Array.isArray(response)) {
      return response.map(toLocation);
    }
    return toLocation(response);
  }

  async provideReferences(document, position, context) {
    const response = await this.requestForDocument("textDocument/references", document, {
      textDocument: { uri: document.uri.toString() },
      position: fromPosition(position),
      context: { includeDeclaration: context.includeDeclaration }
    });
    return (response || []).map(toLocation);
  }

  async prepareRename(document, position) {
    const response = await this.requestForPosition("textDocument/prepareRename", document, position);
    if (!response) {
      throw new Error("No Nova symbol at this position.");
    }
    return toRange(response.range || response);
  }

  async provideRenameEdits(document, position, newName) {
    const response = await this.requestForDocument("textDocument/rename", document, {
      textDocument: { uri: document.uri.toString() },
      position: fromPosition(position),
      newName
    });
    const edit = new vscode.WorkspaceEdit();
    for (const [uri, edits] of Object.entries((response && response.changes) || {})) {
      for (const item of edits) {
        edit.replace(vscode.Uri.parse(uri), toRange(item.range), item.newText);
      }
    }
    return edit;
  }

  async provideDocumentSymbols(document) {
    const response = await this.requestForDocument("textDocument/documentSymbol", document, {
      textDocument: { uri: document.uri.toString() }
    });
    return (response || []).map(item => new vscode.SymbolInformation(
      item.name,
      toSymbolKind(item.kind),
      item.containerName || "",
      toLocation(item.location)
    ));
  }

  async provideCompletionItems(document, position) {
    const response = await this.requestForPosition("textDocument/completion", document, position);
    const items = Array.isArray(response) ? response : (response && response.items) || [];
    return items.map(item => {
      const completion = new vscode.CompletionItem(item.label, toCompletionKind(item.kind));
      completion.detail = item.detail;
      if (item.documentation) {
        completion.documentation = item.documentation.kind === "markdown"
          ? new vscode.MarkdownString(item.documentation.value)
          : item.documentation;
      }
      completion.insertText = item.insertText || item.label;
      if (item.insertTextFormat === 2) {
        completion.insertText = new vscode.SnippetString(item.insertText || item.label);
      }
      completion.sortText = item.sortText;
      return completion;
    });
  }

  requestForPosition(method, document, position) {
    return this.requestForDocument(method, document, {
      textDocument: { uri: document.uri.toString() },
      position: fromPosition(position)
    });
  }

  async requestForDocument(method, document, params) {
    if (!isNovaDocument(document)) {
      return undefined;
    }
    await this.ready;
    return this.request(method, params);
  }

  request(method, params) {
    const id = this.nextID++;
    const payload = {
      jsonrpc: "2.0",
      id,
      method,
      params
    };
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      this.write(payload);
    });
  }

  notify(method, params) {
    this.write({
      jsonrpc: "2.0",
      method,
      params
    });
  }

  write(payload) {
    if (!this.process || !this.process.stdin || this.process.killed) {
      return;
    }
    const body = Buffer.from(JSON.stringify(payload), "utf8");
    this.process.stdin.write(`Content-Length: ${body.length}\r\n\r\n`);
    this.process.stdin.write(body);
  }

  readMessages(chunk) {
    this.buffer = Buffer.concat([this.buffer, chunk]);
    while (true) {
      const headerEnd = this.buffer.indexOf("\r\n\r\n");
      if (headerEnd < 0) {
        return;
      }
      const header = this.buffer.slice(0, headerEnd).toString("utf8");
      const contentLength = parseContentLength(header);
      if (contentLength < 0) {
        this.output.appendLine(`Invalid LSP header: ${header}`);
        this.buffer = Buffer.alloc(0);
        return;
      }
      const bodyStart = headerEnd + 4;
      const bodyEnd = bodyStart + contentLength;
      if (this.buffer.length < bodyEnd) {
        return;
      }
      const body = this.buffer.slice(bodyStart, bodyEnd).toString("utf8");
      this.buffer = this.buffer.slice(bodyEnd);
      this.handleMessage(JSON.parse(body));
    }
  }

  handleMessage(message) {
    if (message.id !== undefined && this.pending.has(message.id)) {
      const pending = this.pending.get(message.id);
      this.pending.delete(message.id);
      if (message.error) {
        pending.reject(new Error(message.error.message));
      } else {
        pending.resolve(message.result);
      }
      return;
    }
    if (message.method === "textDocument/publishDiagnostics") {
      this.publishDiagnostics(message.params);
    }
  }

  publishDiagnostics(params) {
    const uri = vscode.Uri.parse(params.uri);
    const diagnostics = (params.diagnostics || []).map(item => {
      const diagnostic = new vscode.Diagnostic(
        toRange(item.range),
        item.message,
        toDiagnosticSeverity(item.severity)
      );
      diagnostic.code = item.code;
      diagnostic.source = item.source || "nova";
      return diagnostic;
    });
    this.diagnostics.set(uri, diagnostics);
  }

  showStartError(error) {
    const message = `Nova language server failed: ${error.message}`;
    this.output.appendLine(message);
    vscode.window.showErrorMessage(message, "Show Output").then(selection => {
      if (selection === "Show Output") {
        this.output.show();
      }
    });
  }
}

function resolveServer(context) {
  const config = vscode.workspace.getConfiguration("nova");
  const configuredPath = config.get("serverPath", "");
  const configuredArgs = config.get("serverArgs", ["lsp"]);
  if (configuredPath && configuredPath.trim() !== "") {
    return {
      command: configuredPath,
      args: configuredArgs,
      cwd: findNovaWorkspace(context)
    };
  }

  const workspace = findNovaWorkspace(context);
  if (workspace) {
    return {
      command: "go",
      args: ["run", "./cmd/nova", "lsp"],
      cwd: workspace
    };
  }

  return {
    command: "nova",
    args: ["lsp"],
    cwd: undefined
  };
}

function findNovaWorkspace(context) {
  const candidates = [];
  for (const folder of vscode.workspace.workspaceFolders || []) {
    candidates.push(folder.uri.fsPath);
  }
  if (context && context.extensionPath) {
    candidates.push(path.resolve(context.extensionPath, "..", ".."));
  }
  for (const document of vscode.workspace.textDocuments) {
    if (document.uri.scheme === "file") {
      candidates.push(path.dirname(document.uri.fsPath));
    }
  }

  for (const candidate of candidates) {
    const workspace = findNovaWorkspaceAncestor(candidate);
    if (workspace) {
      return workspace;
    }
  }
  return undefined;
}

function findNovaWorkspaceAncestor(start) {
  let current = path.resolve(start);
  while (true) {
    if (isNovaWorkspace(current)) {
      return current;
    }
    const parent = path.dirname(current);
    if (parent === current) {
      return undefined;
    }
    current = parent;
  }
}

function isNovaWorkspace(root) {
  const goModPath = path.join(root, "go.mod");
  const cmdPath = path.join(root, "cmd", "nova", "main.go");
  if (!fs.existsSync(goModPath) || !fs.existsSync(cmdPath)) {
    return false;
  }
  const goMod = fs.readFileSync(goModPath, "utf8");
  return goMod.includes("module github.com/dwlhm/nova");
}

function initializeParams() {
  const folders = vscode.workspace.workspaceFolders || [];
  return {
    processId: process.pid,
    rootUri: folders[0] ? folders[0].uri.toString() : null,
    capabilities: {},
    workspaceFolders: folders.map(folder => ({
      uri: folder.uri.toString(),
      name: folder.name
    }))
  };
}

function isNovaDocument(document) {
  return document.languageId === "nova" && document.uri.scheme === "file";
}

function parseContentLength(header) {
  for (const line of header.split(/\r\n/)) {
    const [name, value] = line.split(":");
    if (name && value && name.toLowerCase() === "content-length") {
      const parsed = Number.parseInt(value.trim(), 10);
      return Number.isFinite(parsed) ? parsed : -1;
    }
  }
  return -1;
}

function fromPosition(position) {
  return {
    line: position.line,
    character: position.character
  };
}

function toRange(range) {
  return new vscode.Range(
    range.start.line,
    range.start.character,
    range.end.line,
    range.end.character
  );
}

function toLocation(location) {
  return new vscode.Location(vscode.Uri.parse(location.uri), toRange(location.range));
}

function toDiagnosticSeverity(severity) {
  switch (severity) {
    case 2:
      return vscode.DiagnosticSeverity.Warning;
    case 3:
      return vscode.DiagnosticSeverity.Information;
    case 4:
      return vscode.DiagnosticSeverity.Hint;
    default:
      return vscode.DiagnosticSeverity.Error;
  }
}

function toSymbolKind(kind) {
  switch (kind) {
    case 5:
      return vscode.SymbolKind.Class;
    case 12:
      return vscode.SymbolKind.Function;
    case 13:
      return vscode.SymbolKind.Variable;
    case 15:
      return vscode.SymbolKind.String;
    case 24:
      return vscode.SymbolKind.Event;
    default:
      return vscode.SymbolKind.Variable;
  }
}

function toCompletionKind(kind) {
  switch (kind) {
    case 3:
      return vscode.CompletionItemKind.Function;
    case 6:
      return vscode.CompletionItemKind.Variable;
    case 7:
      return vscode.CompletionItemKind.Class;
    case 10:
      return vscode.CompletionItemKind.Property;
    case 14:
      return vscode.CompletionItemKind.Keyword;
    case 23:
      return vscode.CompletionItemKind.Event;
    default:
      return vscode.CompletionItemKind.Text;
  }
}

module.exports = {
  activate,
  deactivate,
  NovaClient,
  resolveServer
};
