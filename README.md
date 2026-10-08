# Runivra

Runivra sets up Odoo projects from one command. It clones the sources, writes the Docker files, starts the stack and tells you when Odoo is ready.

It is an independent tool for working with Odoo and is not an Odoo S.A. product.

## Status

Early development. Development setups work end to end. Staging and production setups create the folder layout only; their configuration files are not generated yet.

## Requirements

- Git
- Docker with the Compose plugin (Runivra offers to install it when missing)
- Go 1.26 or newer, to build from source

## Install

There are no packaged releases yet, so Runivra is built from source. Installing through package managers such as `apt` is planned; see [Roadmap](#roadmap).

### Linux and macOS

```
git clone https://github.com/GloriousPurposeVariant/runivra.git
cd runivra
go build -o runivra ./cmd/runivra
sudo mv runivra /usr/local/bin/
```

The last line puts the program where your shell can find it. To try it without installing, skip that line and run it as `./runivra` from the repository folder.

### Windows (PowerShell)

```
git clone https://github.com/GloriousPurposeVariant/runivra.git
cd runivra
go build -o runivra.exe ./cmd/runivra
```

Run it as `.\runivra.exe` from the repository folder. To use it from anywhere, move `runivra.exe` into a folder that is on your `PATH`.

### Any system, with Go's own installer

```
go install github.com/GloriousPurposeVariant/runivra/cmd/runivra@latest
```

This builds the program into Go's `bin` folder (`~/go/bin`, or `%USERPROFILE%\go\bin` on Windows). Add that folder to your `PATH` if it is not there already.

### Platform support

Runivra is developed and used day to day on Windows. On Linux, the automated tests run on every change. A full setup run has not yet been verified on Linux or macOS, so expect rough edges there and please report them.

## Use

The examples below assume `runivra` is on your `PATH`. Otherwise use `./runivra` on Linux and macOS, or `.\runivra.exe` on Windows.

Run the wizard and answer the questions:

```
runivra setup
```

Or give everything as options.

Linux and macOS:

```
runivra setup --env dev --version 20.0 --path ~/projects --name shop --start
```

Windows:

```
runivra setup --env dev --version 20.0 --path C:\projects --name shop --start
```

Run `runivra setup --help` for every option.

To keep a token out of your shell history, set it as an environment variable first.

Linux and macOS:

```
export RUNIVRA_ENTERPRISE_TOKEN=your-token
```

Windows:

```
$env:RUNIVRA_ENTERPRISE_TOKEN = "your-token"
```

### What a development setup creates

```
shop/
  (Odoo source, cloned from the chosen version branch)
  enterprise/          Odoo Enterprise, or empty
  custom/              your addons, or empty
  odoo.conf
  Dockerfile
  docker-compose.yml
```

Odoo then runs at `http://localhost:8069`, or the port you chose. The master password for the database screen is `dev-admin`.

Supported Odoo versions: 16.0 to 20.0.

### Enterprise and private repositories

Odoo Enterprise can be cloned with a Git token or copied from a local folder. A token typed into the wizard is hidden, is not written to any file, and is passed to Git only for that download. On the command line, set `RUNIVRA_ENTERPRISE_TOKEN` or `RUNIVRA_CUSTOM_TOKEN` instead of using the options, so the token stays out of your shell history.

## Roadmap

- Staging and production setups, with Nginx, a domain and a certificate
- Packaged releases, so that installing is one command (`apt install runivra`, and equivalents for macOS and Windows)
- Monitoring with Prometheus and Loki
- A small web control panel


## Develop

```
go test ./...
go vet ./...
gofmt -l .
```

The same checks run on every pull request.

## Licence

MIT. See [LICENSE](LICENSE).
