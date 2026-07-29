# Proofpane

Lean 4 proof-state sidecar with lightweight web view for any editor.
It wraps the `lake serve` language server and launches a local HTTP server that pushes interactive proof states to a browser via Server-Sent Events (SSE).
This project is meant to provide an alternative to the VS Code Lean 4 extension for any editor that implements the LSP.

If you've only ever used Lean via VS Code, here's a quick review to get started with the non-IDE workflow.
First, make sure you have the Lean 4 CLI version manager properly set up on your system; see the [`elan` readme](https://github.com/leanprover/elan#manual-installation) for details.
You can set up a fresh project using Mathlib as follows (all these things may take a while; especially on Windows, I found that temporarily disabling Defender real-time protection really speeds it up):

```sh
lake +leanprover-community/mathlib4:lean-toolchain new <your_project_name> math
cd <your_project_name>
lake exe cache get      # enable Mathlib cache for faster builds
lake build              # type-check your project
```

## Installation and usage

If you already have Go installed, you can simply run `go install github.com/lcwllmr/proofpane`.
Otherwise, download the latest [release](https://github.com/lcwllmr/proofpane/releases) for your platform and make sure it's in your `PATH`.
Next, you'll need to configure your editor.

### Helix

Add the following to your Helix `languages.toml` to use `proofpane serve` instead of the default `lake serve`.
Proofpane wraps and launches your local `lake serve` internally.

```toml
[language-server.proofpane]
command = "proofpane"
args = ["serve"]

[[language]]
name = "lean"
scope = "source.lean"
injection-regex = "lean"
file-types = ["lean"]
roots = ["lakefile.lean", "lakefile.toml", "lean-toolchain"]
language-servers = [ "proofpane" ]
```

Whenever you need to check your proof state, you need to move your cursor to the right place in your proof and then trigger the cursor hook to open or refresh the browser page showing the current context.
Helix does not currently support generic hooks on cursor move natively, so you must bind the cursor update to a keystroke you frequently use.

```toml
[keys.normal]
"C-l" = [":sh proofpane cursor -x --line %{cursor_line} --col %{cursor_column}"]
```
