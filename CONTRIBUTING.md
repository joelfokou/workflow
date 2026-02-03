# Contributing to workflow

First off, thank you for considering contributing to `workflow`! It's people like you that make the open-source community such an amazing place to learn, inspire, and create.

`workflow` is an open-source project and would love to receive contributions from the community! There are many ways to contribute, from writing tutorials or examples, reporting bugs, and submitting feature requests, to writing code which can be incorporated into `workflow` itself.


## 🛠 Development Guide

### Prerequisites
- **Go**: You need Go installed (v1.21 or later recommended).
- **Git**: For version control.

### Setting up the environment
1. **Fork the repository** on GitHub.
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/<YOUR-USERNAME>/workflow.git
   cd workflow
   ```
3. **Install dependencies**:
    ```bash
    go mod download
    ```
4. **Build the binary** to ensure everything is working:
    ```bash
    go build -o wf .
    ./wf --version
    ```

### Running Tests
`workflow` takes testing seriously. Before submitting a PR, please ensure all tests pass.
```bash
# Run all tests
go test ./...

# Run tests with race condition detection
go test -race ./...
```


## 🐛 Reporting Bugs
A good bug report shouldn't leave others needing to chase you up for more information. Please try to be as detailed as possible in your report.

Please include:
1. **Version**: Output of `wf --version`.
2. **OS**: Mac, Linux, or Windows?
3. **Reproduction Steps**: A minimal TOML file or command sequence that causes the crash/bug.
4. **Logs**: Output from `wf logs <run-id>` or the error message.


## 💡 Feature Requests
Feature requests are welcome! But take a moment to find out whether your idea fits with the scope and aims of the project. It's up to you to make a strong case to convince the project's developers of the merits of this feature.

**Note on Scope**: `workflow` aims to be a **local-first** orchestrator. Features involving distributed agents, web servers, or kubernetes operators are currently considered out of scope.


## 📥 Pull Request Process
1. **Create a branch** for your feature or fix:
    ```bash
    git checkout -b feat/amazing-new-feature
    ```
    *(It's recommended to use `feat/`, `fix/`, or `docs/` prefixes)*
2. **Write your code.**
    - Follow standard Go idioms (Effective Go).
    - Keep functions small and testable.
3. **Format and Lint.**
    - Your code must be formatted with `gofmt`.
    - Run `go vet ./...` to catch common issues.
4. **Commit your changes.**
    - We encourage **Conventional Commits**.
    - Example: `feat: add retry logic to task executor` or `fix: resolve race condition in database lock`.
5. **Push and Open a PR.**
    - Push to your fork: `git push origin feat/amazing-new-feature`.
    - Open a Pull Request against the `main` branch of `joelfokou/workflow`.

### Code Review Checklist
When reviewing your PR, the following will be checked:
- [ ] Tests for the new functionality (or regression tests for bug fixes).
- [ ] Updated documentation (if you changed CLI flags or TOML syntax).
- [ ] Clean git history (please squash intermediate "wip" commits).


## 🤝 Code of Conduct
This project is committed to providing a friendly, safe and welcoming environment for all. Please be respectful and considerate in your communication.