#!/bin/bash

# Salam Unified Monitoring Platform - Production Deployment Script
# This script sets up the monitoring platform on a production RHEL system

set -e

# Configuration
INSTALL_DIR="/opt/salam-monitoring"
SERVICE_USER="informaticaadmin"
SERVICE_GROUP="informaticaadmin"
BINARY_NAME="salam-monitor"

echo "=== Salam Unified Monitoring Platform - Production Deployment ==="
echo "Install directory: $INSTALL_DIR"
echo "Service user: $SERVICE_USER"
echo

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run as root"
    exit 1
fi

# Create installation directory
echo "Creating installation directory..."
mkdir -p "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR/config"
mkdir -p "$INSTALL_DIR/data"
mkdir -p "$INSTALL_DIR/logs"

# Copy binary and configuration
echo "Installing application files..."
cp "./$BINARY_NAME" "$INSTALL_DIR/"
cp "config/prod-config.yaml" "$INSTALL_DIR/config/config.yaml"

# Set permissions
echo "Setting file permissions..."
chown -R "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR"
chmod 755 "$INSTALL_DIR/$BINARY_NAME"
chmod 644 "$INSTALL_DIR/config/config.yaml"
chmod 755 "$INSTALL_DIR/logs"
chmod 755 "$INSTALL_DIR/data"

# Install systemd service
echo "Installing systemd service..."
cp "systemd/salam-monitor.service" "/etc/systemd/system/"
systemctl daemon-reload

# Create log rotation configuration
echo "Setting up log rotation..."
cat > "/etc/logrotate.d/salam-monitor" << EOF
$INSTALL_DIR/logs/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
    su $SERVICE_USER $SERVICE_GROUP
}
EOF

# Create NFS monitoring directory if it doesn't exist
echo "Ensuring NFS monitoring directory exists..."
NFS_DIR="/home/informaticaadmin/nfs_backup/monitoring"
if [ ! -d "$NFS_DIR" ]; then
    mkdir -p "$NFS_DIR"
    chown "$SERVICE_USER:$SERVICE_GROUP" "$NFS_DIR"
    echo "Created NFS monitoring directory: $NFS_DIR"
fi

# Enable and start service
echo "Enabling and starting service..."
systemctl enable salam-monitor.service
systemctl start salam-monitor.service

# Wait a moment and check status
sleep 3
if systemctl is-active --quiet salam-monitor.service; then
    echo "✅ Service started successfully!"
    echo
    echo "Service status:"
    systemctl status salam-monitor.service --no-pager -l
    echo
    echo "Web interface available at: http://$(hostname -I | awk '{print $1}'):8080"
    echo
    echo "Useful commands:"
    echo "  sudo systemctl status salam-monitor     # Check service status"
    echo "  sudo systemctl restart salam-monitor    # Restart service"
    echo "  sudo journalctl -u salam-monitor -f     # View live logs"
    echo "  $INSTALL_DIR/$BINARY_NAME logs today    # CLI: Show today's logs"
    echo "  $INSTALL_DIR/$BINARY_NAME --help        # CLI: Show all commands"
else
    echo "❌ Service failed to start!"
    echo "Check logs with: journalctl -u salam-monitor -n 50"
    exit 1
fi

echo
echo "=== Deployment completed successfully! ==="