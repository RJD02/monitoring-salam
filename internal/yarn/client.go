package yarn

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"salam-monitoring/internal/logger"
)

// Application represents a Yarn application
type Application struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	ApplicationType   string  `json:"applicationType"`
	User              string  `json:"user"`
	Queue             string  `json:"queue"`
	State             string  `json:"state"`
	FinalStatus       string  `json:"finalStatus"`
	Progress          float64 `json:"progress"`
	TrackingUI        string  `json:"trackingUI"`
	TrackingURL       string  `json:"trackingUrl"`
	Diagnostics       string  `json:"diagnostics"`
	ClusterID         int64   `json:"clusterId"`
	ApplicationTags   string  `json:"applicationTags"`
	StartedTime       int64   `json:"startedTime"`
	FinishedTime      int64   `json:"finishedTime"`
	ElapsedTime       int64   `json:"elapsedTime"`
	AMContainerLogs   string  `json:"amContainerLogs"`
	AMHostHTTPAddress string  `json:"amHostHttpAddress"`
	AllocatedMB       int64   `json:"allocatedMB"`
	AllocatedVCores   int64   `json:"allocatedVCores"`
	RunningContainers int64   `json:"runningContainers"`
}

// AppsResponse represents the response from Yarn RM API
type AppsResponse struct {
	Apps struct {
		App []*Application `json:"app"`
	} `json:"apps"`
}

// ClusterInfo represents cluster information
type ClusterInfo struct {
	ID                     int64  `json:"id"`
	StartedOn              int64  `json:"startedOn"`
	State                  string `json:"state"`
	HAState                string `json:"haState"`
	ResourceManagerVersion string `json:"resourceManagerVersion"`
}

// ClusterMetrics represents cluster metrics
type ClusterMetrics struct {
	AppsSubmitted         int64 `json:"appsSubmitted"`
	AppsCompleted         int64 `json:"appsCompleted"`
	AppsPending           int64 `json:"appsPending"`
	AppsRunning           int64 `json:"appsRunning"`
	AppsFailed            int64 `json:"appsFailed"`
	AppsKilled            int64 `json:"appsKilled"`
	ReservedMB            int64 `json:"reservedMB"`
	AvailableMB           int64 `json:"availableMB"`
	AllocatedMB           int64 `json:"allocatedMB"`
	TotalMB               int64 `json:"totalMB"`
	ReservedVirtualCores  int64 `json:"reservedVirtualCores"`
	AvailableVirtualCores int64 `json:"availableVirtualCores"`
	AllocatedVirtualCores int64 `json:"allocatedVirtualCores"`
	TotalVirtualCores     int64 `json:"totalVirtualCores"`
	ContainersAllocated   int64 `json:"containersAllocated"`
	ContainersReserved    int64 `json:"containersReserved"`
	ContainersPending     int64 `json:"containersPending"`
	TotalNodes            int64 `json:"totalNodes"`
	ActiveNodes           int64 `json:"activeNodes"`
	LostNodes             int64 `json:"lostNodes"`
	UnhealthyNodes        int64 `json:"unhealthyNodes"`
	DecommissioningNodes  int64 `json:"decommissioningNodes"`
	DecommissionedNodes   int64 `json:"decommissionedNodes"`
	RebootedNodes         int64 `json:"rebootedNodes"`
}

// Client represents a Yarn Resource Manager client
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Yarn RM client
func NewClient(baseURL string) *Client {
	logger.Info("Creating Yarn client for RM: %s", baseURL)
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetRunningApplications retrieves all running applications
func (c *Client) GetRunningApplications() ([]*Application, error) {
	return c.GetApplicationsByState("RUNNING")
}

// GetApplicationsByState retrieves applications by their state
func (c *Client) GetApplicationsByState(state string) ([]*Application, error) {
	url := fmt.Sprintf("%s/ws/v1/cluster/apps?states=%s", c.baseURL, state)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch applications: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	var appsResponse AppsResponse
	if err := json.NewDecoder(resp.Body).Decode(&appsResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return appsResponse.Apps.App, nil
}

// GetApplication retrieves a specific application by ID
func (c *Client) GetApplication(appID string) (*Application, error) {
	url := fmt.Sprintf("%s/ws/v1/cluster/apps/%s", c.baseURL, appID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch application: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	var appResponse struct {
		App *Application `json:"app"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&appResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return appResponse.App, nil
}

// KillApplication kills a specific application
func (c *Client) KillApplication(appID string) error {
	url := fmt.Sprintf("%s/ws/v1/cluster/apps/%s/state", c.baseURL, appID)

	payload := `{"state":"KILLED"}`

	req, err := http.NewRequest("PUT", url, strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to kill application: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to kill application: HTTP %d", resp.StatusCode)
	}

	logger.Info("Successfully killed application: %s", appID)
	return nil
}

// KillApplicationsByPattern kills applications matching a pattern
func (c *Client) KillApplicationsByPattern(pattern string) ([]string, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	apps, err := c.GetRunningApplications()
	if err != nil {
		return nil, fmt.Errorf("failed to get running applications: %w", err)
	}

	var killedApps []string
	for _, app := range apps {
		if regex.MatchString(app.Name) {
			if err := c.KillApplication(app.ID); err != nil {
				logger.LogError(fmt.Sprintf("Failed to kill application %s (%s)", app.ID, app.Name), err)
				continue
			}
			killedApps = append(killedApps, app.ID)
		}
	}

	logger.Info("Killed %d applications matching pattern: %s", len(killedApps), pattern)
	return killedApps, nil
}

// GetStaleApplications returns applications that have been running longer than the specified duration
func (c *Client) GetStaleApplications(maxDuration time.Duration) ([]*Application, error) {
	apps, err := c.GetRunningApplications()
	if err != nil {
		return nil, err
	}

	var staleApps []*Application
	now := time.Now().Unix() * 1000 // Convert to milliseconds

	for _, app := range apps {
		if app.StartedTime > 0 {
			elapsed := time.Duration(now-app.StartedTime) * time.Millisecond
			if elapsed > maxDuration {
				staleApps = append(staleApps, app)
			}
		}
	}

	return staleApps, nil
}

// GetClusterInfo retrieves cluster information
func (c *Client) GetClusterInfo() (*ClusterInfo, error) {
	url := fmt.Sprintf("%s/ws/v1/cluster/info", c.baseURL)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cluster info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	var infoResponse struct {
		ClusterInfo *ClusterInfo `json:"clusterInfo"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&infoResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return infoResponse.ClusterInfo, nil
}

// GetClusterMetrics retrieves cluster metrics
func (c *Client) GetClusterMetrics() (*ClusterMetrics, error) {
	url := fmt.Sprintf("%s/ws/v1/cluster/metrics", c.baseURL)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cluster metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	var metricsResponse struct {
		ClusterMetrics *ClusterMetrics `json:"clusterMetrics"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&metricsResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return metricsResponse.ClusterMetrics, nil
}

// FormatDuration formats duration for display
func FormatDuration(milliseconds int64) string {
	if milliseconds == 0 {
		return "N/A"
	}

	duration := time.Duration(milliseconds) * time.Millisecond

	if duration < time.Minute {
		return fmt.Sprintf("%.0fs", duration.Seconds())
	} else if duration < time.Hour {
		return fmt.Sprintf("%.1fm", duration.Minutes())
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%.1fh", duration.Hours())
	} else {
		days := int(duration.Hours() / 24)
		hours := duration.Hours() - float64(days*24)
		return fmt.Sprintf("%dd %.1fh", days, hours)
	}
}

// FormatMemory formats memory for display
func FormatMemory(megabytes int64) string {
	if megabytes == 0 {
		return "0 MB"
	}

	if megabytes < 1024 {
		return fmt.Sprintf("%d MB", megabytes)
	} else if megabytes < 1024*1024 {
		return fmt.Sprintf("%.1f GB", float64(megabytes)/1024)
	} else {
		return fmt.Sprintf("%.1f TB", float64(megabytes)/(1024*1024))
	}
}

// IsHealthy checks if the cluster appears healthy
func (c *Client) IsHealthy() bool {
	info, err := c.GetClusterInfo()
	if err != nil {
		return false
	}
	return info.State == "STARTED"
}
