# Contributing to Sprout 🌱

First off, thank you for considering contributing to Sprout! It's people like you that make Sprout such a great tool for the Spring Boot community.

Since this project was built partly as a way to learn Go, we are extremely welcoming to beginners, seasoned Gophers, and Java Architects alike!

## How Can I Contribute?

### 1. Adding New Templates
Sprout generates opinionated templates (like Dockerfiles, Security Configs, K8s manifests). If you have a better way of setting up Kafka, Neo4j, RabbitMQ, or anything else:
- Fork the repository.
- Navigate to `internal/generator/templates.go`.
- Add your injection logic! We use standard Go strings and basic I/O to inject files.

### 2. Reporting Bugs
If you run into an issue, please check the existing issues to see if it has already been reported. If not, open a new issue using the **Bug Report** template.

### 3. Suggesting Enhancements
Have an idea for a feature? We'd love to hear it! Open a new issue using the **Feature Request** template.

## Development Setup

1. Make sure you have Go 1.22+ installed.
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR-USERNAME/sprout.git
   cd sprout
   ```
3. Run the CLI locally:
   ```bash
   go run ./cmd/sprout quick
   ```
4. Run the tests:
   ```bash
   go test ./...
   ```

## Pull Request Process

1. Ensure any install or build dependencies are removed before the end of the layer when doing a build.
2. Update the README.md with details of changes to the interface, this includes new environment variables, exposed ports, useful file locations and container parameters.
3. You may merge the Pull Request in once you have the sign-off of the maintainer.
