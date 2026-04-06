# Security Policy

## Supported Versions

`marchat` is currently at **v0.11.0-beta.2**.  
All security updates and fixes are applied to the `main` branch.

| Version            | Supported |
|--------------------|-----------|
| v0.11.x (`main`)   | Yes       |
| v0.10.x            | No        |
| v0.9.x             | No        |
| Earlier versions   | No        |

---

## Reporting a Vulnerability

If you discover a security vulnerability in `marchat`, please do **not** open a public GitHub issue.

Instead, report it privately through one of the following:

- GitHub: [Private Security Advisory](https://github.com/Cod-e-Codes/marchat/security/advisories/new)  
- Email: [cod.e.codes.dev@gmail.com](mailto:cod.e.codes.dev@gmail.com)

> [!IMPORTANT]  
> Your report will only be visible to maintainers and select collaborators until a fix is released.

---

## Disclosure Process

We aim to respond to reports within **2–3 business days**.  
If confirmed:  
1. We'll investigate and prepare a fix in a private branch or fork.  
2. We may coordinate with you on the disclosure timeline.  
3. We'll publish a GitHub Security Advisory and credit contributors (if applicable).

---

## Requesting a CVE

If the issue meets CVE criteria and you want one assigned, let us know in your report.  
GitHub is a CVE Numbering Authority and can issue one after disclosure.

---

## Scope

This policy applies only to the official `marchat` codebase:  
[`https://github.com/Cod-e-Codes/marchat`](https://github.com/Cod-e-Codes/marchat)

It **does not cover**:  
- Misconfigurations in self-hosted deployments  
- Issues caused by modified forks or downstream packaging  
- General UX/UI feedback or feature requests

### Diagnostics output

The `-doctor` / `-doctor-json` commands print masked values for sensitive `MARCHAT_*` variables; avoid sharing raw process environment dumps alongside doctor output. For air-gapped hosts, set `MARCHAT_DOCTOR_NO_NETWORK=1` so doctor does not call the GitHub API.

### Indirect Go modules and vulnerability scanners

Dependabot may flag **transitive** dependencies that do not expose reachable vulnerable APIs in marchat. For example, **CVE-2026-26958** ([GHSA-fw7p-63qq-7hpr](https://github.com/advisories/GHSA-fw7p-63qq-7hpr)) affects **`filippo.io/edwards25519`** before **v1.1.1** (`MultiScalarMult` receiver initialization). marchat does not use that API; the advisory notes many consumers (including typical **`github.com/go-sql-driver/mysql`** usage) are unaffected. The module is still pinned at **v1.1.1** on **`main`** to pick up the fix. For reachability, run **`govulncheck ./...`** against your build.

---

## Questions?

For general bugs, please use:  
- [GitHub Issues](https://github.com/Cod-e-Codes/marchat/issues)

For feature requests or questions, please use:  
- [GitHub Discussions](https://github.com/Cod-e-Codes/marchat/discussions)

Thank you for helping keep `marchat` and its users safe!
