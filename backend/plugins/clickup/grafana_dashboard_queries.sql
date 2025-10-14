-- ============================================================================
-- ClickUp Business Cycle Time Dashboard Queries
-- ============================================================================
-- These queries track the full business cycle from ClickUp task creation
-- to production deployment, separate from standard DORA metrics
--
-- USAGE: Replace $connection_id with your ClickUp connection ID
-- ============================================================================

-- ----------------------------------------------------------------------------
-- STAT PANEL 1: Average Total Business Cycle Time (in hours)
-- ----------------------------------------------------------------------------
-- Shows the average total cycle time from task creation to deployment
-- Similar to "Average PR Cycle Time" in DORA dashboard
-- ----------------------------------------------------------------------------
WITH _tasks as(
  SELECT
    td.task_id,
    t.name as task_name,
    td.task_created_at,
    COALESCE(td.total_cycle_time/60.0, 0) as total_cycle_time_hours
  FROM _tool_clickup_task_deployments td
    JOIN _tool_clickup_tasks t ON t.id = td.task_id AND t.connection_id = td.connection_id
  WHERE
    $__timeFilter(td.deployment_at)
    AND td.connection_id IN (${connection_id})
  GROUP BY 1,2,3,4
)
SELECT
  AVG(total_cycle_time_hours) as 'Business Cycle Time(h)'
FROM _tasks;

-- ----------------------------------------------------------------------------
-- STAT PANEL 1.1: Average Planning Time (in hours)
-- ----------------------------------------------------------------------------
-- Shows average time from task creation to first commit
-- Similar to "Average PR Coding Time" in DORA dashboard
-- ----------------------------------------------------------------------------
WITH _tasks as(
  SELECT
    td.task_id,
    COALESCE(td.planning_time/60.0, 0) as planning_time_hours
  FROM _tool_clickup_task_deployments td
  WHERE
    $__timeFilter(td.deployment_at)
    AND td.connection_id IN (${connection_id})
  GROUP BY 1,2
)
SELECT
  AVG(planning_time_hours) as 'Planning Time(h)'
FROM _tasks;

-- ----------------------------------------------------------------------------
-- STAT PANEL 1.2: Average Code Time (in hours)
-- ----------------------------------------------------------------------------
-- Shows average time from first commit to PR merged
-- Similar to "Average PR Pickup Time" in DORA dashboard
-- ----------------------------------------------------------------------------
WITH _tasks as(
  SELECT
    td.task_id,
    COALESCE(td.code_time/60.0, 0) as code_time_hours
  FROM _tool_clickup_task_deployments td
  WHERE
    $__timeFilter(td.deployment_at)
    AND td.connection_id IN (${connection_id})
  GROUP BY 1,2
)
SELECT
  AVG(code_time_hours) as 'Code Time(h)'
FROM _tasks;

-- ----------------------------------------------------------------------------
-- STAT PANEL 1.3: Average Deploy Time (in hours)
-- ----------------------------------------------------------------------------
-- Shows average time from PR merged to deployment
-- Similar to "Average PR Deploy Time" in DORA dashboard
-- ----------------------------------------------------------------------------
WITH _tasks as(
  SELECT
    td.task_id,
    COALESCE(td.deploy_time/60.0, 0) as deploy_time_hours
  FROM _tool_clickup_task_deployments td
  WHERE
    $__timeFilter(td.deployment_at)
    AND td.connection_id IN (${connection_id})
  GROUP BY 1,2
)
SELECT
  AVG(deploy_time_hours) as 'Deploy Time(h)'
FROM _tasks;

-- ----------------------------------------------------------------------------
-- TABLE PANEL 2: Task Details
-- ----------------------------------------------------------------------------
-- List of individual tasks that completed deployment in selected time range
-- Similar to "PR Details" table in DORA dashboard
-- Useful for drilling down into specific tasks
-- ----------------------------------------------------------------------------
SELECT
  t.id as 'Task ID',
  t.name as 'Task Name',
  t.status as 'Task Status',
  t.priority as 'Priority',
  t.git_hub_pr_url as 'PR URL',
  t.git_hub_pr_url as metric_hidden,
  td.task_created_at as 'Task Created',
  td.first_commit_at as 'First Commit',
  td.pr_merged_at as 'PR Merged',
  td.deployment_at as 'Deployed',
  td.planning_time / 60.0 as 'planning_time',
  td.code_time / 60.0 as 'code_time',
  td.deploy_time / 60.0 as 'deploy_time',
  td.total_cycle_time / 60.0 as business_cycle_time
FROM _tool_clickup_tasks t
JOIN _tool_clickup_task_deployments td ON t.id = td.task_id AND t.connection_id = td.connection_id
WHERE
  $__timeFilter(td.deployment_at)
  AND t.connection_id IN (${connection_id})
ORDER BY td.deployment_at DESC
LIMIT 100;

-- ============================================================================
-- Variables to add to Grafana Dashboard:
-- ============================================================================
-- ${connection_id} - ClickUp connection ID (type: query, multi-select)
-- $__timeFilter() - Built-in Grafana time range filter
-- ============================================================================

-- ============================================================================
-- Notes:
-- ============================================================================
-- 1. Business Cycle Time = Planning Time + Code Time + Deploy Time
--    - Planning Time: Task Created → First Commit
--    - Code Time: First Commit → PR Merged
--    - Deploy Time: PR Merged → Deployment
--
-- 2. Times are stored in minutes in the database, converted to hours for display
--
-- 3. COALESCE is used to handle NULL values for partial cycles
--
-- 4. The metric_hidden column in the table is used for clickable PR URLs
-- ============================================================================
