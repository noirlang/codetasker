# Contributing to CodeTasker

We welcome contributions from the community! Whether you are fixing bugs, improving documentation, or proposing new features, your help is appreciated.

## How to Contribute

### 1. Report Issues
If you find a bug or want to request a feature, please search existing issues first. If it hasn't been reported yet, open a new issue describing:
- Your operating system and environment.
- Steps to reproduce the issue.
- The expected vs. actual behavior.

### 2. Submit Pull Requests
1. **Fork** the repository and create your branch from `main`:
   ```bash
   git checkout -b feature/my-amazing-feature
   ```
2. **Implement** your changes. Make sure to write unit tests if applicable.
3. **Format** and verify your code before committing:
   * **Go (Backend):**
     ```bash
     go fmt ./...
     go test ./...
     ```
   * **TypeScript/React (Frontend):**
     ```bash
     npm run build
     ```
4. **Commit** your changes with clear, descriptive commit messages.
5. **Push** to your fork and submit a Pull Request to the `main` branch of `noirlang/codetasker`.

## Style Guide
- Follow Go coding standards and run `go fmt`.
- Follow React best practices, using TypeScript type checking and modern React hooks.
- Keep UI components clean and follow the established dark theme design system.

## Licensing
By contributing to CodeTasker, you agree that your contributions will be licensed under the GPL-3.0 License.
