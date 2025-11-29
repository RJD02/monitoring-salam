# Salam Unified Monitoring Platform - Deployment Package

## Contents
- `bin/salam-monitor`: Main application binary (statically linked)
- `config/config.yaml`: Configuration file
- `static/`: Static assets (CSS, JS) for offline operation
- `systemd/salam-monitor.service`: Systemd service file
- `scripts/deploy.sh`: Deployment script

## Installation

1. Extract this package to your target directory (e.g., `/opt/salam-monitoring`)
2. Update `config/config.yaml` with your production settings
3. Run the deployment script: `sudo ./scripts/deploy.sh`
4. Start the service: `sudo systemctl start salam-monitor`

## Configuration

Edit `config/config.yaml`:
- Set `mode: prod` for production
- Update `nfs_root_prod` to your NFS path
- Configure Yarn ResourceManager URL
- Set Informatica MySQL database details

## Logs

Application logs are written to: `~/nfs_backup/monitoring/monitoring_util/<date>/info.log`

## Service Management

```bash
# Start service
sudo systemctl start salam-monitor

# Stop service  
sudo systemctl stop salam-monitor

# View logs
sudo journalctl -u salam-monitor -f

# Check status
sudo systemctl status salam-monitor
```

## Network Requirements

This package includes all required JavaScript and CSS files for offline operation.
No external CDN access is required.