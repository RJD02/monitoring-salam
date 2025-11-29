# Salam Monitoring Platform - .env Configuration

## Quick Start

1. **Extract the deployment package:**
   ```bash
   tar -xzf salam-monitoring-env-*.tar.gz
   cd deploy-env/
   ```

2. **Create your configuration:**
   ```bash
   # Copy example configuration
   cp .env.example .env
   
   # Edit configuration values
   nano .env
   ```

3. **Run with custom configuration:**
   ```bash
   # Use your .env file
   ./monitoring-server --config=.env
   
   # Or specify path to any .env file
   ./monitoring-server --config=/path/to/custom.env
   ```

## Configuration Options

### Basic Settings
- `ENV`: Application mode (`test` or `prod`)
- `HOST`: Server bind address (default: `0.0.0.0`)
- `PORT`: Server port (default: `8080`)

### NFS Configuration
- `NFS_ROOT`: Direct path specification (overrides mode-specific paths)
- `NFS_ROOT_TEST`: Test mode NFS path
- `NFS_ROOT_PROD`: Production mode NFS path

### Yarn Configuration
- `YARN_RM_URL`: Yarn Resource Manager URL
- `YARN_RM_URL_TEST`: Test mode Yarn URL (optional)

### SQL Server Configuration
- `INFORMATICA_DB_HOST`: SQL Server hostname
- `INFORMATICA_DB_PORT`: SQL Server port (default: `1433`)
- `INFORMATICA_DB_NAME`: Database name
- `INFORMATICA_DB_USER`: Database username
- `INFORMATICA_DB_PASS`: Database password
- `INFORMATICA_TIME_OFFSET`: Timezone offset in hours (default: `3`)

### Logging Configuration
- `LOG_LEVEL`: Log level (`debug`, `info`, `warn`, `error`)
- `LOG_FILE_PATH`: Log directory path
- `LOG_FILE_ENABLED`: Enable file logging (`true`/`false`)
- `LOG_JSON_ENABLED`: Enable JSON log format (`true`/`false`)

## Examples

### Development Setup
```bash
./monitoring-server --config=.env.example
```

### Production Setup
```bash
./monitoring-server --config=prod.env.example
```

### Custom Configuration
```bash
./monitoring-server --config=/opt/monitoring/production.env
```

## CLI Commands with .env

All CLI commands work with .env configuration:

```bash
# Show configuration
./monitoring-server --config=prod.env config

# List applications
./monitoring-server --config=prod.env yarn list

# Show logs
./monitoring-server --config=prod.env logs today
```

## Legacy YAML Support

YAML configuration files are still supported:
```bash
./monitoring-server --config=config.yaml
```

Environment variables always take precedence over configuration file values.