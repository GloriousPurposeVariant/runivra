# Runivra roadmap

Tick a box by changing `[ ]` to `[x]`. "Proven" means someone watched it work
on a real machine, not only that the tests pass.

## 1. Development setup

### Done

- [x] Command line and wizard for setup
- [x] Animated checklist with progress bar
- [x] Clone Odoo, Enterprise (token or local copy), custom addons
- [x] Stack: everything in Docker
- [x] Stack: Odoo in Docker, PostgreSQL on this computer
- [x] Stack: everything on this computer, no Docker (command line)
- [x] Install Docker, PostgreSQL and Python when missing
- [x] Create a database user, or use an existing one, with the login checked in the wizard
- [x] Start Odoo and report when it answers
- [x] Supported versions 16.0 to 20.0, one shared list

### To do

- [x] Wizard: third stack option and the Python question
- [x] `info` and `db` for projects that run on the host
- [x] `init` for a hand-installed Odoo (no compose file)
- [ ] Database on another port, another server, or a managed service (AWS RDS, Azure)
- [ ] Live output window under the running step
- [ ] One-click fix when the Dockerfile misses a requirements file
- [ ] Linux: PostgreSQL access rules for Docker to reach the host database

### Proven on a real machine

| | Windows | Linux | macOS |
|---|---|---|---|
| Everything in Docker | [x] 19.0, 20.0 | [ ] | [ ] |
| Odoo in Docker, host database | [x] | [ ] | [ ] |
| Everything on the host | [x] 17.0 | [ ] | [ ] |
| Install Docker | [ ] | [ ] | [ ] |
| Install PostgreSQL | [x] | [ ] | [ ] |
| Install Python | [ ] | [ ] | [ ] |
| Odoo 16.0 and 18.0 | [ ] | [ ] | [ ] |

### Checked by CI on every pull request

| | Windows | Linux | macOS |
|---|---|---|---|
| Unit tests and build | [x] | [x] | [x] |

## 2. Working on a project

- [x] Project config in `.runivra/config.yml`, found from any subfolder
- [x] `init` for an existing Docker project
- [x] Addons folders read live from `odoo.conf`
- [x] `db`: list databases and remember one per set of custom folders
- [ ] `shell`, `bash`, `psql`, `logs`, `start`, `stop`
- [ ] Menu: plain `runivra` opens everything in one place

## 3. Upgrade

- [ ] Read module manifests from the tracked folders
- [ ] Read installed versions from the database
- [ ] Detect by version change and by file checksum
- [ ] Upgrade only modules that do not depend on another changed module
- [ ] Preview with the database shown and changeable
- [ ] Run, then record checksums
- [ ] `--db`, `--modules`, `--all`, `--yes` for GitOps

## 4. Backup and restore

- [ ] Backup: database dump, filestore and a manifest
- [ ] Restore, with checks before anything is overwritten
- [ ] Neutralize: optional, automatic for development and staging

## 5. Staging and production

Mostly the development work repeated with production settings.

- [ ] Server templates: `odoo.conf`, `Dockerfile`, `docker-compose.yml`
- [ ] Nginx in front, websocket routed to the gevent port
- [ ] Domain and certificate
- [ ] Generated secrets; database password in `.env` only
- [ ] Bundled database or an external one
- [ ] Restart on reboot, log rotation
- [ ] Kubernetes as a second way to run Odoo

## 6. Multi-server orchestration

- [ ] Controller service and node agents
- [ ] Several application servers behind a load balancer
- [ ] Rolling deployment, one node at a time
- [ ] Database topology: leader, replicas, read-only endpoints
- [ ] Shared filestore

## 7. Later

- [ ] Wizard redesign
- [ ] Monitoring: Prometheus and Loki
- [ ] Automated end-to-end test on 20.0
- [ ] Packaged releases, so no Go is needed to install
