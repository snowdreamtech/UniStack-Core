# Contributing to UniStack Community Registry

Thank you for your interest in contributing to the UniStack ecosystem!

## 1. Package Naming Rules

To prevent dependency confusion and naming conflicts, all community packages **MUST** use a namespace prefix matching your GitHub username or organization name.

- ❌ `nginx` (Reserved for Core Registry)
- ✅ `snowdreamtech/nginx` (Correct community namespace)

## 2. Directory Structure

Place your package in the `packages/` directory, nested by the first letter of your namespace, then your full namespace and package name:

```text
packages/
└── s/
    └── snowdreamtech/
        └── nginx/
            ├── package.yml
            └── tasks/
                └── main.yml
```

## 3. The `package.yml` File

Your `package.yml` must strictly follow the format:

```yaml
apiVersion: "v1"
kind: "package"

metadata:
  name: "snowdreamtech/nginx"
  version: "1.0.0"
  appVersion: "1.24.0"
  description: "Nginx web server packaged by snowdreamtech"
  authors: ["Your Name <email@example.com>"]
  homepage: "https://nginx.org"
  license: "MIT"
```

## 4. Pull Request Process

1. Fork this repository.
2. Create a new branch (`git checkout -b add-mypackage`).
3. Add your package files to the correct directory.
4. Commit your changes (`git commit -m "feat: add snowdreamtech/nginx"`).
5. Push to your fork and submit a Pull Request.
6. The CI will validate your package format and ensure it conforms to namespace rules.
7. **Important**: You must sign the Contributor License Agreement (CLA) when the bot prompts you in the PR comments.

Once approved and merged, the CI pipeline will automatically package, index, and deploy your contribution to the community registry!
