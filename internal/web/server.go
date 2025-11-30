package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strconv"
	"time"

	"salam-monitoring/internal/config"
	"salam-monitoring/internal/informatica"
	"salam-monitoring/internal/logger"
	"salam-monitoring/internal/nfs"
	"salam-monitoring/internal/yarn"

	"github.com/gorilla/mux"
)

// Server represents the web server
type Server struct {
	config      *config.Config
	staticFiles embed.FS
	templates   *template.Template
	router      *mux.Router
	infClient   *informatica.Client
	yarnClient  *yarn.Client
	nfsScanner  *nfs.Scanner
}

// NewServer creates a new web server instance
func NewServer(cfg *config.Config, staticFiles embed.FS) *Server {
	logger.Info("Initializing web server...")

	server := &Server{
		config:      cfg,
		staticFiles: staticFiles,
		router:      mux.NewRouter(),
	}

	// Initialize Informatica client if in production mode
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
			logger.LogError("Failed to initialize Informatica client", err)
		} else {
			server.infClient = infClient
		}
	} else {
		// In test mode, create a mock client
		infConfig := informatica.DatabaseConfig{
			Host:       "localhost",
			Port:       1433,
			Database:   "INFORMATICA_TEST",
			Username:   "test",
			Password:   "test",
			TimeOffset: 3,
		}

		infClient, err := informatica.NewClient(infConfig)
		if err != nil {
			logger.LogError("Failed to initialize Informatica mock client", err)
		} else {
			server.infClient = infClient
		}
	}

	// Initialize NFS scanner
	nfsScanner := nfs.NewScanner(cfg.GetNFSRoot())
	server.nfsScanner = nfsScanner
	logger.Info("NFS scanner initialized for root: %s", cfg.GetNFSRoot())

	// Initialize Yarn client
	yarnClient := yarn.NewClient(cfg.Services.YarnRMURL)
	server.yarnClient = yarnClient
	logger.Info("Yarn client initialized for RM: %s", cfg.Services.YarnRMURL)

	server.setupRoutes()
	server.loadTemplates()

	logger.Info("Web server initialization completed")
	return server
}

// Start starts the web server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Server.Port)
	logger.Info("Starting HTTP server on %s", addr)
	fmt.Printf("Server starting on http://localhost%s\n", addr)
	return http.ListenAndServe(addr, s.router)
}

// loggingMiddleware logs all HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		logger.LogRequest(r.Method, r.URL.Path, r.RemoteAddr, 200, duration)
	})
}

// setupRoutes configures all the routes
func (s *Server) setupRoutes() {
	logger.Info("Setting up HTTP routes...")

	// Add logging middleware
	s.router.Use(s.loggingMiddleware)

	// Static files
	staticSubFS, err := fs.Sub(s.staticFiles, "static")
	if err != nil {
		logger.LogError("Failed to create static sub-filesystem", err)
		staticSubFS = s.staticFiles
	}
	s.router.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(http.FS(staticSubFS))),
	)

	// Main pages
	s.router.HandleFunc("/", s.handleHome).Methods("GET")
	s.router.HandleFunc("/nfs", s.handleNFS).Methods("GET")
	s.router.HandleFunc("/yarn", s.handleYarn).Methods("GET")
	s.router.HandleFunc("/informatica", s.handleInformatica).Methods("GET")
	s.router.HandleFunc("/dashboard", s.handleDashboard).Methods("GET")
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// HTMX endpoints
	s.router.HandleFunc("/api/nfs/logs", s.handleNFSLogs).Methods("GET")
	s.router.HandleFunc("/api/nfs/search", s.handleNFSSearch).Methods("POST")
	s.router.HandleFunc("/api/nfs/log-content", s.handleNFSLogContent).Methods("GET")
	s.router.HandleFunc("/api/yarn/apps", s.handleYarnApps).Methods("GET")
	s.router.HandleFunc("/api/yarn/cluster-metrics", s.handleYarnClusterMetrics).Methods("GET")
	s.router.HandleFunc("/api/yarn/kill", s.handleYarnKill).Methods("POST")
	s.router.HandleFunc("/api/informatica/workflows", s.handleInformaticaWorkflows).Methods("GET")
	s.router.HandleFunc("/api/dashboard/yarn-summary", s.handleDashboardYarnSummary).Methods("GET")
	s.router.HandleFunc("/api/health/status", s.handleHealthStatus).Methods("GET")

	// New Informatica endpoints as per specs
	s.router.HandleFunc("/informatica/workflows/today", s.handleInformaticaWorkflowsToday).Methods("GET")
	s.router.HandleFunc("/informatica/workflow/{statId:[0-9]+}", s.handleInformaticaWorkflowDetail).Methods("GET")

	logger.Info("HTTP routes configured successfully")
}

// loadTemplates loads all HTML templates
func (s *Server) loadTemplates() {
	logger.Info("Loading HTML templates...")
	var err error
	s.templates, err = template.ParseFS(s.staticFiles, "templates-deploy/*.html")
	if err != nil {
		logger.LogError("Failed to load templates", err)
	} else {
		logger.Info("Templates loaded successfully")
	}
}

// Template data structure
type TemplateData struct {
	Title   string
	Mode    string
	IsProd  bool
	NFSRoot string
	Data    interface{}
}

// Route handlers
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	logger.Info("Handling home page request")
	data := map[string]string{
		"message":    "Welcome to Salam Unified Monitoring Platform",
		"LastUpdate": time.Now().Format("2006-01-02 15:04:05"),
	}
	s.renderPageTemplate(w, "Dashboard", "index.html", data)
}

func (s *Server) handleNFS(w http.ResponseWriter, r *http.Request) {
	logger.Info("Handling NFS page request")
	s.renderPageTemplate(w, "NFS Monitoring", "nfs.html", nil)
}

func (s *Server) handleYarn(w http.ResponseWriter, r *http.Request) {
	logger.Info("Handling Yarn page request")
	s.renderPageTemplate(w, "Yarn Applications", "yarn.html", nil)
}

func (s *Server) handleInformatica(w http.ResponseWriter, r *http.Request) {
	logger.Info("Handling Informatica page request")
	s.renderPageTemplate(w, "Informatica Workflows", "informatica.html", nil)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	logger.Info("Handling dashboard page request")
	data := map[string]string{
		"message":    "Dashboard Overview",
		"LastUpdate": time.Now().Format("2006-01-02 15:04:05"),
	}
	s.renderPageTemplate(w, "Dashboard", "dashboard.html", data)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	logger.Info("Handling health page request")
	s.renderPageTemplate(w, "System Health", "health.html", nil)
}

// renderPageTemplate renders a full page template with layout
func (s *Server) renderPageTemplate(w http.ResponseWriter, title, contentTemplate string, data interface{}) {
	templateData := TemplateData{
		Title:   title,
		Mode:    s.config.Mode,
		IsProd:  s.config.IsProdMode(),
		NFSRoot: s.config.GetNFSRoot(),
		Data:    data,
	}

	if s.templates != nil {
		// First try to render the layout which includes the content template
		if err := s.templates.ExecuteTemplate(w, "layout.html", templateData); err != nil {
			logger.LogError(fmt.Sprintf("Failed to execute template layout for %s", contentTemplate), err)
			// Fallback: try to render just the content template directly
			if err2 := s.templates.ExecuteTemplate(w, contentTemplate, templateData); err2 != nil {
				logger.LogError(fmt.Sprintf("Fallback template execution also failed for %s", contentTemplate), err2)
				s.renderFallbackHTML(w, title, fmt.Sprintf("Template errors: %v, %v", err, err2))
			}
		}
	} else {
		logger.Error("Templates not loaded for: %s", contentTemplate)
		s.renderFallbackHTML(w, title, "Template system not loaded")
	}
}

// renderFallbackHTML renders a basic HTML fallback
func (s *Server) renderFallbackHTML(w http.ResponseWriter, title, message string) {
	w.Header().Set("Content-Type", "text/html")
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - Salam Monitoring</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100">
    <div class="min-h-screen flex items-center justify-center">
        <div class="bg-white p-8 rounded-lg shadow-lg">
            <h1 class="text-2xl font-bold text-gray-900 mb-4">%s</h1>
            <p class="text-gray-600 mb-4">%s</p>
            <div class="space-y-2">
                <p class="text-sm text-gray-500">Mode: %s</p>
                <p class="text-sm text-gray-500">NFS Root: %s</p>
            </div>
            <div class="mt-6">
                <a href="/" class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700">Go Home</a>
            </div>
        </div>
    </div>
</body>
</html>`, title, title, message, s.config.Mode, s.config.GetNFSRoot())

	w.Write([]byte(html))
}

// HTMX API handlers
func (s *Server) handleNFSLogs(w http.ResponseWriter, r *http.Request) {
	logger.Info("Handling NFS logs request")

	if s.nfsScanner == nil {
		logger.Error("NFS scanner not available")
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<div class="text-red-600">NFS scanner not available</div>`)
		return
	}

	// Get query parameters
	source := r.URL.Query().Get("source")
	status := r.URL.Query().Get("status")
	dateStr := r.URL.Query().Get("date")

	// Default to today's logs
	var workflowSummaries []*nfs.WorkflowSummary
	var err error

	if dateStr != "" {
		// Use specific date
		workflowSummaries, err = s.nfsScanner.ScanLogsForDate(dateStr)
	} else {
		// Use today's logs
		workflowSummaries, err = s.nfsScanner.ScanTodaysLogs()
	}

	if err != nil {
		logger.LogError("Failed to scan NFS logs", err)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<div class="text-red-600">Failed to scan NFS logs: %v</div>`, err)
		return
	}

	// Filter workflows by source and status
	filteredWorkflows := filterWorkflows(workflowSummaries, source, status)

	w.Header().Set("Content-Type", "text/html")
	if len(filteredWorkflows) == 0 {
		fmt.Fprintf(w, `<div class="text-gray-600 p-8 text-center">No logs found for the selected criteria</div>`)
		return
	}

	// Render workflows
	fmt.Fprintf(w, `<div class="space-y-6">`)
	for _, workflow := range filteredWorkflows {
		statusClass := getWorkflowStatusClass(workflow.Status)
		fmt.Fprintf(w, `
			<div class="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden hover:shadow-md transition-shadow">
				<div class="px-6 py-4 bg-gradient-to-r from-gray-50 to-white border-b border-gray-200">
					<div class="flex items-center justify-between">
						<div class="flex items-center space-x-4">
							<h3 class="text-lg font-semibold text-gray-900">%s</h3>
							<span class="px-3 py-1 text-xs font-medium rounded-full %s">%s</span>
						</div>
						<div class="flex items-center space-x-2 text-sm text-gray-500">
							<span>%s</span>
							<span>•</span>
							<span>%d files</span>
						</div>
					</div>
				</div>
				<div class="px-6 py-4">
					<div class="space-y-3">
		`, workflow.Workflow, statusClass, workflow.Status, workflow.Source, len(workflow.Logs))

		for _, log := range workflow.Logs {
			errorIcon := ""
			if log.HasErrors {
				errorIcon = `<svg class="w-5 h-5 text-red-500" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd"></path></svg>`
			}

			fmt.Fprintf(w, `
				<div class="flex items-center justify-between p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors cursor-pointer" 
					 onclick="showLogDetails('%s', '%s', '%s')">
					<div class="flex items-center space-x-3">
						%s
						<div>
							<div class="font-medium text-gray-900">%s</div>
							<div class="text-sm text-gray-500">%s • %.1f KB</div>
						</div>
					</div>
					<div class="text-xs text-gray-400">%s</div>
				</div>
			`, log.FilePath, log.LogType, log.Workflow, errorIcon, log.LogType, log.Date, float64(log.Size)/1024, log.ModTime.Format("15:04"))
		}

		fmt.Fprintf(w, `
					</div>
				</div>
			</div>
		`)
	}
	fmt.Fprintf(w, `</div>`)
}

// filterWorkflows filters workflows by source and status
func filterWorkflows(workflows []*nfs.WorkflowSummary, source, status string) []*nfs.WorkflowSummary {
	var filtered []*nfs.WorkflowSummary
	for _, workflow := range workflows {
		// Filter by source
		if source != "" && workflow.Source != source {
			continue
		}
		// Filter by status
		if status != "" && workflow.Status != status {
			continue
		}
		filtered = append(filtered, workflow)
	}
	return filtered
}

// getWorkflowStatusClass returns CSS classes for workflow status
func getWorkflowStatusClass(status string) string {
	switch status {
	case "completed":
		return "bg-green-100 text-green-800"
	case "failed":
		return "bg-red-100 text-red-800"
	case "running":
		return "bg-yellow-100 text-yellow-800"
	default:
		return "bg-gray-100 text-gray-800"
	}
}

func (s *Server) handleNFSSearch(w http.ResponseWriter, r *http.Request) {
	logger.Info("Handling NFS search request")

	searchQuery := r.FormValue("search")
	if searchQuery == "" {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<div class="text-gray-600">Enter search terms</div>`)
		return
	}

	// TODO: Implement search functionality
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div class="bg-yellow-100 p-4 rounded">Search for "%s" - Feature coming soon!</div>`, searchQuery)
}

func (s *Server) handleNFSLogContent(w http.ResponseWriter, r *http.Request) {
	logger.Info("Handling NFS log content request")

	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "File path required", http.StatusBadRequest)
		return
	}

	// TODO: Read and return actual log file content
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
		<div class="bg-gray-900 text-green-400 p-4 rounded font-mono text-sm overflow-x-auto">
			<div class="mb-2 text-gray-400">File: %s</div>
			<pre class="whitespace-pre-wrap">2024-11-21 10:30:00 INFO  Starting workflow execution\n2024-11-21 10:30:01 INFO  Connecting to database\n2024-11-21 10:30:02 INFO  Processing data batch 1/10\n2024-11-21 10:30:05 WARN  Slow query detected\n2024-11-21 10:30:10 INFO  Workflow completed successfully</pre>
		</div>
	`, filePath)
}

func (s *Server) handleDashboardYarnSummary(w http.ResponseWriter, r *http.Request) {
	logger.Info("Handling dashboard Yarn summary request")

	if s.yarnClient == nil {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<div class="text-gray-600">Yarn client not available</div>`)
		return
	}

	metrics, err := s.yarnClient.GetClusterMetrics()
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<div class="text-gray-600">Unable to connect to Yarn RM</div>`)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
		<div class="grid grid-cols-2 gap-4">
			<div class="bg-blue-50 p-4 rounded-lg">
				<div class="text-2xl font-bold text-blue-600">%d</div>
				<div class="text-sm text-gray-600">Running Apps</div>
			</div>
			<div class="bg-green-50 p-4 rounded-lg">
				<div class="text-2xl font-bold text-green-600">%.1f GB</div>
				<div class="text-sm text-gray-600">Available Memory</div>
			</div>
		</div>
	`, metrics.AppsRunning, float64(metrics.AvailableMB)/1024)
}

func (s *Server) handleYarnClusterMetrics(w http.ResponseWriter, r *http.Request) {
	logger.Info("Handling Yarn cluster metrics request")

	if s.yarnClient == nil {
		logger.Error("Yarn client not available")
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<div class="text-red-600">Yarn client not available</div>`)
		return
	}

	metrics, err := s.yarnClient.GetClusterMetrics()
	if err != nil {
		logger.LogError("Failed to get Yarn cluster metrics", err)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<div class="text-red-600">Failed to get cluster metrics: %v</div>`, err)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
		<div class="bg-blue-50 p-3 rounded text-center">
			<div class="text-2xl font-bold text-blue-600">%d</div>
			<div class="text-sm text-gray-600">Running Apps</div>
		</div>
		<div class="bg-yellow-50 p-3 rounded text-center">
			<div class="text-2xl font-bold text-yellow-600">%d</div>
			<div class="text-sm text-gray-600">Pending Apps</div>
		</div>
		<div class="bg-green-50 p-3 rounded text-center">
			<div class="text-2xl font-bold text-green-600">%.1f GB</div>
			<div class="text-sm text-gray-600">Available Memory</div>
		</div>
		<div class="bg-purple-50 p-3 rounded text-center">
			<div class="text-2xl font-bold text-purple-600">%d</div>
			<div class="text-sm text-gray-600">Active Nodes</div>
		</div>
	`, metrics.AppsRunning, metrics.AppsPending, float64(metrics.AvailableMB)/1024, metrics.ActiveNodes)
}

func (s *Server) handleYarnApps(w http.ResponseWriter, r *http.Request) {
	logger.Info("Handling Yarn applications request")

	if s.yarnClient == nil {
		logger.Error("Yarn client not available")
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<div class="text-red-600">Yarn client not available</div>`)
		return
	}

	// Get query parameters
	state := r.URL.Query().Get("state")
	if state == "" {
		state = "RUNNING"
	}

	apps, err := s.yarnClient.GetApplicationsByState(state)
	if err != nil {
		logger.LogError("Failed to get Yarn applications", err)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<div class="text-red-600">Failed to connect to Yarn RM: %v</div>`, err)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if len(apps) == 0 {
		fmt.Fprintf(w, `<div class="text-gray-600 p-4">No %s applications found</div>`, state)
		return
	}

	// Render applications table
	fmt.Fprintf(w, `<div class="overflow-x-auto">`)
	fmt.Fprintf(w, `<table class="min-w-full bg-white border border-gray-300">`)
	fmt.Fprintf(w, `<thead class="bg-gray-50">`)
	fmt.Fprintf(w, `<tr><th class="px-4 py-2 text-left">Application ID</th><th class="px-4 py-2 text-left">Name</th><th class="px-4 py-2 text-left">Type</th><th class="px-4 py-2 text-left">State</th><th class="px-4 py-2 text-left">Progress</th><th class="px-4 py-2 text-left">Actions</th></tr>`)
	fmt.Fprintf(w, `</thead><tbody>`)

	for _, app := range apps {
		fmt.Fprintf(w, `<tr class="border-t">`)
		fmt.Fprintf(w, `<td class="px-4 py-2 font-mono text-sm">%s</td>`, app.ID)
		fmt.Fprintf(w, `<td class="px-4 py-2">%s</td>`, app.Name)
		fmt.Fprintf(w, `<td class="px-4 py-2">%s</td>`, app.ApplicationType)
		fmt.Fprintf(w, `<td class="px-4 py-2"><span class="px-2 py-1 text-xs rounded %s">%s</span></td>`,
			getStateColor(app.State), app.State)
		fmt.Fprintf(w, `<td class="px-4 py-2">%.1f%%</td>`, app.Progress)
		fmt.Fprintf(w, `<td class="px-4 py-2">`)
		if app.State == "RUNNING" {
			fmt.Fprintf(w, `<button onclick="killApplication('%s')" class="bg-red-500 text-white px-2 py-1 rounded text-xs hover:bg-red-600">Kill</button>`, app.ID)
		}
		fmt.Fprintf(w, `</td>`)
		fmt.Fprintf(w, `</tr>`)
	}

	fmt.Fprintf(w, `</tbody></table></div>`)
}

// getStateColor returns CSS classes for different application states
func getStateColor(state string) string {
	switch state {
	case "RUNNING":
		return "bg-green-100 text-green-800"
	case "PENDING":
		return "bg-yellow-100 text-yellow-800"
	case "FINISHED":
		return "bg-blue-100 text-blue-800"
	case "FAILED":
		return "bg-red-100 text-red-800"
	case "KILLED":
		return "bg-gray-100 text-gray-800"
	default:
		return "bg-gray-100 text-gray-800"
	}
}

func (s *Server) handleYarnKill(w http.ResponseWriter, r *http.Request) {
	logger.Info("Handling Yarn kill request")

	if s.yarnClient == nil {
		logger.Error("Yarn client not available")
		http.Error(w, "Yarn client not available", http.StatusServiceUnavailable)
		return
	}

	appID := r.FormValue("appId")
	if appID == "" {
		http.Error(w, "Application ID required", http.StatusBadRequest)
		return
	}

	err := s.yarnClient.KillApplication(appID)
	if err != nil {
		logger.LogError("Failed to kill Yarn application", err)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<div class="text-red-600">Failed to kill application: %v</div>`, err)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div class="text-green-600">Application %s killed successfully</div>`, appID)
}

func (s *Server) handleInformaticaWorkflows(w http.ResponseWriter, r *http.Request) {
	logger.Info("Handling Informatica workflows request")

	if s.infClient == nil {
		logger.Error("Informatica client not available")
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<div class="text-red-600">Informatica client not available</div>`)
		return
	}

	view := r.URL.Query().Get("view")

	var workflows []informatica.WorkflowStat
	var err error

	if view == "running" {
		workflows, err = s.infClient.GetRunningWorkflows()
	} else {
		workflows, err = s.infClient.GetWorkflowsToday()
	}
	if err != nil {
		logger.LogError("Failed to get Informatica workflows", err)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<div class="text-red-600">Failed to get workflows: %v</div>`, err)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if len(workflows) == 0 {
		fmt.Fprintf(w, `<div class="text-gray-600 p-8 text-center">No workflows found for today</div>`)
		return
	}

	// Render workflows
	fmt.Fprintf(w, `<div class="space-y-4">`)
	for _, workflow := range workflows {
		statusClass := getInformaticaStatusClass(workflow.Status)
		fmt.Fprintf(w, `
			<div class="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden hover:shadow-lg transition-all duration-300">
				<div class="px-6 py-4 bg-gradient-to-r from-purple-50 to-indigo-50 border-b border-gray-200">
					<div class="flex items-center justify-between">
						<div class="flex items-center space-x-4">
							<div class="flex-shrink-0">
								<div class="w-10 h-10 bg-purple-100 rounded-full flex items-center justify-center">
									<svg class="w-6 h-6 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01"></path>
									</svg>
								</div>
							</div>
							<div>
								<h3 class="text-lg font-semibold text-gray-900">%s</h3>
								<p class="text-sm text-gray-600">%s</p>
							</div>
						</div>
						<div class="flex items-center space-x-3">
							<span class="px-3 py-1 text-xs font-medium rounded-full %s">%s</span>
							<button onclick="showWorkflowDetails(%d)" class="text-indigo-600 hover:text-indigo-900 text-sm font-medium">
								View Details
							</button>
						</div>
					</div>
				</div>
				<div class="px-6 py-4">
					<div class="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
						<div><span class="text-gray-500">Start Time:</span> <span class="font-medium">%s</span></div>
						<div><span class="text-gray-500">End Time:</span> <span class="font-medium">%s</span></div>
						<div><span class="text-gray-500">Duration:</span> <span class="font-medium">%s</span></div>
						<div><span class="text-gray-500">Folder:</span> <span class="font-medium">%s</span></div>
					</div>
				</div>
			</div>
		`, workflow.WorkflowName, "Folder", statusClass, workflow.Status, workflow.StatID,
			formatTime(workflow.StartedAt), formatTimePtr(workflow.FinishedAt),
			calculateDurationPtr(workflow.StartedAt, workflow.FinishedAt), "Default")
	}
	fmt.Fprintf(w, `</div>`)
}

// getInformaticaStatusClass returns CSS classes for Informatica workflow status
func getInformaticaStatusClass(status string) string {
	switch status {
	case "Succeeded":
		return "bg-green-100 text-green-800"
	case "Failed":
		return "bg-red-100 text-red-800"
	case "Running":
		return "bg-yellow-100 text-yellow-800"
	case "Suspended":
		return "bg-orange-100 text-orange-800"
	default:
		return "bg-gray-100 text-gray-800"
	}
}

// Helper functions for time formatting
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "N/A"
	}
	return t.Format("15:04:05")
}

func formatTimePtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return "N/A"
	}
	return t.Format("15:04:05")
}

func calculateDuration(start, end time.Time) string {
	if start.IsZero() || end.IsZero() {
		return "N/A"
	}
	duration := end.Sub(start)
	if duration < 0 {
		return "In Progress"
	}
	return duration.Truncate(time.Second).String()
}

func calculateDurationPtr(start time.Time, end *time.Time) string {
	if start.IsZero() || end == nil || end.IsZero() {
		return "In Progress"
	}
	duration := end.Sub(start)
	if duration < 0 {
		return "In Progress"
	}
	return duration.Truncate(time.Second).String()
}

func (s *Server) handleHealthStatus(w http.ResponseWriter, r *http.Request) {
	logger.Info("Handling health status request")

	// Check various system components
	health := map[string]string{
		"Server":      "OK",
		"Config":      "OK",
		"Templates":   "Unknown",
		"NFS":         "Unknown",
		"Yarn":        "Unknown",
		"Informatica": "Unknown",
	}

	if s.templates != nil {
		health["Templates"] = "OK"
	} else {
		health["Templates"] = "ERROR"
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
		<div class="grid grid-cols-2 gap-4">
			<div class="bg-green-100 p-4 rounded"><strong>Server:</strong> %s</div>
			<div class="bg-green-100 p-4 rounded"><strong>Config:</strong> %s</div>
			<div class="bg-%s-100 p-4 rounded"><strong>Templates:</strong> %s</div>
			<div class="bg-gray-100 p-4 rounded"><strong>NFS:</strong> %s</div>
			<div class="bg-gray-100 p-4 rounded"><strong>Yarn:</strong> %s</div>
			<div class="bg-gray-100 p-4 rounded"><strong>Informatica:</strong> %s</div>
		</div>
	`, health["Server"], health["Config"],
		map[string]string{"OK": "green", "ERROR": "red", "Unknown": "gray"}[health["Templates"]],
		health["Templates"], health["NFS"], health["Yarn"], health["Informatica"])
}

// handleInformaticaWorkflowsToday returns today's workflows from Informatica in JSON format
func (s *Server) handleInformaticaWorkflowsToday(w http.ResponseWriter, r *http.Request) {
	logger.Info("Handling Informatica workflows today request")

	if s.infClient == nil {
		http.Error(w, "Informatica client not available", http.StatusServiceUnavailable)
		return
	}

	view := r.URL.Query().Get("view")

	var workflows []informatica.WorkflowStat
	var err error

	if view == "running" {
		workflows, err = s.infClient.GetRunningWorkflows()
	} else {
		workflows, err = s.infClient.GetWorkflowsToday()
	}
	if err != nil {
		logger.LogError("Failed to get Informatica workflows", err)
		http.Error(w, "Failed to get workflows", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workflows)
}

// handleInformaticaWorkflowDetail returns a specific workflow with its tasks
func (s *Server) handleInformaticaWorkflowDetail(w http.ResponseWriter, r *http.Request) {
	logger.Info("Handling Informatica workflow detail request")

	if s.infClient == nil {
		http.Error(w, "Informatica client not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	statIDStr := vars["statId"]

	statID, err := strconv.ParseInt(statIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid stat ID", http.StatusBadRequest)
		return
	}

	workflowWithTasks, err := s.infClient.GetWorkflowWithTasks(statID)
	if err != nil {
		logger.LogError("Failed to get workflow with tasks", err)
		http.Error(w, "Failed to get workflow", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workflowWithTasks)
}
