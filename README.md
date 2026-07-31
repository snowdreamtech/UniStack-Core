# UniStack Core Registry

[English](README.md) | [简体中文](README.zh-CN.md)
Welcome to the **UniStack Core Registry**! This repository serves as the official central index and package registry for the core ecosystem of UniStack.

## 📦 What is this?

This registry is maintained by the official team and contains the highest quality, foundational packages. When a user runs `unistack install package`, the `unistack` CLI queries this registry by default, and core packages have the highest resolution priority.

## 🚀 How to Contribute a Package

Core packages act as the foundation of the ecosystem, and all changes require strict review.

### Namespace Rules
- Core packages **MUST NOT** use any namespace prefix. All core packages reside in the global root namespace (e.g., just `nginx`).
- Your package files should be placed under the directory structure: `packages/[first_letter_of_package]/[package_name]/` (e.g., `packages/n/nginx/`).

### Submission Process
1. Fork this repository.
2. Create your package structure following the core namespace rules.
3. Include a valid `package.yml` and any installation scripts.
4. Open a Pull Request! Our CI will automatically validate your YAML format, no-prefix rule, and run a Sandbox Installation Test.

For detailed instructions, please read our [Contributing Guide](CONTRIBUTING.md).

## ⚖️ License & Agreements

By contributing to this repository, you agree that your contributions will be licensed under the [MIT License](LICENSE) (or the license specified in your package). 
All contributors are required to sign our [Contributor License Agreement (CLA)](CLA.md) upon opening their first Pull Request.
