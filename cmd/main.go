package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"salam-monitoring/internal/config"
	"salam-monitoring/internal/informatica"
	"salam-monitoring/internal/logger"
	"salam-monitoring/internal/nfs"
	"salam-monitoring/internal/web"
	"salam-monitoring/internal/yarn"
)

//go:embed static/* templates-deploy/*
var staticFiles embed.FS

var (
	configPath = flag.String("config", "", "Path to config file")
	mode       = flag.String("mode", "", "Override mode (test|prod)")
	showHelp   = flag.Bool("help", false, "Show help")
	version    = flag.Bool("version", false, "Show version")
)

const appVersion = "1.0.0"

func main() {
	flag.Parse()

	// Initialize logging first
	if err := logger.InitLogger(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.CloseLogger()

	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logger.Info("Received shutdown signal")
		logger.CloseLogger()
		os.Exit(0)
	}()

	logger.Info("Starting Salam Unified Monitoring Platform v%s", appVersion)

	if *showHelp {
		showUsage()
		return
	}

	if *version {
		fmt.Printf("Salam Unified Monitoring Platform v%s\n", appVersion)
		return
	}

	// Handle CLI commands
	args := flag.Args()
	if len(args) > 0 {
		handleCLI(args, *configPath)
		return
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.LogError("Failed to load configuration", err)
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override mode if specified via flag
	if *mode != "" {
		cfg.Mode = *mode
	}

	logger.Info("Configuration loaded - Mode: %s, NFS Root: %s, Port: %d", cfg.Mode, cfg.GetNFSRoot(), cfg.Server.Port)
	fmt.Printf("Starting Salam Monitoring Platform v%s in %s mode\n", appVersion, cfg.Mode)
	fmt.Printf("NFS Root: %s\n", cfg.GetNFSRoot())
	fmt.Printf("Server will start on port %d\n", cfg.Server.Port)

	// Start web server
	server := web.NewServer(cfg, staticFiles)
	if err := server.Start(); err != nil {
		logger.LogError("Server failed", err)
		log.Fatalf("Server failed: %v", err)
	}
}

// getConfigSource returns a description of where config is loaded from
func getConfigSource(configPath string) string {
	if configPath == "" {
		return "Default + Environment Variables"
	}
	if strings.HasSuffix(strings.ToLower(configPath), ".env") {
		return fmt.Sprintf(".env file: %s", configPath)
	}
	return fmt.Sprintf("YAML file: %s", configPath)
}

func handleCLI(args []string, configPath string) {
	command := args[0]

	switch command {
	case "config":
		// Load configuration for debug display
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Configuration Debug Info:\n")
		fmt.Printf("  Config Source: %s\n", getConfigSource(configPath))
		fmt.Printf("  Mode: %s\n", cfg.Mode)
		fmt.Printf("  Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
		fmt.Printf("  Yarn RM URL: %s\n", cfg.Services.YarnRMURL)
		fmt.Printf("  NFS Root: %s\n", cfg.GetNFSRoot())
		fmt.Printf("  Informatica DB: %s:%d/%s\n", cfg.Services.InformaticaDB.Host, cfg.Services.InformaticaDB.Port, cfg.Services.InformaticaDB.Database)
		fmt.Printf("  Log Level: %s\n", cfg.Logging.Level)
		os.Exit(0)
	case "logs":
		handleLogsCommand(args[1:], configPath)
	case "yarn":
		handleYarnCommand(args[1:], configPath)
	case "wf":
		handleWorkflowCommand(args[1:], configPath)
	default:
		fmt.Printf("Unknown command: %s\\n", command)
		showUsage()
		os.Exit(1)
	}
}

func handleLogsCommand(args []string, configPath string) {
	if len(args) == 0 {
		fmt.Println("Usage: salam-monitor logs <subcommand>")
		fmt.Println("Subcommands:")
		fmt.Println("  today    Show today's logs")
		return
	}

	switch args[0] {
	case "today":
		fmt.Println("Showing today's logs...")

		// Load config to get NFS path
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			return
		}

		// Initialize NFS scanner and scan today's workflows
		scanner := nfs.NewScanner(cfg.GetNFSRoot())
		workflows, err := scanner.ScanTodaysLogs()
		if err != nil {
			fmt.Printf("Error scanning workflows: %v\n", err)
			return
		}

		fmt.Printf("Found %d workflows today:\n\n", len(workflows))
		for _, wf := range workflows {
			fmt.Printf("Workflow: %s\n", wf.Workflow)
			fmt.Printf("  Source: %s\n", wf.Source)
			fmt.Printf("  Status: %s\n", wf.Status)
			fmt.Printf("  Log Entries: %d\n", len(wf.Logs))
			if wf.HasErrors {
				fmt.Printf("  ⚠️  HAS ERRORS\n")
			}
			fmt.Println()
		}
	default:
		fmt.Printf("Unknown logs subcommand: %s\n", args[0])
	}
}

func handleYarnCommand(args []string, configPath string) {
	if len(args) == 0 {
		fmt.Println("Usage: salam-monitor yarn <subcommand>")
		fmt.Println("Subcommands:")
		fmt.Println("  kill pattern=\"<pattern>\"    Kill jobs matching pattern")
		fmt.Println("  list                         List running applications")
		return
	}

	// Load config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	// Initialize Yarn client
	client := yarn.NewClient(cfg.GetYarnURL())

	switch args[0] {
	case "kill":
		if len(args) < 2 || !strings.HasPrefix(args[1], "pattern=") {
			fmt.Println("Usage: yarn kill pattern=\"<pattern>\"")
			return
		}
		pattern := strings.TrimPrefix(args[1], "pattern=")
		pattern = strings.Trim(pattern, "\"")

		fmt.Printf("Killing Yarn applications matching pattern: %s\n", pattern)
		killedApps, err := client.KillApplicationsByPattern(pattern)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Successfully killed %d applications\n", len(killedApps))
		for _, appID := range killedApps {
			fmt.Printf("  - %s\n", appID)
		}
	case "list":
		fmt.Println("Listing running Yarn applications...")
		apps, err := client.GetRunningApplications()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Found %d running applications:\n\n", len(apps))
		for _, app := range apps {
			fmt.Printf("App ID: %s\n", app.ID)
			fmt.Printf("  Name: %s\n", app.Name)
			fmt.Printf("  State: %s\n", app.State)
			fmt.Printf("  User: %s\n", app.User)
			fmt.Printf("  Queue: %s\n", app.Queue)
			fmt.Printf("  Progress: %.1f%%\n", app.Progress)
			fmt.Println()
		}
	default:
		fmt.Printf("Unknown yarn subcommand: %s\n", args[0])
	}
}

func handleWorkflowCommand(args []string, configPath string) {
	if len(args) == 0 {
		fmt.Println("Usage: salam-monitor wf <subcommand>")
		fmt.Println("Subcommands:")
		fmt.Println("  tree platform=\"<platform>\"    Show workflow tree")
		return
	}

	// Load config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	switch args[0] {
	case "tree":
		if len(args) < 2 || !strings.HasPrefix(args[1], "platform=") {
			fmt.Println("Usage: wf tree platform=\"<platform>\"")
			return
		}
		platform := strings.TrimPrefix(args[1], "platform=")
		platform = strings.Trim(platform, "\"")

		fmt.Printf("Showing workflow tree for platform: %s\n\n", platform)

		// Initialize Informatica client if available
		if cfg.IsProdMode() {
			infConfig := informatica.DatabaseConfig{
				Host:       cfg.Services.InformaticaDB.Host,
				Port:       cfg.Services.InformaticaDB.Port,
				Database:   cfg.Services.InformaticaDB.Database,
				Username:   cfg.Services.InformaticaDB.Username,
				Password:   cfg.Services.InformaticaDB.Password,
				TimeOffset: cfg.Services.InformaticaDB.TimeOffset,
			}

			infClient, err := informatica.NewClient(infConfig)
			if err != nil {
				fmt.Printf("Error connecting to Informatica: %v\n", err)
				return
			}
			defer infClient.Close()

			// Get today's workflows
			workflows, err := infClient.GetWorkflowsToday()
			if err != nil {
				fmt.Printf("Error getting workflows: %v\n", err)
				return
			}

			// Filter by platform if specified
			for _, wf := range workflows {
				if platform == "" || strings.Contains(strings.ToLower(wf.WorkflowName), strings.ToLower(platform)) {
					fmt.Printf("📁 %s\n", wf.WorkflowName)
					fmt.Printf("   Status: %s\n", wf.Status)
					fmt.Printf("   Started: %s\n", wf.StartedAt.Format("2006-01-02 15:04:05"))

					// Get tasks for this workflow
					wfWithTasks, err := infClient.GetWorkflowWithTasks(wf.StatID)
					if err == nil && len(wfWithTasks.Tasks) > 0 {
						fmt.Printf("   Tasks:\n")
						for _, task := range wfWithTasks.Tasks {
							fmt.Printf("   └─ %s (%s) - %s\n", task.TaskName, task.ServiceName, task.Status)
						}
					}
					fmt.Println()
				}
			}
		} else {
			fmt.Println("Informatica workflow tree only available in production mode")
			fmt.Println("Showing NFS-based workflow information instead...")

			// Fall back to NFS scanning
			scanner := nfs.NewScanner(cfg.GetNFSRoot())
			workflows, err := scanner.ScanTodaysLogs()
			if err != nil {
				fmt.Printf("Error scanning NFS: %v\n", err)
				return
			}

			for _, wf := range workflows {
				if platform == "" || strings.Contains(strings.ToLower(wf.Source), strings.ToLower(platform)) {
					fmt.Printf("📁 %s\n", wf.Workflow)
					fmt.Printf("   Source: %s\n", wf.Source)
					fmt.Printf("   Status: %s\n", wf.Status)
					fmt.Printf("   Log Entries: %d\n", len(wf.Logs))
					fmt.Println()
				}
			}
		}
	default:
		fmt.Printf("Unknown workflow subcommand: %s\n", args[0])
	}
}

func showUsage() {
	fmt.Printf("Salam Unified Monitoring Platform v%s\n\n", appVersion)
	fmt.Println("Usage:")
	fmt.Println("  salam-monitor [flags]                    Start web server")
	fmt.Println("  salam-monitor [flags] <command> [args]   Run CLI command")
	fmt.Println()
	fmt.Println("Flags:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  config                                   Show current configuration")
	fmt.Println("  logs today                               Show today's logs")
	fmt.Println("  yarn kill pattern=\"spark_ingest\"         Kill jobs matching pattern")
	fmt.Println("  yarn list                                List running applications")
	fmt.Println("  wf tree platform=\"miniboss\"             Show workflow tree for platform")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Println("  Use .env file (recommended):             salam-monitor --config=path/to/.env")
	fmt.Println("  Use YAML file (legacy):                  salam-monitor --config=config.yaml")
	fmt.Println("  Environment variables override all settings")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  salam-monitor --config=/opt/monitoring/.env")
	fmt.Println("  salam-monitor --config=./prod.env --mode=prod")
	fmt.Println("  salam-monitor config")
	fmt.Println("  salam-monitor logs today")
}
