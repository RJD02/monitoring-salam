Salam Unified Monitoring Platform — Production Environment Setup & Build Guide

This document describes the production environment setup, build instructions, deployment layout, and feature/tech stack overview for the Salam Unified Monitoring Platform (Web UI + CLI).

⸻

#️⃣ 1. Overview

The Salam Unified Monitoring Platform is a single-binary Go application providing:
	•	Real‑time Spark/Yarn application monitoring
	•	NFS workflow log monitoring
	•	Informatica workflow hierarchy visualization
	•	Job kill operations (Spark/Yarn)
	•	Alerts and failure detection
	•	Platform-wide observability

It runs entirely offline, requires no external packages, and is built using Go + HTMX.

⸻

#️⃣ 2. Production Environment Requirements

2.1 Target Machine (Production Node)
	•	RHEL 7/8/9 or equivalent
	•	No internet access
	•	Local filesystem access to:
	•	/home/informaticaadmin/nfs_backup/monitoring/ (log root)
	•	Yarn Resource Manager REST API
	•	Informatica Repository DB (Oracle)
	•	Free ports (default: 8080)
	•	Systemd service manager

2.2 No External Dependencies
	•	❌ No Go compiler needed
	•	❌ No Node/npm
	•	❌ No database server
	•	✔ Everything embedded inside a single binary
	•	✔ Optionally uses SQLite for local history retention

⸻

#️⃣ 3. Project Build & Packaging

3.1 Build the Production Binary

Run on a build machine (any machine with Go installed):

go mod tidy
go build -o salam-monitor

This produces a self-contained binary with:
	•	Backend
	•	HTML templates
	•	HTMX JS
	•	Static assets (CSS, icons)
	•	Config templates

All assets are bundled using:

import "embed"
//go:embed static/* templates/*

3.2 Cross-Compile (if needed)

For RHEL 8 target:

GOOS=linux GOARCH=amd64 go build -o salam-monitor

3.3 Package Structure (Build Output)

release/
 └── salam-monitor        # single binary

Optionally compress:

tar -czvf salam-monitor.tar.gz salam-monitor


⸻

#️⃣ 4. Production Deployment Structure

4.1 Folder Layout on Production Machine

/opt/salam-monitoring/
 ├── salam-monitor               # binary
 ├── config/
 │    └── config.yaml            # runtime config
 ├── data/
 │    └── history.db             # sqlite (optional)
 ├── logs/
 │    ├── app.log
 │    ├── errors.log
 │    └── access.log
 └── systemd/
      └── salam-monitor.service

4.2 Configuration File (config.yaml)

server:
  port: 8080

paths:
  nfs_root: "/home/informaticaadmin/nfs_backup/monitoring"

services:
  yarn_rm_url: "http://rm-host:8088"
  informatica_db:
    host: 172.x.x.x
    service: ORCL
    user: repo_read
    password: XXXXX


⸻

#️⃣ 5. Systemd Service Setup

5.1 Create Unit File

/etc/systemd/system/salam-monitor.service

[Unit]
Description=Salam Unified Monitoring Platform
After=network.target

[Service]
ExecStart=/opt/salam-monitoring/salam-monitor --config /opt/salam-monitoring/config/config.yaml
WorkingDirectory=/opt/salam-monitoring/
Restart=always
User=informaticaadmin
Group=informaticaadmin

[Install]
WantedBy=multi-user.target

5.2 Start Service

sudo systemctl daemon-reload
sudo systemctl enable salam-monitor
sudo systemctl start salam-monitor

5.3 Check Status

sudo systemctl status salam-monitor


⸻

#️⃣ 6. Production File Access Design

6.1 NFS Backup Logs

The app auto‑scans:

/home/informaticaadmin/nfs_backup/monitoring/<source>/<YYYY-MM-DD>/<workflow>/

Files scanned:
	•	info.log
	•	error.log
	•	run.log

6.2 Yarn API Access

Uses:

<RM_HOST>:8088/ws/v1/cluster/apps?states=RUNNING

Supports:
	•	kill application by ID
	•	list running applications
	•	list stale jobs

6.3 Informatica Repository DB

Reads:
	•	workflows
	•	sessions
	•	parent-child structure
	•	workflow run status

⸻

#️⃣ 7. Platform Features

7.1 NFS Monitoring Tab
	•	Auto-detect today’s logs
	•	Error scanning
	•	Log preview & download
	•	Highlight failed workflows
	•	Search logs by keyword

7.2 Yarn Monitoring Tab
	•	List running Spark apps
	•	Show queues, progress, elapsed time
	•	Bulk kill jobs
	•	Kill jobs by regex

7.3 Informatica Workflow Tab
	•	Visual hierarchical tree
	•	Parent → Child → Task → Session mapping
	•	Current run state
	•	Last run status
	•	Session logs

7.4 Failure Dashboard
	•	Workflows failed today
	•	Jobs stuck > X minutes
	•	Spark failures grouped by cause
	•	Queue pressure alerts

7.5 System Health Tab
	•	CPU / RAM usage (via /proc)
	•	Disk usage
	•	Port checks
	•	Yarn node health

7.6 Spark Error Diagnostics
	•	Broadcast timeout detection
	•	OOM detection
	•	Driver bind failure (4040+) detection
	•	Oracle/Singlestore connection errors

7.7 CLI Tools
	•	salam-monitor logs today
	•	salam-monitor yarn kill pattern="spark_ingest"
	•	salam-monitor wf tree platform=miniboss

⸻

#️⃣ 8. Tech Stack

Backend
	•	Go 1.22+
	•	net/http
	•	HTML templates
	•	embed for bundling
	•	SQLite (optional)
	•	Goroutines for concurrency

Frontend
	•	HTMX (no build setup)
	•	Hyperscript (optional)
	•	Precompiled TailwindCSS
	•	Vanilla HTML templates

Why Go + HTMX
	•	No internet dependencies
	•	One single binary
	•	No JS build tools
	•	Extremely fast on RHEL
	•	Perfect for CLI + Web UI hybrid

⸻

#️⃣ 9. Deployment Lifecycle
	1.	Build binary on any machine
	2.	Copy binary → /opt/salam-monitoring/
	3.	Copy config
	4.	Enable systemd service
	5.	App runs at boot
	6.	Updates simply overwrite binary
	7.	Old logs archived automatically

⸻

#️⃣ 10. Test Environment Setup

For local or test development environments, the platform simulates the production NFS monitoring structure.

10.1 Local Project Folder Layout

During development, the repository will include its own mock NFS directory:

project-root/
 ├── cmd/
 ├── internal/
 ├── web/
 ├── nfs_backup/              # simulated test logs
 │    └── monitoring/
 │         └── <source>/<date>/<workflow>/info.log
 └── config/
      └── config.yaml

10.2 Test Mode Configuration

The config.yaml will include a switch:

mode: "test"  # or "prod"

paths:
  nfs_root_test: "./nfs_backup/monitoring"
  nfs_root_prod: "/home/informaticaadmin/nfs_backup/monitoring"

When mode = test, the application uses the local structure.
When mode = prod, it uses the real cluster filesystem.

10.3 Benefits of Test Mode
	•	Safe simulation of workflow logs
	•	Ability to replay failures or test log parsing
	•	CI/CD-compatible (no real cluster access)
	•	Reproducible sample Yarn responses (via mocked JSON files)
	•	Developers can run the UI in local mode with instant refresh

10.4 Swap Between Test & Prod

Command-line override:

./salam-monitor --mode=test
./salam-monitor --mode=prod

Environment variable override:

export SALAM_MODE=test


⸻

#️⃣ 11. Security Considerations**
	•	Only internal network access
	•	Binary permission: chmod 750
	•	Config contains credentials → restrict access
	•	Add token-based auth for UI (optional)

⸻
=