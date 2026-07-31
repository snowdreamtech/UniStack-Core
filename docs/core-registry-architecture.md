# Core Registry Architecture

This document defines the standard architecture and operational model for the UniStack Core Registry.

## 1. Overview

The UniStack community ecosystem adopts a hybrid architecture: **Centralized PR-driven as primary, Distributed Custom Registry as auxiliary**. This model ensures security, high performance, and rapid ecosystem growth while accommodating enterprise-level private deployments.

## 2. Registry Tiers

The ecosystem is divided into three distinct tiers:

### 2.1 Core Registry (`unistack-core`)
- **Maintainer**: Official UniStack Team.
- **Characteristics**: Heavily audited, highly secure, and guaranteed backward compatibility.
- **Namespace**: Occupies the global, prefix-free root namespace (e.g., `nginx`).

### 2.2 Community Registry (`unistack-community`)
- **Maintainer**: Community-driven, reviewed by maintainers and automated systems.
- **Characteristics**: A centralized monolithic repository (Monorepo) where developers submit Pull Requests (PRs) to add or update packages.
- **Namespace**: MUST include an author or organization prefix (e.g., `snowdreamtech/nginx`).

### 2.3 Custom Registries (Taps)
- **Maintainer**: Individual users or enterprises.
- **Characteristics**: Fully decentralized. Users manually add the registry URL via CLI (e.g., `unistack registry add my-company https://...`).
- **Use Case**: Ideal for private, internal, or closed-source enterprise software that cannot be published to the centralized community.

## 3. Storage and Distribution (GitOps Model)

The registry implements a strict separation of source code and binary distribution.

- **Source of Truth (GitHub)**: The GitHub repositories (`unistack-community` and `unistack-core`) host ONLY plain-text configuration files (e.g., `package.yml`). No binary files (`.tar.gz`, `.uspkg`) are ever committed to the repository, keeping the Git history clean and fast.
- **Artifact Distribution (CDN)**: Once a PR is merged, the automated CI pipeline builds the source into binary packages, updates the registry index (`packages.db`), and pushes these artifacts to a CDN or static object storage (e.g., Cloudflare R2, AWS S3).
- **Client Retrieval**: The UniStack CLI downloads indexes and packages directly from the high-speed CDN, completely bypassing the GitHub repository.

## 4. Security and Quality Control

To prevent ecosystem contamination, such as namesquatting, dependency confusion, and malicious code execution, the following security measures are strictly enforced:

### 4.1 Strict Namespacing
Community packages are strictly prohibited from using the global root namespace. Any PR attempting to submit a package without a prefix will be automatically rejected.

### 4.2 Automated Sandbox Validation
Every PR submitted to `unistack-community` must pass an automated CI/CD pipeline before a human review. The CI pipeline will:
1. Verify the `package.yml` format.
2. Ensure download URLs use HTTPS and checksums match.
3. Perform a sandbox installation test inside an isolated ephemeral Docker container to detect malicious scripts or unexpected system modifications.

### 4.3 Contributor License Agreement (CLA)
To ensure commercial compliance and legal safety, all external contributors MUST sign the CLA via an automated platform (e.g., CLA Assistant) before their PR can be merged.

## 5. Client Resolution Strategy

The UniStack CLI is designed to support multiple registry sources simultaneously. To prevent dependency confusion attacks (where a malicious community package tries to overwrite a core package using a higher version number), the client implements a strict priority-based resolution strategy:

1. **Highest Priority**: `unistack-core`
2. **Medium Priority**: `unistack-community`
3. **Lowest Priority**: Custom Registries (unless manually overridden by the user)

When a package name conflict occurs, the client will strictly resolve to the package from the registry with the highest priority, guaranteeing that core infrastructure remains untampered.
