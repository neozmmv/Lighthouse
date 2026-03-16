## DEVELOPMENT!

To get the CLI binary with the version injected, use

```bash
go build -ldflags "-X github.com/neozmmv/Lighthouse/cli/cmd.version=1.0.0" -o lighthouse
```

For development, build with

```bash
go build -o lighthouse
```
