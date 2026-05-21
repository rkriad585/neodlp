# Contributing to NeoDLP

First off, thanks for taking the time to contribute!

## How to contribute

### Report bugs

Open an issue at https://github.com/rkriad585/neodlp/issues with:

- A clear, descriptive title
- Steps to reproduce
- Expected vs actual behavior
- Your OS and NeoDLP version (`neodlp version`)
- The URL you tried to download (if applicable)

### Suggest features

Open an issue describing:

- What you want to add or change
- Why it's useful
- Any implementation ideas

### Submit code

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Run `go vet neodlp/...` and ensure no errors
5. Run `go build -o neodlp .` and verify it compiles
6. Commit with a clear message
7. Push and open a pull request

### Code style

- Follow standard Go formatting (`go fmt`)
- Use existing patterns from the codebase
- Keep functions focused and small
- Write tests for new functionality where appropriate
- No comments unless necessary for clarity

### Pull request checklist

- [ ] Code compiles without errors
- [ ] `go vet` passes
- [ ] Commit messages are clear and descriptive
- [ ] PR description explains what and why

## Development setup

```bash
git clone https://github.com/rkriad585/neodlp.git
cd neodlp
go mod download
go build -o neodlp .
```

## Questions?

Open an issue or reach out to rkriad585@gmail.com.
