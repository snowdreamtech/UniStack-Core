# Feature Specification: Bulk Build UniStack Core Packages

**Feature Branch**: `002-bulk-core-packages`

**Created**: 2026-07-31

**Status**: Draft

**Input**: User description: "Follow the speckit process based on the implementation plan."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Machine Extraction of Mappings and Base Generation (Priority: P1)

The automation tool (Harvester) can periodically download the daily database dump (`.sql.zst`) from Repology.org, parse the raw SQL directly via a custom Go implementation without requiring a PostgreSQL instance, combine it with the service mapping dictionary, and automatically generate standard `package.yml` file structures for various open-source software using Go's template engine.

**Why this priority**: This is the fundamental infrastructure of the entire batch package generation pipeline; without it, "mass production" is impossible.

**Independent Test**:
Process 3 common open-source packages (e.g., `nginx`, `curl`, `htop`) with the script, and verify whether the `spec.packages` field in the generated `package.yml` is correctly mapped across systems.

**Acceptance Scenarios**:

1. **Given** Repology package data and a local service dictionary, **When** executing the `harvester` generation task, **Then** a fully compliant `package.yml` is generated in the `packages/nginx/` directory with the `nginx` service correctly marked.
2. **Given** a service-less tool like `curl`, **When** executing `harvester`, **Then** its `package.yml` lacks the `spec.services` field, utilizing the service-less package template.

---

### User Story 2 - Safe-Merge for Manual Hybrid Editing (Priority: P2)

Humans create `tasks/main.yml` in the generated package directory to implement customized configurations and manually append tags in `package.yml`. When Harvester runs a second time, it only updates the version and does not damage human customizations.

**Why this priority**: This is a core requirement for long-term operations and maintenance. If brute-force overwrites occur every time, the entire project becomes unmaintainable.

**Independent Test**: Add a line of custom content to an already generated package directory, run the update command, and verify that the custom content survives.

**Acceptance Scenarios**:

1. **Given** a `package.yml` with manually edited `spec.services`, **When** Harvester updates the package version, **Then** based on AST merging, the original manual configuration MUST be 100% preserved.
2. **Given** the user manually wrote `tasks/main.yml`, **When** Harvester runs, **Then** modifications to these custom manual files are completely skipped.

---

### User Story 3 - Continuous Integration and Auto-Merge Pipeline (Priority: P3)

The entire process achieves a low-barrier automation: Harvester executes periodically to discover version changes, automatically submits GitHub Pull Requests, and automatically Merges them after passing CI sandbox validation (blocking if installation or startup fails).

**Why this priority**: This is the key closed loop for scalable, long-term unattended operations.

**Independent Test**: Simulate an update generation and submit a PR, observing whether the pipeline can correctly spin up a container sandbox and execute Auto-Merge based on the exit code.

**Acceptance Scenarios**:

1. **Given** a package that generates a dependency update, **When** the PR triggers GitHub Actions, **Then** after a successful `unistack install` validation inside the container, the PR is automatically merged.
2. **Given** validation detects a package name conflict or startup failure, **When** the sandbox exits with a failure, **Then** the PR remains open, awaiting human intervention.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST implement a unified template rendering mechanism based on Go `text/template`, and templates must align with the UniStack core App installation mechanism.
- **FR-002**: System MUST be able to parse service data from mainstream distributions (Debian, RHEL, Alpine, etc.) to complement the manual `services_dict.yml`.
- **FR-003**: System MUST utilize `gopkg.in/yaml.v3` (or similar AST-level parsers) to accomplish idempotent updates of package information, prohibiting brute-force text-level overwrites.
- **FR-004**: System MUST create package directories directly in `packages/<pkg_name>/` without adding an organizational namespace.
- **FR-005**: Script logic MUST NOT replace `UniStack Core` decisions on `systemd/init` selection; it is only responsible for extracting and injecting service names.
- **FR-006**: System MUST obtain Repology data by downloading the bulk `.sql.zst` dump and parsing it in-memory via Go, completely avoiding the Repology REST API and any external PostgreSQL database engine.

### Key Entities

- **Harvester CLI**: A tool engine written in Go for fetching, cleansing, and rendering package description files.
- **Services Dictionary (`services_dict.yml`)**: Acts as a local source to correct or supplement service data that Repology cannot provide.
- **UniStack Core Packages**: The target directory structure after rendering (e.g., `packages/nginx/package.yml`).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Initially, the number of packages capable of passing validation and being automatically generated at once MUST exceed 500+.
- **SC-002**: The execution time of periodic automated tasks (Actions) should be stable and support incremental idempotent builds.
- **SC-003**: Content manually modified in `package.yml` retains a 100% survival rate after 100 cycles of generation.

## Assumptions

- Developers need to rely on the GitHub Actions environment to successfully run basic container-level "installation/startup testing".
- The vast majority of common cross-platform mappings can still obtain reliable support from the Repology API or by directly parsing source data.
