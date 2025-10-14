# ClickUp Plugin for Apache DevLake

## Overview

The ClickUp plugin enables tracking of **full business cycle time** from task creation in ClickUp to deployment in production. This provides a complete view of delivery time that complements standard DORA metrics.

### What This Plugin Does

- **Collects tasks** from ClickUp via API
- **Links tasks to GitHub PRs** by extracting PR URLs from task descriptions
- **Tracks deployment completion** by connecting to existing DevLake deployment data
- **Calculates business metrics**:
  - **Planning Time**: Task created → First commit
  - **Code Time**: First commit → PR merged
  - **Deploy Time**: PR merged → Deployment
  - **Total Business Cycle Time**: Task created → Deployment

### How It's Different from DORA Metrics

| Metric | DORA Lead Time | Business Cycle Time |
|--------|----------------|---------------------|
| **Start** | First commit | ClickUp task created |
| **End** | Deployment | Deployment |
| **Measures** | Technical delivery speed | Full business delivery speed |
| **Use Case** | Engineering efficiency | Business planning & forecasting |

This plugin **does not affect** existing DORA metrics. It creates separate tables and metrics specifically for business cycle tracking.

## Architecture

### Data Flow

```
ClickUp API → Collector → Extractor → Enricher → Business Cycle Calculator
                                          ↓
                                    GitHub PRs
                                          ↓
                                    Deployments
```

### Database Tables

1. **`_tool_clickup_connections`** - Connection credentials
2. **`_tool_clickup_tasks`** - Raw task data from ClickUp
3. **`_tool_clickup_task_deployments`** - Calculated business cycle metrics

## Setup

### 1. Generate ClickUp API Token

1. Go to ClickUp Settings → Apps
2. Click "Generate" under API Token
3. Copy the token (starts with `pk_`)

### 2. Create Connection in DevLake

Using the DevLake API:

```bash
curl -X POST http://localhost:8080/plugins/clickup/connections \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ClickUp Production",
    "endpoint": "https://api.clickup.com/api/v2",
    "token": "pk_your_api_token_here",
    "rateLimitPerHour": 100
  }'
```

Response will include `connection_id` - save this for configuration.

### 3. Configure Data Collection

#### Option A: Via Blueprint (Recommended)

Add to your blueprint configuration:

```json
{
  "plugin": "clickup",
  "options": {
    "connectionId": 1,
    "listId": "your-clickup-list-id",
    "spaceId": "your-space-id",
    "projectName": "your-devlake-project"
  }
}
```

#### Option B: Direct Pipeline

```bash
curl -X POST http://localhost:8080/pipelines \
  -H "Content-Type: application/json" \
  -d '{
    "name": "clickup-business-metrics",
    "tasks": [[{
      "plugin": "clickup",
      "options": {
        "connectionId": 1,
        "listId": "123456789",
        "projectName": "my-project"
      }
    }]]
  }'
```

### 4. How to Get Your ClickUp List ID

**Method 1: From URL**
- Open your list in ClickUp
- Look at the URL: `https://app.clickup.com/123456/v/li/987654321`
- The number after `/li/` is your list ID: `987654321`

**Method 2: Using API**
```bash
curl -H "Authorization: pk_your_token" \
  https://api.clickup.com/api/v2/space/YOUR_SPACE_ID/list
```

## Task Requirements

### GitHub PR Linking

The plugin supports **TWO methods** to link ClickUp tasks with GitHub Pull Requests:

#### **Method 1: Task ID in PR Title or Branch Name** ⭐ Recommended

Include the ClickUp task ID in your PR title or branch name:

**Examples:**
```bash
# Branch name
git checkout -b feature/86b6tmtc3-enable-payouts

# Or in PR title
"86b6tmtc3: Enable payouts in PE"

# Or both
git checkout -b 86b6tmtc3-user-auth
# PR title: "86b6tmtc3: Implement user authentication"
```

**How it works:**
- The plugin searches the DevLake `pull_requests` table
- Finds PRs where `title LIKE '%86b6tmtc3%'` OR `head_ref LIKE '%86b6tmtc3%'`
- Automatically links the task to the PR

**Pros:**
- ✅ No manual work after creating PR
- ✅ Follows same pattern as Jira integration
- ✅ Fast (no extra API calls)
- ✅ Works for teams with naming conventions

---

#### **Method 2: PR URL in Task Comments**

Add a comment to the ClickUp task with the GitHub PR URL:

**Example:**
1. Go to the ClickUp task
2. Add a comment: `https://github.com/myorg/myrepo/pull/456`
3. The plugin will extract the PR number automatically

**How it works:**
- The plugin collects all comments from the task
- Uses regex to find GitHub PR URLs: `https://github\.com/[^/]+/[^/]+/pull/(\d+)`
- Updates the task with the PR information

**Pros:**
- ✅ Works even without naming conventions
- ✅ Can link PRs after the fact
- ✅ Supports multiple PRs per task

**Cons:**
- ❌ Requires manual comment
- ❌ Slower (requires API call to get comments)

---

### Linking Priority

The plugin tries both methods in this order:

1. **First:** Search by Task ID in PR title/branch (Method 1)
2. **Then:** Use PR number from comments if found (Method 2)
3. **Finally:** Fall back to regex in task description (legacy method)

This ensures maximum compatibility and flexibility.

---

### Best Practices

1. **Use Method 1** (Task ID in PR name) as your default approach
2. **Use Method 2** (comments) for exceptional cases or manual linking
3. **Consistent naming:** Use format `{task-id}-{description}` for branches
4. **Update task status** as work progresses through stages
5. **Close tasks in ClickUp** when deployed to production

## Configuration Options

| Option | Required | Description | Example |
|--------|----------|-------------|---------|
| `connectionId` | ✓ | ClickUp connection ID | `1` |
| `listId` | ✓ | ClickUp list to collect from | `"123456789"` |
| `spaceId` | | ClickUp space ID | `"sp_12345"` |
| `projectName` | | DevLake project for correlation | `"my-project"` |
| `since` | | Only collect tasks updated after this date | `"2025-01-01T00:00:00Z"` |

## Subtasks Execution Order

1. **collectTasks** - Fetch tasks from ClickUp API
2. **extractTasks** - Parse task data and extract PR links from description
3. **collectComments** - Fetch comments for each task
4. **extractComments** - Extract PR URLs from comments (Method 2)
5. **enrichTasksWithGitHub** - Link tasks to GitHub branches
6. **calculateBusinessCycleTime** - Calculate all timing metrics (uses Method 1 & 2)

## Grafana Dashboard

SQL queries for visualizing business cycle metrics are in `grafana_dashboard_queries.sql`.

### Key Panels to Add

1. **Median Business Cycle Time** - Overall time from task to deployment
2. **Phase Breakdown** - Planning vs Code vs Deploy time
3. **DORA Comparison** - Side-by-side with standard Lead Time
4. **Throughput** - Tasks deployed per week
5. **Individual Tasks** - Detailed list with cycle times

### Dashboard Variables

Add these variables to your Grafana dashboard:

- `$connection_id` - Your ClickUp connection ID
- `$project_name` - Your DevLake project name

## Troubleshooting

### No Data Showing in Dashboard

**Check 1: Are tasks being collected?**
```sql
SELECT COUNT(*) FROM _tool_clickup_tasks WHERE connection_id = 1;
```

**Check 2: Do tasks have PR links?**
```sql
SELECT COUNT(*) FROM _tool_clickup_tasks
WHERE connection_id = 1 AND github_pr_url != '';
```

**Check 3: Are cycle times calculated?**
```sql
SELECT COUNT(*) FROM _tool_clickup_task_deployments WHERE connection_id = 1;
```

### Tasks Not Linking to Deployments

**Verify PR exists in DevLake:**
```sql
SELECT id, number, head_ref FROM pull_requests WHERE number = 456;
```

**Verify deployment exists:**
```sql
SELECT id, finished_date FROM cicd_deployments
WHERE finished_date >= '2025-01-01';
```

### ClickUp API Rate Limiting

If you see 429 errors, reduce `rateLimitPerHour` in connection settings:

```bash
curl -X PATCH http://localhost:8080/plugins/clickup/connections/1 \
  -H "Content-Type: application/json" \
  -d '{"rateLimitPerHour": 50}'
```

## Data Model

### ClickUpTask Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | ClickUp task ID |
| `name` | string | Task title |
| `description` | text | Task description (contains PR links) |
| `status` | string | Current task status |
| `priority` | string | Task priority level |
| `created_at` | timestamp | When task was created |
| `github_pr_url` | string | Extracted GitHub PR URL |
| `github_pr_number` | int | Extracted PR number |

### ClickUpTaskDeploymentRelationship Fields

| Field | Type | Description |
|-------|------|-------------|
| `task_id` | string | ClickUp task ID |
| `deployment_id` | string | DevLake deployment ID |
| `task_created_at` | timestamp | Task creation time |
| `first_commit_at` | timestamp | First commit in PR |
| `pr_created_at` | timestamp | PR creation time |
| `pr_merged_at` | timestamp | PR merge time |
| `deployment_at` | timestamp | Deployment completion time |
| `planning_time` | int | Minutes: task → first commit |
| `code_time` | int | Minutes: first commit → PR merged |
| `deploy_time` | int | Minutes: PR merged → deployment |
| `total_cycle_time` | int | Minutes: task → deployment |

## API Reference

### ClickUp API Endpoints Used

- `GET /api/v2/list/{list_id}/task` - Collect tasks
  - Parameters: `page`, `include_closed=true`, `subtasks=true`, `date_updated_gt`

### Authentication

The plugin uses ClickUp Personal API Token authentication:

```
Authorization: pk_your_token_here
```

## Development

### Building the Plugin

```bash
cd backend/plugins/clickup
go build
```

### Running Tests

```bash
go test ./...
```

### Adding to DevLake

1. Register plugin in `backend/server/services/init.go`
2. Run migrations: `go run server/main.go migrate`
3. Restart DevLake server

## FAQ

**Q: Can I track multiple ClickUp lists?**
A: Yes, create separate pipeline runs with different `listId` values.

**Q: Does this replace DORA metrics?**
A: No, this is complementary. DORA metrics continue to work independently.

**Q: What if tasks don't have PR links?**
A: They'll be collected but won't have cycle time calculations.

**Q: Can I use ClickUp custom fields instead of description parsing?**
A: Currently no, but this could be added in future versions.

**Q: How often should I run collection?**
A: Recommended: Daily for active projects, weekly for less active ones.

## License

Licensed under the Apache License, Version 2.0. See LICENSE file for details.

## Contributing

Contributions welcome! Please follow the DevLake contribution guidelines.
