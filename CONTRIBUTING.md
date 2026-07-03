# Contributing

English | [简体中文](./CONTRIBUTING.zh-CN.md)

Start with [README.md](./README.md) for the project overview and platform
scope. Then read [AGENTS.md](./AGENTS.md), which is the source of truth for
EcoPaste's Rust-first architecture boundaries, coding conventions, release
notes policy, and supported platforms.

## Development Setup

### Prerequisites

- macOS or Windows.
- Node.js 20 or newer.
- pnpm 10 or newer.
- Rust toolchain from `rust-toolchain.toml` (`1.96.0`, with `rustfmt` and
  `clippy`).
- Native dependencies required by Tauri v2. See the
  [Tauri prerequisites](https://tauri.app/start/prerequisites/) for your
  operating system.

### Install

```bash
pnpm install
```

### Run in Development

```bash
pnpm tauri dev
```

### Build

```bash
pnpm tauri build
```

## Quality Checks

Frontend:

```bash
pnpm lint
pnpm tsc
```

Rust:

```bash
cd src-tauri
cargo fmt
cargo clippy -- -D warnings
cargo test
```

Format frontend files:

```bash
pnpm format
```

## Ordinary Development

Use this flow when you are contributing without an AI coding agent:

1. Create a branch from the current target branch.
2. Follow the setup, run, build, and quality commands in this guide.
3. Keep durable behavior in Rust and UI behavior in React, following
   [AGENTS.md](./AGENTS.md).
4. Update [RELEASE-NEXT.md](./RELEASE-NEXT.md) for user-visible new
   capabilities.
5. Run the checks relevant to the change before opening a pull request.

## AI Development

AI-assisted development must use Trellis. Trellis keeps task plans, research,
project specs, AI session notes, and review context in files under
`.trellis/`, so future sessions do not have to rely on chat history.

Trellis documentation: [https://docs.trytrellis.app/](https://docs.trytrellis.app/)

Before an AI agent changes code, it should:

1. Load the current Trellis context:
   ```bash
   python3 ./.trellis/scripts/get_context.py
   python3 ./.trellis/scripts/get_context.py --mode phase
   python3 ./.trellis/scripts/get_context.py --mode packages
   ```
2. For non-trivial work, create or continue a task under `.trellis/tasks/`.
   Lightweight documentation-only changes may stay lightweight, but the agent
   must still read the relevant Trellis context and project rules.
3. Read the applicable `.trellis/spec/` index files before editing.
4. Keep task artifacts, implementation notes, and verification results current.
5. Finish with the quality checks required by the affected layer.

## Python Environment

Trellis scripts run through Python. Configure a local virtual environment before
using the Trellis workflow.

macOS:

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install --upgrade pip
```

Windows PowerShell:

```powershell
py -3 -m venv .venv
.\.venv\Scripts\Activate.ps1
python -m pip install --upgrade pip
```

This repository does not currently have a Python dependency manifest. If a
future Trellis script or hook reports a missing package, install the package in
the activated virtual environment and document the new dependency.

## First-Time AI Contributor Setup

Each AI contributor needs a personal Trellis developer identity. Initialize it
once after configuring Python:

```bash
python3 ./.trellis/scripts/init_developer.py <your-name>
```

This creates `.trellis/.developer` and `.trellis/workspace/<your-name>/`.
The identity file is local, while the workspace stores AI session journals and
task records that help future work continue from the right context.
