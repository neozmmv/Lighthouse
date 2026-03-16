## DEVELOPMENT!

To get the CLI binary with the version injected, use

```bash
go build -ldflags "-X github.com/neozmmv/Lighthouse/cli/cmd.version=1.0.0" -o lighthouse
```

For development, build with

```bash
go build -o lighthouse
```

GitHub Actions will build on new tag, like "v1.1.2"
`git tag v1.1.2`, `git push origin v1.1.2`
