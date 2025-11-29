Informatica Monitoring Module — Technical Specification for Coding Agent

This document defines the requirements, data sources, queries, structures, and functional behavior needed to build the Informatica Monitoring component inside the Salam Unified Monitoring Platform.

It is derived from the sample SQL provided from the Informatica repository (PO_WORKFLOWSTAT, PO_TASKSTAT) and translated into a clear, implementation-ready specification.

⸻

#️⃣ 1. Purpose of This Module

The Informatica Monitoring module will:
	•	Fetch running, failed, and completed workflows from the Informatica Repository DB.
	•	Display workflows in the UI under a dedicated “Informatica” tab.
	•	Show parent workflows and their child tasks in a hierarchical tree.
	•	Display workflow states, start/end time, and elapsed duration.
	•	Auto-update at a fixed interval (e.g., every 10 seconds via HTMX partial refresh).
	•	Allow filtering by workflow name.
	•	Provide REST endpoints for programmatic access.

⸻

#️⃣ 2. Data Source Tables

The Informatica repository exposes two main statistics tables:

2.1 PO_WORKFLOWSTAT

Contains workflow-level metadata:
	•	POW_STATID — workflow run ID (primary key)
	•	POW_WORKFLOWDEFINITIONNAM — workflow name
	•	POW_CREATEDTIME, POW_STARTTIME, POW_ENDTIME, POW_LASTUPDATETIME
	•	POW_STATE — workflow state (0=RUNNING,1=SUCCESS,3=FAILED)

2.2 PO_TASKSTAT

Contains task-level data belonging to a workflow:
	•	POT_PARENTSTATID — foreign key mapping to POW_STATID
	•	POT_TASKNAME
	•	POT_SERVICENAME
	•	POT_NODE_NAME
	•	POT_STARTTIME, POT_ENDTIME, etc.

These two tables combined allow building a hierarchical workflow tree.

⸻

#️⃣ 3. Understanding Informatica Time Fields

In Informatica, all time fields are stored as Unix epoch in milliseconds.

Example:

DATEADD(SECOND, pow_starttime / 1000, '1970-01-01')

The platform must convert these fields into human-readable timestamps.

Time Fields to Convert:
	•	pow_createdtime
	•	pow_starttime
	•	pow_endtime
	•	pow_lastupdatetime
	•	Same for task table (pot_…)

Offset

The SQL query adds +3 hours offset:

DATEADD(HOUR, 3, ...)

This must be configurable using an environment variable:

TIME_OFFSET_HOURS=3


⸻

#️⃣ 4. Workflow-Level Query Logic

The workflow-level query (from PO_WORKFLOWSTAT) must return:
	•	Workflow name
	•	Run status
	•	Start time
	•	End time (null if running)
	•	Duration (hrs, min, sec)
	•	POW_STATID

Workflow Status Mapping

pow_state = 0  → RUNNING
pow_state = 1  → SUCCESS
pow_state = 3  → FAILED
other         → raw numeric

Running Duration Logic

If pow_endtime = 0 or NULL, duration = now - start.
Otherwise, duration = end - start.

⸻

#️⃣ 5. Task-Level Query Logic

The task-level CTE retrieves tasks belonging to a workflow.

The key join:

wf.POW_STATID = ts.POT_PARENTSTATID

Fields needed:
	•	Task name
	•	Service name (Session, Command, etc.)
	•	Node name
	•	Status
	•	Start & Finish timestamps
	•	Duration split into hrs/min/sec

Task Status Mapping

pot_state = 1 → RUNNING
pot_state = 2 → SUCCESS
else          → raw value


⸻

#️⃣ 6. Required Endpoints for the Monitoring Project

The coding agent must implement the following API endpoints in Go.

6.1 GET /informatica/workflows/today

Returns all workflows that started today.

Response example:

[
  {
    "stat_id": 1234,
    "workflow_name": "BRM_LOAD_JOB",
    "status": "RUNNING",
    "started_at": "2025-01-22T09:30:00",
    "finished_at": null,
    "elapsed": {
      "hrs": 1,
      "min": 22,
      "sec": 10
    }
  }
]

6.2 GET /informatica/workflow/{stat_id}

Returns workflow + all its child tasks.

Example response:

{
  "workflow": {...},
  "tasks": [
     { "task_name": "S_BILLING", "status": "RUNNING", ... },
     { "task_name": "S_AGGR", "status": "SUCCESS", ... }
  ]
}


⸻

#️⃣ 7. UI Requirements (Go + HTMX)

7.1 Informatica Tab Layout

The UI must contain:
	•	A filter box for workflow name
	•	Table listing: workflow, status, elapsed, start time
	•	Status badges (green=success, red=failed, yellow=running)
	•	A clickable row that expands into task tree

7.2 Task Tree Rendering

Each workflow row expands via HTMX:

<tr hx-get="/informatica/workflow/1234" hx-target="#wf-1234-tasks"></tr>

Child nodes displayed as:

- Session_1 (running)
  - Command_2 (success)
  - Command_3 (running)

7.3 Auto-Refresh

Use HTMX polling:

<div hx-get="/informatica/workflows/today" hx-trigger="every 10s" hx-target="#wf-table"></div>


⸻

#️⃣ 8. Error & Edge Case Handling
	•	Handle workflows with no tasks.
	•	Handle Informatica storing end time as 0.
	•	Handle workflows that started yesterday and still running.
	•	Handle missing task children.
	•	Handle DB unreachable state → display fallback UI.

⸻

#️⃣ 9. Database Access Specification

9.1 Library

Use Go Oracle driver:

goracle "github.com/godror/godror"

(Will be embedded or manually provided since no internet available.)

9.2 Connection Pool Settings

max_open_conns: 5
max_idle_conns: 1
conn_lifetime: 2m


⸻

#️⃣ 10. Required Helper Functions

The coding agent must implement:

✓ convertEpochMillisToTime(epoch_ms, offsetHours)

Converts Informatica fields.

✓ calculateElapsed(start_time, end_time)

Returns hrs/min/sec.

✓ mapWorkflowState(pow_state int)

Returns SUCCESS / FAILED / RUNNING.

✓ mapTaskState(pot_state int)

Returns SUCCESS / RUNNING.

⸻

#️⃣ 11. Final Expected Deliverables

The coding agent must deliver:

Backend
	•	Query builders for workflow & task stats
	•	JSON API endpoints
	•	Error-safe DB wrapper
	•	Struct models for workflow & task
	•	Time conversion helpers

Frontend
	•	HTMX templates for workflows
	•	HTMX expansion for tasks
	•	Status UI badges
	•	Auto-refresh table

Documentation
	•	Code comments
	•	Data type references
	•	Error messages

⸻

#️⃣ 12. Notes for Implementation
	•	DB credentials provided via .env
	•	Time offset (3 hours) also via .env
	•	Fail-safe retry logic required
	•	Avoid full table scans → apply date filters

⸻

This document provides everything needed for the coding agent to build all Informatica-related features for the monitoring project. Let me know if you want diagrams, database schemas, or a step-by-step coding plan added.