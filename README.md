```text
  ███████╗ ██████╗  ██████╗   ██████╗  ██╗   ██╗ ████████╗
  ██╔════╝ ██╔══██╗ ██╔══██╗ ██╔═══██╗ ██║   ██║ ╚══██╔══╝
  ███████╗ ██████╔╝ ██████╔╝ ██║   ██║ ██║   ██║    ██║   
  ╚════██║ ██╔═══╝  ██╔══██╗ ██║   ██║ ██║   ██║    ██║   
  ███████║ ██║      ██║  ██║ ╚██████╔╝ ╚██████╔╝    ██║   
  ╚══════╝ ╚═╝      ╚═╝  ╚═╝  ╚═════╝   ╚═════╝     ╚═╝   
```

# 🌱 Sprout - The Custom Spring Initializer

Hey there! 👋 Welcome to **Sprout**.

If you've ever started a Spring Boot project, you know the drill: head to `start.spring.io`, pick your dependencies, download the zip, extract it, and *then* spend the next hour configuring Docker, Redis, databases, and writing boilerplate.

I built Sprout because I was tired of doing that over and over again. I wanted a tool that didn't just grab the raw Spring dependencies, but instantly injected my own opinionated templates (like a pre-configured `docker-compose.yml`, custom database configs, and standard security setups) right into the generated codebase. 

### 🤔 Why use Sprout over the official CLI?
The official Spring CLI is great, but it leaves you hanging when it comes to the "glue" code. 

With Sprout, you get a fast terminal UI that asks you what you need. Then, it:
1. Hits the Spring Initializr API to get the base project.
2. Extracts it locally.
3. **Automatically injects custom templates** (like Dockerfiles, Redis configurations, etc.) so your app is actually ready to run.

### 🚀 Getting Started

**Installation**

*For macOS / Linux (Homebrew):*
```bash
brew install jessn-dev/tap/sprout
```

*For Go Developers:*
```bash
go install github.com/jessn-dev/sprout/cmd/sprout@latest
```
*(Or grab the latest binary from the [Releases](https://github.com/jessn-dev/sprout/releases) page!)*

**Usage**

Sprout offers several commands depending on what you need:

1. **The Main Setup (`sprout`)**
   Just run `sprout` in your terminal. This opens the main TUI where you can set up your project metadata and search through live Spring Boot dependencies. You can generate a full application or just a standalone module (it even has options to strip out `mvnw`/`gradlew` so it nests cleanly into your existing repo).

2. **Fast Setup (`sprout quick`)**
   If you just need a standard project right now and don't want to dig through menus, run `sprout quick`. It asks you for a project name and what type of template you want (Standard, Cloud, or Security) and generates it using standard defaults (Java 21, Maven, latest Boot version).

3. **Ecosystem Explorer (`sprout explore`)**
   If you want to read up on what a specific Spring dependency actually does, run `sprout explore`. It opens a terminal browser where you can search and read descriptions for everything on `start.spring.io`.

4. **Non-Interactive / CI (`sprout new`)**
   Fully flag-driven, no prompts — perfect for scripts and CI:
   ```bash
   sprout new --name shop-api --group com.acme --deps web,data-jpa,docker --git
   sprout new --name api --type security --java 21
   ```
   Run `sprout new --help` for every flag. Coordinates are validated up front, so you get a clear error instead of an opaque Initializr `400`.

Check your installed version any time with `sprout --version`.

### 🤝 Let's Build This Together
I built the first version of this to solve my own annoying workflow problems, but I'd love for this to become a genuinely useful starting point for other Spring developers.

If you have a better way of setting up Neo4j, RabbitMQ, Spring Security, or you want to add templates for your own tools, **please contribute!** Fork the repo, add your injection templates in `internal/generator`, and submit a PR. 

Let's spend less time writing boilerplate and more time building the actual app. Happy sprouting! 🌱
