# Contributing to marchat

Thank you for your interest in contributing! This guide explains how to contribute effectively.

## Types of Contributions

### Bug Reports (GitHub Issues)
- Use the issue tracker for bugs and problems only
- Include:
  - Clear description of the problem
  - Steps to reproduce
  - Expected vs actual behavior
  - Your OS and marchat version
  - Any relevant logs or screenshots

### Ideas and Questions (GitHub Discussions)
- Start a discussion for:
  - Feature requests and suggestions
  - Questions about setup or usage
  - General feedback and ideas
  - Sharing your experience
  - Showing off customizations

### Code Contributions

1. **Fork & Clone**
   - Fork the repo on GitHub
   - Clone your fork locally
   - Keep your fork in sync with upstream

2. **Create a Branch**
   - Branch from `main`
   - Use a clear, descriptive name
   - One feature/fix per branch

3. **Code Style**
   - Use `gofmt` or `go fmt`
   - Follow idiomatic Go patterns
   - Keep code readable and well-commented
   - Write clear commit messages

4. **Testing**
   - Add tests for new features
   - Ensure all tests pass locally
   - Run `go test ./...`

5. **Submit Pull Request**
   - Push to your fork
   - Open a PR against `main`
   - Describe your changes clearly
   - Link related issues

## Automation

- GitHub Actions runs CI on all PRs
- Tests must pass before merge
- Dependabot handles dependency updates
- Do not manually update dependencies unless needed for a fix

## Communication

- Be respectful and constructive
- Follow the [Code of Conduct](CODE_OF_CONDUCT.md)
- All contributions are welcome; no idea is too small!

---

Thank you for helping make marchat better! 