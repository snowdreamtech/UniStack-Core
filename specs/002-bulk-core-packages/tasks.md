---
description: "Task list for Bulk Build UniStack Core Packages"
---

# Tasks: Bulk Build UniStack Core Packages

**Input**: Design documents from `/specs/002-bulk-core-packages/`

**Prerequisites**: plan.md, spec.md

**Organization**: Tasks are grouped by implementation modules and user stories to enable independent implementation and testing of each phase.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)

---

## Phase 1: Setup & Foundational (Module 1 - Templates)

**Purpose**: Establish the core templates ensuring 100% compatibility with UniStack App mechanisms.

- [ ] T001 Initialize the `scripts/harvester/` Go project structure
- [ ] T002 Create the `templates/template_pkg_only.yml.tpl` template based on UniStack examples
- [ ] T003 Create the `templates/template_pkg_with_service.yml.tpl` template based on UniStack examples
- [ ] T004 Manually render and verify these templates using the UniStack installer to guarantee structural correctness

---

## Phase 2: Foundational (Module 2 - Data Fetchers)

**Purpose**: Build the fetchers for external mapping data and bootstrap the service dictionary.

- [ ] T005 [P] Implement `fetcher_repology.go` to download the `repology-database-dump-latest.sql.zst` dump, decompress it using Go's `zstd`, and parse the raw SQL text directly to extract package mappings (without using a DB instance)
- [ ] T006 [P] Implement `fetcher_services.go` to scan Debian/RHEL/Alpine repository data for `.service` or `init.d` filenames
- [ ] T007 Combine fetchers to output the initial `services_dict.yml` (saving it to `scripts/harvester/`)

**Checkpoint**: Foundation ready - we now have the templates and the raw data sources required for generation.

---

## Phase 3: User Story 1 - Machine Extraction & Base Generation (Priority: P1) 🎯 MVP

**Goal**: Automatically generate standard `package.yml` file structures by combining Repology data and the service dictionary.

**Independent Test**: Run Harvester for 3 common packages (`nginx`, `curl`, `htop`) and verify output directories in `packages/`.

### Implementation for User Story 1

- [ ] T008 [US1] Load and parse the local `services_dict.yml` in the Harvester engine
- [ ] T009 [US1] Implement data merging logic: match Repology package names with their corresponding service names (if any)
- [ ] T010 [US1] Implement template rendering logic using Go's `text/template` based on whether a service exists
- [ ] T011 [US1] Output the generated `package.yml` files directly into `packages/<pkg_name>/`

**Checkpoint**: Harvester can perform a one-time massive generation of clean packages.

---

## Phase 4: User Story 2 - Safe-Merge for Manual Hybrid Editing (Priority: P2)

**Goal**: Ensure subsequent runs only update versions/mappings without destroying manually added service definitions or custom tasks.

**Independent Test**: Modify a generated `package.yml` manually, run Harvester again, and verify the manual changes survive.

### Implementation for User Story 2

- [ ] T012 [US2] Integrate `gopkg.in/yaml.v3` for AST-level parsing of existing `package.yml` files
- [ ] T013 [US2] Implement idempotent merge logic: update `spec.packages` and `version`, but strictly preserve human modifications to `spec.services`
- [ ] T014 [US2] Implement directory exclusion rules: explicitly ignore and preserve any existing `tasks/` and `templates/` directories within the package folder

**Checkpoint**: Safe-merge is robust; humans and machines can co-author the package definitions safely.

---

## Phase 5: User Story 3 - Continuous Integration & Auto-Merge Pipeline (Priority: P3)

**Goal**: Establish a periodic GitHub Actions workflow that automatically generates updates, tests them in a sandbox, and auto-merges PRs.

**Independent Test**: Trigger the workflow manually and observe the PR generation, CI status, and auto-merge behavior.

### Implementation for User Story 3

- [ ] T015 [US3] Create `.github/workflows/harvester-cron.yml` to trigger the Harvester script weekly
- [ ] T016 [US3] Configure the workflow to detect git diffs and automatically create a Pull Request via GitHub CLI
- [ ] T017 [US3] Add a CI validation job to the workflow: start a sandbox container and run `unistack install <changed_packages>`
- [ ] T018 [US3] Configure branch protection rules or GitHub Actions logic to automatically merge the PR *only* if the validation job passes

**Checkpoint**: The entire long-term, unattended pipeline is closed and fully functional.
