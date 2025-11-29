#!/bin/bash

# Create deployment package for RHEL production systems
# This script creates a complete deployment package with all necessary files

set -e

echo "=== Creating Salam Monitoring Deployment Package ==="

# Configuration
PACKAGE_NAME="salam-monitoring-deploy"
BUILD_DIR="build"
PACKAGE_DIR="$BUILD_DIR/$PACKAGE_NAME"

# Clean and create build directory
rm -rf "$BUILD_DIR"
mkdir -p "$PACKAGE_DIR"

echo "Building application for RHEL..."
# Build statically linked binary for RHEL compatibility
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags="-s -w" -o "$PACKAGE_DIR/salam-monitor" ./cmd

echo "Copying deployment files..."
# Copy configuration files
mkdir -p "$PACKAGE_DIR/config"
cp "config/prod-config.yaml" "$PACKAGE_DIR/config/"

# Copy systemd service
mkdir -p "$PACKAGE_DIR/systemd"
cp "systemd/salam-monitor.service" "$PACKAGE_DIR/systemd/"

# Copy deployment script
mkdir -p "$PACKAGE_DIR/scripts"
cp "scripts/deploy-prod.sh" "$PACKAGE_DIR/scripts/"

# Copy documentation
cp "README.md" "$PACKAGE_DIR/" 2>/dev/null || echo "No README.md found, skipping..."

# Create environment file template
cat > "$PACKAGE_DIR/.env.production" << 'EOF'
# Salam Monitoring Platform - Production Environment Configuration
# Copy this file to .env and customize for your environment

# Environment Mode
ENV=prod

# Server Configuration
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# NFS Paths
NFS_ROOT_PROD=/home/informaticaadmin/nfs_backup/monitoring

# Yarn Resource Manager
YARN_RM_URL=http://rm-host:8088

# Informatica Database (Oracle)
INF_DB_HOST=172.16.1.100
INF_DB_PORT=1521
INF_DB_SERVICE=ORCL
INF_DB_USER=repo_read
INF_DB_PASSWORD=change_this_password
TIME_OFFSET_HOURS=3

# Logging
LOG_LEVEL=info
LOG_FILE=true
LOG_JSON=false
LOG_DIR=/opt/salam-monitoring/logs
EOF

# Create installation README
cat > "$PACKAGE_DIR/INSTALL.md" << 'EOF'
# Salam Unified Monitoring Platform - Installation Guide

## Production Installation on RHEL 7/8/9

### Prerequisites
- RHEL 7, 8, or 9 system
- Root access for installation
- Network access to Yarn Resource Manager
- Access to Informatica Oracle database
- NFS mount at `/home/informaticaadmin/nfs_backup/monitoring`

### Installation Steps

1. **Extract the package**:
   ```bash
   tar -xzf salam-monitoring-deploy.tar.gz
   cd salam-monitoring-deploy
   ```

2. **Review and customize configuration**:
   ```bash
   # Copy and edit the environment configuration
   cp .env.production .env
   nano .env
   
   # Edit the main configuration file
   nano config/prod-config.yaml
   ```

3. **Run the installation script**:
   ```bash
   sudo ./scripts/deploy-prod.sh
   ```

### Post-Installation

- **Web Interface**: http://your-server:8080
- **Service Management**:
  ```bash
  sudo systemctl status salam-monitor
  sudo systemctl restart salam-monitor
  sudo systemctl stop salam-monitor
  ```

- **View Logs**:
  ```bash
  sudo journalctl -u salam-monitor -f
  tail -f /opt/salam-monitoring/logs/*.log
  ```

- **CLI Commands**:
  ```bash
  /opt/salam-monitoring/salam-monitor logs today
  /opt/salam-monitoring/salam-monitor yarn list
  /opt/salam-monitoring/salam-monitor wf tree platform="miniboss"
  ```

### Configuration

Key configuration files:
- `/opt/salam-monitoring/config/config.yaml` - Main configuration
- `/opt/salam-monitoring/.env` - Environment overrides
- `/etc/systemd/system/salam-monitor.service` - Systemd service

### Troubleshooting

1. **Service won't start**:
   ```bash
   journalctl -u salam-monitor -n 50
   ```

2. **Check NFS access**:
   ```bash
   ls -la /home/informaticaadmin/nfs_backup/monitoring
   ```

3. **Test database connectivity**:
   ```bash
   /opt/salam-monitoring/salam-monitor --mode=prod
   ```

4. **Check port availability**:
   ```bash
   netstat -tlnp | grep :8080
   ```

For additional support, check the application logs and system journal.
EOF

# Create the compressed package
echo "Creating deployment package..."
cd "$BUILD_DIR"
tar -czf "${PACKAGE_NAME}.tar.gz" "$PACKAGE_NAME"
cd ..

# Show package information
echo
echo "=== Deployment Package Created ==="
echo "Package: $BUILD_DIR/${PACKAGE_NAME}.tar.gz"
echo "Size: $(du -h $BUILD_DIR/${PACKAGE_NAME}.tar.gz | cut -f1)"
echo
echo "Package contents:"
echo "$(tar -tzf $BUILD_DIR/${PACKAGE_NAME}.tar.gz | head -20)"
echo
echo "To deploy on production system:"
echo "1. Copy $BUILD_DIR/${PACKAGE_NAME}.tar.gz to target RHEL system"
echo "2. Extract: tar -xzf ${PACKAGE_NAME}.tar.gz"
echo "3. Run: sudo ./${PACKAGE_NAME}/scripts/deploy-prod.sh"
echo
echo "=== Package creation completed! ==="