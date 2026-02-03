# Security Policy

## Supported Versions

As this project is in early development (v0.x), only the latest release is officially supported with security updates.

| Version | Supported |
| ------- | ----------- |
| v0.1.x  | ✔ |
| < 0.1.0 | ✖ |

## Reporting a Vulnerability

`workflow` takes its security seriously. If you believe you have found a security vulnerability, please report it as described below.

**Please do not report security vulnerabilities through public GitHub issues.**

### How to Report

Please email **dev.dilute902@passinbox.com** with the subject line `[SECURITY] workflow vulnerability`.

In your email, please include:
1. The specific version of `wf` you are using.
2. A description of the vulnerability.
3. Steps to reproduce the issue (e.g., a malicious TOML file or specific command sequence).

### Review Process

1. You will receive an acknowledgement of receipt of your report within 48 hours.
2. The issue will be investigated and its impact determined.
3. If confirmed, a patch will be released as quickly as possible.
4. Once the patch is released, your contribution will be publicly acknowledged (unless you prefer to remain anonymous).

### Scope

`workflow` is a local execution tool. Specific interests are placed in:
- **Arbitrary Code Execution:** Situations where `wf` executes code not defined in the intended workflow.
- **Privilege Escalation:** If `wf` can be tricked into running commands with higher privileges than the user intended.
- **Data Leakage:** If `wf` logs sensitive environment variables despite configuration to suppress them.

Thank you for helping keep `workflow` safe!