# Salam Unified Monitoring Platform - Production Deployment Package

## Package Contents (3.3MB ZIP file)

```
salam-monitoring-deployment.zip
└── deploy-package/
    ├── bin/salam-monitor                    # Main application binary (7.7MB, statically linked)
    ├── config/config.yaml                  # Configuration template
    ├── static/                             # Self-contained web assets
    │   ├── css/tailwind.min.css            # Complete CSS framework
    │   └── js/htmx.min.js                  # Interactive web functionality
    ├── templates/                          # HTML templates (embedded in binary)
    ├── systemd/salam-monitor.service       # Linux service configuration
    ├── scripts/deploy.sh                   # Comprehensive deployment script
    ├── install.sh                          # Quick installation script
    └── README.md                           # Detailed documentation
```

## Installation on Target Machine

### Method 1: Quick Install (Recommended)
```bash
unzip salam-monitoring-deployment.zip
cd deploy-package/
sudo ./install.sh
```

### Method 2: Manual Installation
```bash
unzip salam-monitoring-deployment.zip
cd deploy-package/
sudo ./scripts/deploy.sh
```

## Key Features

✅ **Self-Contained**: No external dependencies or CDN requirements
✅ **Statically Linked**: Works on any Linux x86_64 system (GLIBC independent)
✅ **Complete Web Assets**: CSS and JavaScript bundled locally
✅ **MySQL Ready**: Informatica database integration with go-sql-driver/mysql
✅ **Comprehensive Logging**: Timestamped logs to `~/nfs_backup/monitoring/monitoring_util/<date>/info.log`
✅ **Production Ready**: Systemd service, monitoring user, proper permissions

## Post-Installation Configuration

1. **Edit Configuration** (Required):
   ```bash
   sudo nano /opt/salam-monitoring/config/config.yaml
   ```
   - Set `mode: prod`
   - Update `nfs_root_prod: "/your/nfs/path"`
   - Configure MySQL connection:
     ```yaml
     services:
       informatica_db:
         host: "your-mysql-server"
         service: "informatica_db"
         user: "informatica_user"
         password: "your_password"
       yarn_rm_url: "http://your-yarn-rm:8088"
     ```

2. **Start Service**:
   ```bash
   sudo systemctl start salam-monitor
   sudo systemctl status salam-monitor
   ```

3. **Access Web Interface**:
   - Open browser: `http://server-ip:8080`
   - Default port: 8080 (configurable in config.yaml)

## Troubleshooting

- **Service Logs**: `sudo journalctl -u salam-monitor -f`
- **Application Logs**: `/home/monitoring/nfs_backup/monitoring/monitoring_util/<date>/info.log`
- **Health Check**: `curl http://localhost:8080/api/health/status`
- **Configuration Test**: `/opt/salam-monitoring/bin/salam-monitor --version`

## Network Requirements

- **Outbound**: None (fully self-contained)
- **Inbound**: Port 8080 (configurable)
- **Internal**: MySQL connection to Informatica database, Yarn ResourceManager REST API

## System Requirements

- **OS**: Linux x86_64 (any distribution)
- **Memory**: 256MB+ RAM
- **Storage**: 50MB+ disk space
- **Privileges**: Root access for installation

## Security Notes

- Service runs as dedicated `monitoring` user (non-privileged)
- No external network dependencies
- Local file system access for NFS monitoring
- MySQL connection uses standard authentication

This deployment package resolves the styling issues by including all CSS and JavaScript files locally instead of relying on external CDNs.