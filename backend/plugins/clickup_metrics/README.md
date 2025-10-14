# ClickUp Metrics Plugin

This is a **metric plugin** that calculates business cycle time metrics for ClickUp tasks. It runs automatically after DORA and refdiff plugins complete.

## Overview

The plugin tracks the complete journey of a ClickUp task from creation to production deployment:

- **Planning Time**: Task Created → First Commit
- **Code Time**: First Commit → PR Merged
- **Deploy Time**: PR Merged → Deployment
- **Total Cycle Time**: Task Created → Deployment

## Architecture

This plugin is separate from the `clickup` data collection plugin and implements the `MetricPluginBlueprintV200` interface, which means:

1. It runs automatically as part of project metrics collection
2. It executes **after** `dora` and `refdiff` plugins via `RunAfter()`
3. It uses tables created by the `clickup` plugin (`_tool_clickup_tasks`, `_tool_clickup_task_deployments`)
4. It filters by `project_name` (like DORA) rather than by connection

## Dependencies

- **clickup plugin**: Must be installed and have collected task data
- **dora plugin**: Must run first to generate `cicd_deployments` and `cicd_deployment_commits`
- **refdiff plugin**: Must run to generate `commits_diffs` table
- **GitHub plugin**: Must have collected PR and commit data

## How It Works

1. Finds ClickUp tasks linked to GitHub PRs (via `git_hub_pr_number` field or PR title/branch matching)
2. Locates the PR in the `pull_requests` table
3. Finds the first commit in `pull_request_commits`
4. Uses DORA's deployment strategy to find when the PR's merge commit was deployed (via `commits_diffs`)
5. Calculates time intervals and stores results in `_tool_clickup_task_deployments`

## Compilation

To compile this plugin:

```bash
DEVLAKE_PLUGINS=clickup,clickup_metrics make dev
```

Or add to your environment:
```bash
export DEVLAKE_PLUGINS="clickup,clickup_metrics"
make dev
```

## Configuration

The plugin is configured automatically when added to a project's metric plugins. It uses:

- `projectName`: The DevLake project name (automatically provided by the framework)

## Database Schema

Uses tables from the `clickup` plugin:

### _tool_clickup_task_deployments

Stores the relationship between tasks and deployments with calculated metrics:

| Field | Type | Description |
|-------|------|-------------|
| task_id | varchar(255) | ClickUp task ID |
| deployment_id | varchar(255) | CI/CD deployment ID |
| connection_id | uint64 | ClickUp connection ID |
| task_created_at | datetime | When task was created |
| first_commit_at | datetime | First commit in PR |
| pr_created_at | datetime | When PR was created |
| pr_merged_at | datetime | When PR was merged |
| deployment_at | datetime | When deployed to production |
| planning_time | int | Minutes from task to first commit |
| code_time | int | Minutes from first commit to PR merge |
| deploy_time | int | Minutes from PR merge to deployment |
| total_cycle_time | int | Minutes from task to deployment |

## Grafana Dashboards

SQL queries for Grafana dashboards are in `backend/plugins/clickup/grafana_dashboard_queries.sql`.

Example dashboards:
- `grafana/dashboards/ClickUpBusinessCycleTime.json` - Standalone business metrics
- `grafana/dashboards/DORAWithBusiness.json` - DORA + business cycle time combined
