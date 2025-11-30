package informatica

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"salam-monitoring/internal/logger"

	_ "github.com/denisenkom/go-mssqldb" // SQL Server driver
)

// WorkflowStat represents a workflow from PO_WORKFLOWSTAT
type WorkflowStat struct {
	StatID       int64       `json:"stat_id"`
	WorkflowName string      `json:"workflow_name"`
	Status       string      `json:"status"`
	StartedAt    time.Time   `json:"started_at"`
	FinishedAt   *time.Time  `json:"finished_at"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	Elapsed      ElapsedTime `json:"elapsed"`
}

// TaskStat represents a task from PO_TASKSTAT
type TaskStat struct {
	ParentStatID int64       `json:"parent_stat_id"`
	TaskName     string      `json:"task_name"`
	ServiceName  string      `json:"service_name"`
	NodeName     string      `json:"node_name"`
	Status       string      `json:"status"`
	StartedAt    time.Time   `json:"started_at"`
	FinishedAt   *time.Time  `json:"finished_at"`
	Elapsed      ElapsedTime `json:"elapsed"`
}

// ElapsedTime represents duration broken down into hours, minutes, seconds
type ElapsedTime struct {
	Hrs int `json:"hrs"`
	Min int `json:"min"`
	Sec int `json:"sec"`
}

// WorkflowWithTasks represents a workflow with its child tasks
type WorkflowWithTasks struct {
	Workflow WorkflowStat `json:"workflow"`
	Tasks    []TaskStat   `json:"tasks"`
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host       string
	Port       int
	Database   string
	Username   string
	Password   string
	TimeOffset int // hours offset for timezone conversion
}

// Client represents an Informatica SQL Server database client
type Client struct {
	config     DatabaseConfig
	db         *sql.DB
	timeOffset int
	mockMode   bool // For development when SQL Server is not available
}

// NewClient creates a new Informatica SQL Server client
func NewClient(config DatabaseConfig) (*Client, error) {
	logger.Info("Creating Informatica SQL Server client")

	client := &Client{
		config:     config,
		timeOffset: config.TimeOffset,
		mockMode:   false, // Try real connection first
	}

	// Construct SQL Server connection string
	dsn := fmt.Sprintf("server=%s;port=%d;database=%s;user id=%s;password=%s;encrypt=disable",
		config.Host, config.Port, config.Database, config.Username, config.Password)

	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		logger.LogError("Failed to connect to SQL Server, falling back to mock mode", err)
		client.mockMode = true
		return client, nil
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		logger.LogError("Failed to ping SQL Server, falling back to mock mode", err)
		db.Close()
		client.mockMode = true
		return client, nil
	}

	client.db = db
	logger.Info("Successfully connected to Informatica SQL Server database")
	return client, nil
}

// Close closes the database connection
func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// convertEpochMillisToTime converts Informatica epoch milliseconds to time with offset
func (c *Client) convertEpochMillisToTime(epochMs int64) time.Time {
	if epochMs == 0 {
		return time.Time{}
	}

	// Convert milliseconds to seconds and apply offset
	epochSeconds := epochMs / 1000
	timeOffset := time.Duration(c.timeOffset) * time.Hour

	return time.Unix(epochSeconds, 0).UTC().Add(timeOffset)
}

// calculateElapsed calculates elapsed time between start and end
func (c *Client) calculateElapsed(startTime, endTime time.Time) ElapsedTime {
	var duration time.Duration

	if endTime.IsZero() {
		// Still running - calculate from now
		duration = time.Since(startTime)
	} else {
		duration = endTime.Sub(startTime)
	}

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	return ElapsedTime{
		Hrs: hours,
		Min: minutes,
		Sec: seconds,
	}
}

// mapWorkflowState maps POW_STATE to readable status
func mapWorkflowState(powState int) string {
	switch powState {
	case 0:
		return "RUNNING"
	case 1:
		return "SUCCESS"
	case 3:
		return "FAILED"
	default:
		return fmt.Sprintf("UNKNOWN_%d", powState)
	}
}

// mapTaskState maps POT_STATE to readable status
func mapTaskState(potState int) string {
	switch potState {
	case 1:
		return "RUNNING"
	case 2:
		return "SUCCESS"
	default:
		return fmt.Sprintf("UNKNOWN_%d", potState)
	}
}

// GetWorkflowsToday retrieves all workflows that started today
func (c *Client) GetWorkflowsToday() ([]WorkflowStat, error) {
	if c.mockMode {
		return c.getMockWorkflowsToday(), nil
	}

	// SQL Server query for workflows that started today
	query := `
SELECT
POW_STATID,
POW_WORKFLOWDEFINITIONNAM,
POW_STATE,
POW_STARTTIME,
POW_ENDTIME,
POW_CREATEDTIME,
POW_LASTUPDATETIME
FROM PO_WORKFLOWSTAT
WHERE POW_STARTTIME >= DATEDIFF(SECOND, '1970-01-01', CAST(GETDATE() AS DATE)) * 1000
ORDER BY POW_STARTTIME DESC
`

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	workflows, err := c.queryWorkflows(ctx, query)
	if err != nil {
		return nil, err
	}

	logger.Info("Retrieved %d workflows for today", len(workflows))
	return workflows, nil
}

// GetWorkflowWithTasks retrieves a specific workflow and its tasks
func (c *Client) GetWorkflowWithTasks(statID int64) (*WorkflowWithTasks, error) {
	if c.mockMode {
		return c.getMockWorkflowWithTasks(statID), nil
	}

	logger.Info("Getting workflow with tasks for stat_id: %d", statID)

	// Get the workflow first
	workflowQuery := `
		SELECT 
			POW_STATID,
			POW_WORKFLOWDEFINITIONNAM,
			POW_STATE,
			POW_STARTTIME,
			POW_ENDTIME,
			POW_CREATEDTIME,
			POW_LASTUPDATETIME
		FROM PO_WORKFLOWSTAT
		WHERE POW_STATID = ?
	`

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wf WorkflowStat
	var powState int
	var startTimeMs, createdTimeMs, updatedTimeMs int64
	var endTimePtr *int64

	err := c.db.QueryRowContext(ctx, workflowQuery, statID).Scan(
		&wf.StatID,
		&wf.WorkflowName,
		&powState,
		&startTimeMs,
		&endTimePtr,
		&createdTimeMs,
		&updatedTimeMs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	// Convert workflow data
	wf.Status = mapWorkflowState(powState)
	wf.StartedAt = c.convertEpochMillisToTime(startTimeMs)
	wf.CreatedAt = c.convertEpochMillisToTime(createdTimeMs)
	wf.UpdatedAt = c.convertEpochMillisToTime(updatedTimeMs)

	if endTimePtr != nil {
		endTime := c.convertEpochMillisToTime(*endTimePtr)
		wf.FinishedAt = &endTime
		wf.Elapsed = c.calculateElapsed(wf.StartedAt, endTime)
	} else {
		wf.Elapsed = c.calculateElapsed(wf.StartedAt, time.Time{})
	}

	// Get tasks for this workflow
	tasksQuery := `
		SELECT 
			POT_PARENTSTATID,
			POT_TASKNAME,
			POT_SERVICENAME,
			POT_NODENAME,
			POT_STATE,
			POT_STARTTIME,
			POT_ENDTIME
		FROM PO_TASKSTAT
		WHERE POT_PARENTSTATID = ?
		ORDER BY POT_STARTTIME
	`

	rows, err := c.db.QueryContext(ctx, tasksQuery, statID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}
	defer rows.Close()

	var tasks []TaskStat
	for rows.Next() {
		var task TaskStat
		var potState int
		var taskStartMs int64
		var taskEndPtr *int64

		err := rows.Scan(
			&task.ParentStatID,
			&task.TaskName,
			&task.ServiceName,
			&task.NodeName,
			&potState,
			&taskStartMs,
			&taskEndPtr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task row: %w", err)
		}

		// Convert task data
		task.Status = mapTaskState(potState)
		task.StartedAt = c.convertEpochMillisToTime(taskStartMs)

		if taskEndPtr != nil {
			taskEndTime := c.convertEpochMillisToTime(*taskEndPtr)
			task.FinishedAt = &taskEndTime
			task.Elapsed = c.calculateElapsed(task.StartedAt, taskEndTime)
		} else {
			task.Elapsed = c.calculateElapsed(task.StartedAt, time.Time{})
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task rows: %w", err)
	}

	logger.Info("Retrieved workflow %s with %d tasks", wf.WorkflowName, len(tasks))
	return &WorkflowWithTasks{
		Workflow: wf,
		Tasks:    tasks,
	}, nil
}

// IsHealthy checks if the Informatica database connection is healthy
func (c *Client) IsHealthy() bool {
	if c.mockMode {
		return true
	}

	if c.db == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.db.PingContext(ctx) == nil
}

// Mock data for development/testing
func (c *Client) getMockWorkflowsToday() []WorkflowStat {
	now := time.Now()
	startTime1 := now.Add(-2 * time.Hour)
	startTime2 := now.Add(-1 * time.Hour)
	endTime1 := now.Add(-30 * time.Minute)

	workflows := []WorkflowStat{
		{
			StatID:       1001,
			WorkflowName: "BRM_LOAD_JOB",
			Status:       "RUNNING",
			StartedAt:    startTime1,
			FinishedAt:   nil,
			CreatedAt:    startTime1,
			UpdatedAt:    now,
			Elapsed:      c.calculateElapsed(startTime1, time.Time{}),
		},
		{
			StatID:       1002,
			WorkflowName: "BILLING_ETL_WORKFLOW",
			Status:       "SUCCESS",
			StartedAt:    startTime2,
			FinishedAt:   &endTime1,
			CreatedAt:    startTime2,
			UpdatedAt:    endTime1,
			Elapsed:      c.calculateElapsed(startTime2, endTime1),
		},
		{
			StatID:       1003,
			WorkflowName: "CUSTOMER_DATA_SYNC",
			Status:       "FAILED",
			StartedAt:    startTime1,
			FinishedAt:   &endTime1,
			CreatedAt:    startTime1,
			UpdatedAt:    endTime1,
			Elapsed:      c.calculateElapsed(startTime1, endTime1),
		},
	}

	return workflows
}

func (c *Client) getMockWorkflowWithTasks(statID int64) *WorkflowWithTasks {
	workflows := c.getMockWorkflowsToday()

	// Find the workflow
	var workflow WorkflowStat
	found := false
	for _, wf := range workflows {
		if wf.StatID == statID {
			workflow = wf
			found = true
			break
		}
	}

	if !found {
		return &WorkflowWithTasks{}
	}

	// Generate mock tasks for the workflow
	taskStart1 := workflow.StartedAt.Add(5 * time.Minute)
	taskStart2 := workflow.StartedAt.Add(10 * time.Minute)
	taskEnd1 := workflow.StartedAt.Add(15 * time.Minute)

	tasks := []TaskStat{
		{
			ParentStatID: statID,
			TaskName:     "S_BILLING_EXTRACT",
			ServiceName:  "Session",
			NodeName:     "ETL_NODE_01",
			Status:       "SUCCESS",
			StartedAt:    taskStart1,
			FinishedAt:   &taskEnd1,
			Elapsed:      c.calculateElapsed(taskStart1, taskEnd1),
		},
		{
			ParentStatID: statID,
			TaskName:     "S_BILLING_TRANSFORM",
			ServiceName:  "Session",
			NodeName:     "ETL_NODE_01",
			Status:       "RUNNING",
			StartedAt:    taskStart2,
			FinishedAt:   nil,
			Elapsed:      c.calculateElapsed(taskStart2, time.Time{}),
		},
	}

	return &WorkflowWithTasks{
		Workflow: workflow,
		Tasks:    tasks,
	}
}

// GetRunningWorkflows returns only running top-level workflows (excludes child workflows when possible)
func (c *Client) GetRunningWorkflows() ([]WorkflowStat, error) {
	if c.mockMode {
		return c.getMockRunningWorkflows(), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	runningQueryWithParent := `
SELECT
POW_STATID,
POW_WORKFLOWDEFINITIONNAM,
POW_STATE,
POW_STARTTIME,
POW_ENDTIME,
POW_CREATEDTIME,
POW_LASTUPDATETIME
FROM PO_WORKFLOWSTAT
WHERE POW_STATE = 0
AND (POW_PARENTSTATID IS NULL OR POW_PARENTSTATID = 0)
ORDER BY POW_STARTTIME DESC
`

	runningQueryWithoutParent := `
SELECT
POW_STATID,
POW_WORKFLOWDEFINITIONNAM,
POW_STATE,
POW_STARTTIME,
POW_ENDTIME,
POW_CREATEDTIME,
POW_LASTUPDATETIME
FROM PO_WORKFLOWSTAT
WHERE POW_STATE = 0
ORDER BY POW_STARTTIME DESC
`

	workflows, err := c.queryWorkflows(ctx, runningQueryWithParent)
	if err != nil {
		if strings.Contains(strings.ToUpper(err.Error()), "POW_PARENTSTATID") {
			logger.Info("POW_PARENTSTATID column unavailable, retrying running workflows without child filter")
			return c.queryWorkflows(ctx, runningQueryWithoutParent)
		}
		return nil, err
	}

	return workflows, nil
}

// queryWorkflows executes a workflow-level query and converts the results
func (c *Client) queryWorkflows(ctx context.Context, query string, args ...any) ([]WorkflowStat, error) {
	logger.Info("Executing workflow query: %s", query)

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute workflow query: %w", err)
	}
	defer rows.Close()

	var workflows []WorkflowStat
	for rows.Next() {
		var wf WorkflowStat
		var powState int
		var startTimeMs, createdTimeMs, updatedTimeMs int64
		var endTimePtr *int64

		err := rows.Scan(
			&wf.StatID,
			&wf.WorkflowName,
			&powState,
			&startTimeMs,
			&endTimePtr,
			&createdTimeMs,
			&updatedTimeMs,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow row: %w", err)
		}

		wf.Status = mapWorkflowState(powState)
		wf.StartedAt = c.convertEpochMillisToTime(startTimeMs)
		wf.CreatedAt = c.convertEpochMillisToTime(createdTimeMs)
		wf.UpdatedAt = c.convertEpochMillisToTime(updatedTimeMs)

		if endTimePtr != nil {
			endTime := c.convertEpochMillisToTime(*endTimePtr)
			wf.FinishedAt = &endTime
			wf.Elapsed = c.calculateElapsed(wf.StartedAt, endTime)
		} else {
			wf.Elapsed = c.calculateElapsed(wf.StartedAt, time.Time{})
		}

		workflows = append(workflows, wf)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating workflow rows: %w", err)
	}

	return workflows, nil
}

func (c *Client) getMockRunningWorkflows() []WorkflowStat {
	all := c.getMockWorkflowsToday()
	var running []WorkflowStat
	for _, wf := range all {
		if wf.Status == "RUNNING" {
			running = append(running, wf)
		}
	}
	return running
}
