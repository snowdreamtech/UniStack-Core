# Feature Specification: Community Registry Architecture

**Feature Branch**: `001-community-registry-architecture`

**Created**: 2026-07-26

**Status**: Draft

**Input**: User description: "Summarize the discussion about the community library architecture into a document and update it into the project"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Submit Community Package (Priority: P1)

Developers want to submit their own developed packages to the UniStack community registry so that other users can easily search and install them via the client.

**Why this priority**: Being able to submit packages is the most fundamental part of building a community ecosystem. Without it, there is no Community ecosystem.

**Independent Test**: This can be independently tested by submitting a PR containing a valid `package.yml`, and observing the automated validation scripts and the final index generation after merging.

**Acceptance Scenarios**:

1. **Given** a developer has prepared a third-party package configuration, **When** the developer submits a PR to the `unistack-community` repository, **Then** the CI will verify if the namespace prefix is compliant and validate the package installation in a sandbox.
2. **Given** the PR passes validation, **When** the Maintainer merges the PR, **Then** the CI automatically builds the source code into a binary package, updates index files like `packages.db`, and pushes them to the CDN.

---

### User Story 2 - Add Enterprise/Personal Custom Registry (Priority: P2)

Enterprise users or individual developers want to add internal private packages or non-open-source tools as independent registries to the client, in order to meet specific enterprise security and commercial distribution needs.

**Why this priority**: It compensates for the shortcomings of the centralized community registry in private deployments and the distribution of sensitive/closed-source code.

**Independent Test**: This can be tested by setting up a custom registry that meets the specifications and configuring it in the client, then verifying that the client can parse and download packages from this registry.

**Acceptance Scenarios**:

1. **Given** a user has a custom registry URL, **When** the user executes an add command (e.g., `unistack registry add my-company https://...`), **Then** the client can add this registry to the local Registry configuration file.
2. **Given** the client is configured with multiple registries, **When** an installation command is executed and a package name conflict occurs, **Then** the client resolves it according to priority, returning the Core registry first, effectively preventing malicious overwriting by community packages.

### Edge Cases

- **Namesquatting and Dependency Confusion**: When a third-party submitted PR attempts to publish a package without a namespace prefix (e.g., trying to occupy a core registry name like `nginx`), the CI system should intercept and reject the build directly.
- **Storage Source Crash and Contamination**: If the CDN distribution system fails or is maliciously contaminated, how can operations recover it? (Simply re-trigger the CI of the main repository to rebuild everything from the single source of truth—the GitHub source code—and overwrite the push, ensuring code is tamper-proof).
- **Malicious Script Interception**: What if a community package contains malicious mining scripts or attempts to tamper with critical system directories during sandbox installation validation? (The CI sandbox environment test fails immediately, preventing the Merge and alerting the Reviewer).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The community ecosystem MUST adopt a hybrid architecture ("Centralized PR-driven as primary, Distributed Custom Registry as auxiliary"). All official community package source configurations are hosted in a single GitHub repository (`unistack-community`).
- **FR-002**: The system MUST implement strict Namespacing control. Community registry package names MUST mandatorily include the author or organization namespace prefix (e.g., `snowdreamtech/nginx`). The globally unique and prefix-free top-level namespace is reserved exclusively for the Core registry.
- **FR-003**: A GitOps automatic distribution mechanism MUST be utilized to achieve separation of source code and distribution. After a PR is merged, the system MUST push only the final built package files (e.g., `.tar.gz`) and `repodata` indexes to the CDN or static object storage. The GitHub code repository does NOT host binary files.
- **FR-004**: The system MUST provide a CI pipeline (e.g., GitHub Actions) to automatically perform format validation, malicious code scanning, and automated package installation testing in an isolated container (sandbox) before PR merging.
- **FR-005**: The client MUST support managing multiple registry sources (Registries) and implement a strict priority-based fallback mechanism (the Core registry enjoys the highest priority and is absolutely never overwritten by lower-priority community packages).
- **FR-006**: Before accepting code and PR merges, the system MUST force external contributors to sign a CLA (Contributor License Agreement) via an automated platform to ensure commercial compliance.

### Key Entities

- **Source Code and Configuration File (Manifest / package.yml)**: A plain text configuration describing package metadata, dependencies, and installation actions. It is the only medium hosted by the Git repository.
- **Registry Index (packages.db)**: An index list containing metadata and Hashes of all available packages in the repository, deployed in static storage for client retrieval and querying.
- **Namespace**: A prefix identifier used to isolate the identity of community package authors and prevent homoglyph/namesquatting attacks (e.g., `[org_name]/[pkg_name]`).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: "Zero-Intervention Machine Audit Rate": Newly submitted standard packages can automatically complete security scanning and sandbox loading without manual intervention, with the pass/fail status fed back within 5 minutes after PR submission.
- **SC-002**: "GitOps Release Duration": After the main branch is merged, the entire go-live process from building the new package and signing it, to updating the `packages.db` CDN nodes takes less than 10 minutes.
- **SC-003**: "Zero Dependency Confusion": The system can 100% automatically intercept and reject all requests attempting to publish community packages without a prefix or attempting to overwrite core registry packages with higher version, identically named community packages.
- **SC-004**: "Core Lightweighting": The main distribution Git repository size is kept under 50MB, completely eliminating binary burden and achieving second-level `git clone` speeds.

## Assumptions

- **Infrastructure**: The static storage and CDN services (e.g., Cloudflare R2, GitHub Releases) relied upon for distribution can withstand high concurrent pull requests at a controllable low cost.
- **Toolchain Readiness**: The UniStack Builder tool can be correctly invoked by GitHub Actions to generate `packages.db`.
- **Compliance Tools**: The CLA verification mechanism can be integrated via mature GitHub Actions available in the market (e.g., CLA Assistant).
