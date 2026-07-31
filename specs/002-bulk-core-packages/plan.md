# Bulk Build UniStack Core Packages Strategy Plan

Based on your in-depth analysis and feedback, we will build a fully UniStack-compliant package production pipeline (Harvester) that operates on a **long-term, periodic, and intelligent** basis. To ensure engineering quality, we will adopt a **modular implementation** strategy.

## Core Workflow Design (6-Step Strategy)

### 1. Strict Adherence to UniStack Specification Documents
All `package.yml` files and directory structures produced by the auto-generation engine MUST 100% comply with existing UniStack specifications. The underlying logic is entirely judged intelligently by **UniStack** (Installer); the generation script does not interfere with the underlying installation mechanisms.

### 2. Machine Extraction: Repology Package Mapping Data
To avoid API rate limits, the Harvester will download the daily PostgreSQL database dump (`repology-database-dump-latest.sql.zst`) from Repology.org. Instead of spinning up a heavyweight PostgreSQL instance, we will implement a custom Go parser (Scheme B) to extract the necessary table data directly from the uncompressed raw SQL stream via regex/string matching. This ensures maximum performance and zero external database dependencies.

### 3. Data Complement: Service Data Dictionary and Service Name Discovery
**Addressing service data extraction coverage (Debian, RHEL, Alpine, etc.):**
* **Four Major Faction Coverage Strategy**: Modern Linux package management and system service naming are generally divided into three major factions. By writing scripts to parse repository metadata of the following typical representatives, we can cover 99% of scenarios:
  1. **Debian/Ubuntu Faction (DEB)**: Represents `apt` and `systemd` service architectures (e.g., `apache2`).
  2. **RHEL/Fedora Faction (RPM)**: Represents `dnf/yum` and `systemd` service architectures (e.g., `httpd`).
  3. **Alpine Faction (APK)**: Represents minimalist images, using `OpenRC / init.d` architectures.
  4. **Arch Linux Faction (Optional Extension)**: As a supplementary dictionary.
* **Manual Fallback Mechanism**: Automatically parsing the `.service` filenames of the above distributions helps construct 90% of the initial service dictionary. The remaining 10% edge cases are manually verified and handled by a locally persisted `services_dict.yml` dictionary.

### 4. Manual Maintenance: Scenario-based Templates
**Strict Adherence to UniStack App Mechanism**: The core base templates maintained manually in the generator's `templates/` directory will be written **completely based on the existing app mechanism and official example packages in UniStack**. This ensures that the generated structures are natively compatible with the underlying intelligent installation framework.
* `template_pkg_only.yml.tpl`: Suitable for pure tool-based software.
* `template_pkg_with_service.yml.tpl`: Suitable for software with background services.
(These templates will be as minimal as possible, delegating logic to the underlying App intelligent framework).

### 5. Intelligent Rendering and Automatic Merging
* Combine Repology data + service dictionary data to automatically select the correct template.
* **Output Generation**: No organizational namespace; directly output to `UniStack-Core/packages/<pkg_name>/`.

### 6. Long-term Maintenance, Anti-Overwrite, and Automated CI/CD Validation
* **Periodic Intelligent Pipeline**: Encapsulated as GitHub Actions, triggered periodically every week.
* **Absolutely Reliable AST Anti-Overwrite Mechanism**: Utilizing Go's AST YAML parsing engine to **absolutely ensure** that direct manual modifications to `package.yml` (including service data) and the `tasks/` directory possess the highest preservation privilege.
* **Safe Auto-Merge After CI Validation**: After generating changes and submitting a PR, Auto-Merge is only permitted after CI sandbox validation verifies successful installation and execution. Otherwise, it is intercepted and handed over to manual processing.

---

## Phased Implementation Roadmap

To proceed steadily, the development of the entire Harvester engine will be implemented in the following modules:

* **Module 1: Core Template Design & UniStack App Mechanism Alignment**
  Write robust `package.yml` templates and `tasks/main.yml` (if needed), and manually verify them within the UniStack engine.
* **Module 2: Automated Data Fetchers**
  Implement `Repology` data fetching and cleansing logic; implement `Debian/RHEL/Alpine` service package structure parsing to generate the initial service dictionary.
* **Module 3: Intelligent Merge Engine (AST YAML Merger)**
  Develop core Go logic: Based on collected data and base templates, generate/update local `package.yml` files, ensuring safe merging via AST without overwriting manual changes.
* **Module 4: CI/CD & Auto-Merge Automated Pipeline**
  Write GitHub Actions workflows, including scheduled triggers, change detection, PR generation, sandbox validation, and final automatic merging.
