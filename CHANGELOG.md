# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-06-24

### Added
- **Future-Proof Java Support**: Added official offline fallback support for Java 25 and Java 26, and updated the generator to dynamically enable Virtual Threads (`spring.threads.virtual.enabled=true`) for *all* versions `>= 21` (instead of relying on a hardcoded list).
- **Architectural Defaults (Java 21+)**: Automatically enables Virtual Threads (`spring.threads.virtual.enabled=true`) for modern Java concurrency out of the box.
- **Enterprise Package Scaffolding**: Automatically creates a full, standard Java enterprise project layout (`domain`, `service`, `repository`, `web`, `config`, `dto`, `exception`) in both `src/main` and `src/test` to ensure scalability and ease of onboarding.
- **Strict Package Boundaries**: Injected source files are now strictly separated into bounded contexts (`.web`, `.config`) to prevent the "Big Ball of Mud" antipattern and enforce clean architecture.
- **API Contract First**: Automatically injects `springdoc-openapi-starter-webmvc-ui` into the build files (Maven/Gradle) so every web project instantly boots with a live Swagger UI.
- **Testcontainers Ready**: If a database is selected, Testcontainers and JUnit dependencies are automatically injected into the build files for real integration testing.
- **Production-Grade Config Split**: Split application properties into `application.properties` (safe, env-overridable defaults) and `application-dev.properties` (ddl-auto=update, show-sql, verbose actuator), ensuring dev settings never leak to prod.
- **Env-Overridable Credentials**: Database and Redis templates now use `${DB_URL:...}`, `${DB_USERNAME:user}`, and `${REDIS_HOST:localhost}` by default, seamlessly matching the generated `docker-compose.yml`.
- **Samples Opt-Out**: Added `--no-samples` flag to skip generating the default `HelloController` while still keeping necessary configs like Security/Redis intact.
- **Kotlin Parity**: The generator now correctly translates and writes `HelloController`, `SecurityConfig`, and `RedisConfig` into `.kt` files for Kotlin projects.

### Fixed
- **Submodule CI Clutter**: The `Create a module immediately` workflow now properly strips the nested `.github` folder out of the generated submodule, preventing useless nested CI pipelines that GitHub Actions would ignore anyway.
- **Gradle Docker Build**: The Gradle `Dockerfile` template now correctly targets the runnable jar (via `bootJar` excluding `-plain.jar`), fixing broken Docker builds.
- **Security 401 Lockout**: Injected `SecurityConfig` for the security profile now properly permits `/api/hello` and `/actuator/health` while authenticating the rest, fixing the unreachable sample endpoints.
- **DB Name Sanitization**: Replaced hyphens with underscores in database names (e.g., `secure-api` → `secure_api`) in JDBC URLs and compose files to generate valid SQL identifiers.
- **Stale Boot version**: When unknown, `bootVersion` is omitted so start.spring.io applies its own current default (was pinned to a version the API later rejected with `400`).
- **Controller injection**: Sample `@RestController` only injected when Spring Web is present, so web-less projects/modules still compile.
- **Overwrite guard**: Generation aborts instead of silently clobbering an existing directory.
- **Safe folder names**: Output directory sanitized from the artifact ID (no space-broken paths).
- **HTTP timeout**: Project download bounded (30s) with a spinner — a network stall can't hang the CLI.

### Changed
- **Documentation Overhaul**: Completely rewrote the README to highlight enterprise architecture features, add execution and shell-completion instructions, and include a personal story about building the project to learn Go.
- Docker templates honor the selected Java version, drop the obsolete Compose `version:` key, and name the DB after the project.
- Offline runs fall back to a curated dependency catalogue so the picker still works.
- CI: GitHub Actions Go version bumped to match `go.mod` (1.26).

## [0.1.0] - 2026-06-23

### Added
- **`sprout new` (non-interactive)**: Fully flag-driven generation for scripts and CI (`--name`, `--group`, `--deps`, `--type`, `--git`, ...).
- **Config validation**: Group/Artifact/Package/Name validated before the request — clear error instead of an opaque Initializr `400`.
- **Post-generation help**: Optional `git init` and printed next-steps (`cd ...`, `./mvnw spring-boot:run`).
- **Dependency descriptions**: TUI shows the highlighted dependency's description.
- **Incompatible-dependency guard**: Generating with deps outside the selected Boot version's range requires explicit confirmation.
- **Dynamic version**: `sprout --version` reports the GoReleaser-injected build version.
- **Interactive TUI**: Completely rewrote the setup flow using `bubbletea`. It now runs in a 2-column view with live fuzzy-searching for dependencies and compatibility warnings.
- **`sprout quick`**: Added a fast-path subcommand to skip the big TUI and just generate a project from standard defaults with two questions.
- **`sprout explore`**: Added a terminal browser to search and read up on all the different Spring Boot dependencies without leaving your CLI.
- **Dynamic Java Versions**: The TUI now pulls live Java versions directly from the Initializr metadata instead of relying on a hardcoded static list.
- **Submodule Generation**: You can now choose to generate standalone modules. It includes options to automatically strip out the Maven/Gradle wrappers so the code drops cleanly into an existing parent repo.
- **Homebrew Releases**: Configured `.goreleaser.yaml` to automatically publish to a Homebrew Tap for Mac/Linux users.
- **Live Spring Versions**: The tool now dynamically fetches the active Spring Boot versions from the Initializr API instead of relying on hardcoded lists.
- **More Custom Modules**: Added automated setups for PostgreSQL, MySQL, H2, Kafka, Spring Security, Validation, Lombok, and Actuator.
- **Dynamic Docker Compose**: The tool now reads your selected dependencies and automatically spins up the right containers (like Postgres or Kafka) in the generated `docker-compose.yml`.
- **Project Restructure**: Moved everything into a standard Go layout (`cmd/sprout`, `internal/`) to keep it maintainable.
- **Unit Tests**: Added tests for the core template injection engine.
- **Initializr Integration**: Wired up the core engine to pull the base zip payload directly from `start.spring.io`.
- **CI/CD**: Added GitHub Actions to automatically run tests and trigger GoReleaser builds.
