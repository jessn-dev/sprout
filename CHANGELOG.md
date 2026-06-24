# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **`sprout new` (non-interactive)**: Fully flag-driven generation for scripts and CI (`--name`, `--group`, `--deps`, `--type`, `--git`, ...).
- **Config validation**: Group/Artifact/Package/Name validated before the request — clear error instead of an opaque Initializr `400`.
- **Post-generation help**: Optional `git init` and printed next-steps (`cd ...`, `./mvnw spring-boot:run`).
- **Dependency descriptions**: TUI shows the highlighted dependency's description.
- **Incompatible-dependency guard**: Generating with deps outside the selected Boot version's range requires explicit confirmation.
- **Dynamic version**: `sprout --version` reports the GoReleaser-injected build version.

### Fixed
- **Stale Boot version**: When unknown, `bootVersion` is omitted so start.spring.io applies its own current default (was pinned to a version the API later rejected with `400`).
- **Controller injection**: Sample `@RestController` only injected when Spring Web is present, so web-less projects/modules still compile.
- **Overwrite guard**: Generation aborts instead of silently clobbering an existing directory.
- **Safe folder names**: Output directory sanitized from the artifact ID (no space-broken paths).
- **HTTP timeout**: Project download bounded (30s) with a spinner — a network stall can't hang the CLI.
- **CI**: GitHub Actions Go version bumped to match `go.mod` (1.26).

### Changed
- Docker templates honor the selected Java version, drop the obsolete Compose `version:` key, and name the DB after the project.
- Offline runs fall back to a curated dependency catalogue so the picker still works.

### Added (initial release)
- **Interactive TUI**: Completely rewrote the setup flow using `bubbletea`. It now runs in a 2-column view with live fuzzy-searching for dependencies and compatibility warnings.
- **`sprout quick`**: Added a fast-path subcommand to skip the big TUI and just generate a project from standard defaults with two questions.
- **`sprout explore`**: Added a terminal browser to search and read up on all the different Spring Boot dependencies without leaving your CLI.
- **Submodule Generation**: You can now choose to generate standalone modules. It includes options to automatically strip out the Maven/Gradle wrappers so the code drops cleanly into an existing parent repo.
- **Homebrew Releases**: Configured `.goreleaser.yaml` to automatically publish to a Homebrew Tap for Mac/Linux users.
- **Live Spring Versions**: The tool now dynamically fetches the active Spring Boot versions from the Initializr API instead of relying on hardcoded lists.
- **More Custom Modules**: Added automated setups for PostgreSQL, MySQL, H2, Kafka, Spring Security, Validation, Lombok, and Actuator.
- **Dynamic Docker Compose**: The tool now reads your selected dependencies and automatically spins up the right containers (like Postgres or Kafka) in the generated `docker-compose.yml`.
- **Project Restructure**: Moved everything into a standard Go layout (`cmd/sprout`, `internal/`) to keep it maintainable.
- **Unit Tests**: Added tests for the core template injection engine.
- **Initializr Integration**: Wired up the core engine to pull the base zip payload directly from `start.spring.io`.
- **CI/CD**: Added GitHub Actions to automatically run tests and trigger GoReleaser builds.
