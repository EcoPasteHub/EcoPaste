# Contribution Guidelines

Thanks for your interests and contributions to `EcoPaste`! Before you submit your Pull Request, please take a moment to review the following guidelines to ensure a smooth process.

## Transparent Development

All work is conducted openly on GitHub. Whether you are a core team member or an external contributor, all Pull Requests must go through the same review process.

## Submitting Issues

We use [GitHub Issues](https://github.com/EcoPasteHub/EcoPaste/issues) for bug reports and feature suggestions. 

Before submitting an issue, please search for similar problems, as they may have already been addressed or are in progress. 

- For feature suggestions, describe the change you want and the expected behavior.

- For bug reports, please include detailed steps to reproduce the issue. 

## Submitting Pull Requests

### Contribution Workflow

- Claim an issue: Create a new issue on GitHub and claim it (or claim an existing one) to inform others that you are working on it, avoiding duplicate efforts.

- Project Development: Once you are ready, proceed with bug fixes or feature development in your local environment.

- Test: Test your changes thoroughly to ensure they do not break existing functionality.

- Submit a PR.

### Setup

- [Rust](https://tauri.app/v1/guides/getting-started/prerequisites/): Install Rust according to the official instructions.

- [Node.js](https://nodejs.org/en/): Required to run the project.

- [Pnpm](https://pnpm.io/): This project uses Pnpm for package management.

### Install Dependencies

```shell
pnpm install
```

### Start the Application

```shell
pnpm tauri dev
```

### Build the Application

> If you need to debug after building, add `--debug` to the following command

```shell
pnpm tauri build
```

## Commit Guidelines

Please follow the [conventional-changelog standard](https://www.conventionalcommits.org/en/v1.0.0/) for commit messages.

### Commit Types

Here are the available commit types:

- feat: New feature or functionality
- fix: Bug fix
- docs: Documentation updates
- style: Code style updates
- refactor: Code refactoring without new features or bug fixes
- perf: Performance improvements
- chore: Other changes

We're looking forward to your contributions to make EcoPaste much better!