#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
extension_source="$repo_root/editors/vscode"
extension_id="dwlhm.nova-vscode-0.1.0"
extensions_root="${CURSOR_EXTENSIONS_DIR:-$HOME/.cursor/extensions}"
extension_target="$extensions_root/$extension_id"

if [[ ! -f "$extension_source/package.json" ]]; then
  echo "Nova extension source is missing: $extension_source/package.json" >&2
  exit 1
fi

mkdir -p "$extensions_root"

if [[ -L "$extension_target" ]]; then
  current_target="$(readlink "$extension_target")"
  if [[ "$current_target" == "$extension_source" ]]; then
    echo "Nova Cursor extension is already linked at $extension_target"
    exit 0
  fi
  echo "Refusing to replace existing symlink: $extension_target -> $current_target" >&2
  exit 1
fi

if [[ -e "$extension_target" ]]; then
  echo "Refusing to replace existing Cursor extension directory: $extension_target" >&2
  echo "Move it aside first, then rerun this script." >&2
  exit 1
fi

ln -s "$extension_source" "$extension_target"
echo "Linked Nova Cursor extension:"
echo "  $extension_target -> $extension_source"
echo
echo "Reload Cursor, then open a .nova file."
