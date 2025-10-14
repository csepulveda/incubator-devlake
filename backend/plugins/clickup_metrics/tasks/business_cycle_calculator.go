/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tasks

import (
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/clickup/models"
)

var CalculateClickUpBusinessCycleTimeMeta = plugin.SubTaskMeta{
	Name:             "calculateClickUpBusinessCycleTime",
	EntryPoint:       CalculateClickUpBusinessCycleTime,
	EnabledByDefault: true,
	Description:      "Calculate full business cycle time from ClickUp task to deployment",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET, plugin.DOMAIN_TYPE_CICD},
}

func CalculateClickUpBusinessCycleTime(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*ClickUpMetricsTaskData)

	taskCtx.GetLogger().Info("Starting ClickUp business cycle time calculation for project: %s", data.Options.ProjectName)

	// Get all ClickUp tasks with PR numbers from this project
	// We need to join with project_mapping to filter by project
	cursor, err := db.Cursor(
		dal.From(&models.ClickUpTask{}),
		dal.Join("LEFT JOIN pull_requests pr ON CAST(pr.pull_request_key AS CHAR) = CAST(_tool_clickup_tasks.git_hub_pr_number AS CHAR)"),
		dal.Join("LEFT JOIN project_mapping pm ON (pm.row_id = pr.base_repo_id AND pm.table = 'repos')"),
		dal.Where("_tool_clickup_tasks.git_hub_pr_number > 0 AND pm.project_name = ?", data.Options.ProjectName),
	)
	if err != nil {
		return err
	}
	defer cursor.Close()

	processedCount := 0
	deploymentsFound := 0

	for cursor.Next() {
		var task models.ClickUpTask
		err = db.Fetch(cursor, &task)
		if err != nil {
			return err
		}

		// Query to get PR and deployment information
		var prData struct {
			Id              string    // Full PR ID for joining with pull_request_commits
			PullRequestKey  string    // PR number
			CreatedDate     time.Time
			MergedDate      *time.Time
			MergeCommitSha  string
			FirstCommitSha  string
			FirstCommitDate *time.Time
		}

		// Method 1: Try to find PR by task ID in title or branch name
		// This is the preferred method (like Jira does)
		prResults := make([]struct {
			Id             string
			PullRequestKey string
			CreatedDate    time.Time
			MergedDate     *time.Time
			MergeCommitSha string
		}, 0, 1)

		err = db.All(
			&prResults,
			dal.From("pull_requests"),
			dal.Select("id, pull_request_key, created_date, merged_date, merge_commit_sha"),
			dal.Where("title LIKE ? OR head_ref LIKE ?", "%"+task.Id+"%", "%"+task.Id+"%"),
			dal.Orderby("created_date ASC"),
			dal.Limit(1),
		)

		// Method 2: If not found by task ID, try using the PR number from task (extracted from comments/description)
		if (err != nil || len(prResults) == 0) && task.GitHubPRNumber > 0 {
			err = db.All(
				&prResults,
				dal.From("pull_requests"),
				dal.Select("id, pull_request_key, created_date, merged_date, merge_commit_sha"),
				dal.Where("pull_request_key = ?", task.GitHubPRNumber),
				dal.Limit(1),
			)
		}

		if err != nil || len(prResults) == 0 {
			taskCtx.GetLogger().Debug("PR not found for task %s (pr_number: %d)", task.Id, task.GitHubPRNumber)
			continue
		}

		// Copy first result to prData
		prData.Id = prResults[0].Id
		prData.PullRequestKey = prResults[0].PullRequestKey
		prData.CreatedDate = prResults[0].CreatedDate
		prData.MergedDate = prResults[0].MergedDate
		prData.MergeCommitSha = prResults[0].MergeCommitSha

		// Get first commit date from pull_request_commits
		firstCommits := make([]struct {
			CommitAuthoredDate time.Time
			CommitSha          string
		}, 0, 1)

		err = db.All(
			&firstCommits,
			dal.From("pull_request_commits"),
			dal.Select("commit_authored_date, commit_sha"),
			dal.Where("pull_request_id = ?", prData.Id),
			dal.Orderby("commit_authored_date ASC"),
			dal.Limit(1),
		)

		if err == nil && len(firstCommits) > 0 {
			prData.FirstCommitDate = &firstCommits[0].CommitAuthoredDate
			prData.FirstCommitSha = firstCommits[0].CommitSha
		}

		// Get deployment info using DORA's strategy with merge_commit_sha
		// This works with both normal merges and squash merges
		var deployment struct {
			DeploymentId string
			FinishedDate time.Time
		}

		if prData.MergeCommitSha != "" {
			taskCtx.GetLogger().Debug("Looking for deployment of merge commit %s for PR #%d (task %s)", prData.MergeCommitSha, task.GitHubPRNumber, task.Id)

			// Strategy: Use commits_diffs like DORA does
			// This table contains all commits between deployments and works with squash commits
			deploymentCommits := make([]*struct {
				DeploymentId string
				FinishedDate time.Time
			}, 0, 1)

			err = db.All(
				&deploymentCommits,
				dal.Select("dc.cicd_deployment_id as deployment_id, d.finished_date"),
				dal.From("cicd_deployment_commits dc"),
				dal.Join("LEFT JOIN cicd_deployment_commits p ON (dc.prev_success_deployment_commit_id = p.id)"),
				dal.Join("INNER JOIN commits_diffs cd ON (cd.new_commit_sha = dc.commit_sha AND cd.old_commit_sha = COALESCE(p.commit_sha, ''))"),
				dal.Join("INNER JOIN cicd_deployments d ON (d.id = dc.cicd_deployment_id)"),
				dal.Where("dc.prev_success_deployment_commit_id <> ''"),
				dal.Where("dc.environment = ?", "PRODUCTION"),
				dal.Where("cd.commit_sha = ? AND d.result = ?", prData.MergeCommitSha, "SUCCESS"),
				dal.Orderby("d.finished_date ASC"),
				dal.Limit(1),
			)

			if err == nil && len(deploymentCommits) > 0 {
				deployment.DeploymentId = deploymentCommits[0].DeploymentId
				deployment.FinishedDate = deploymentCommits[0].FinishedDate
				taskCtx.GetLogger().Info("Found PRODUCTION deployment for task %s: %s (finished: %v)", task.Id, deployment.DeploymentId, deployment.FinishedDate)
			} else {
				// Fallback: Try without PRODUCTION filter
				taskCtx.GetLogger().Debug("No PRODUCTION deployment found for merge commit %s, trying any successful deployment", prData.MergeCommitSha)

				err = db.All(
					&deploymentCommits,
					dal.Select("dc.cicd_deployment_id as deployment_id, d.finished_date"),
					dal.From("cicd_deployment_commits dc"),
					dal.Join("LEFT JOIN cicd_deployment_commits p ON (dc.prev_success_deployment_commit_id = p.id)"),
					dal.Join("INNER JOIN commits_diffs cd ON (cd.new_commit_sha = dc.commit_sha AND cd.old_commit_sha = COALESCE(p.commit_sha, ''))"),
					dal.Join("INNER JOIN cicd_deployments d ON (d.id = dc.cicd_deployment_id)"),
					dal.Where("dc.prev_success_deployment_commit_id <> ''"),
					dal.Where("cd.commit_sha = ? AND d.result = ?", prData.MergeCommitSha, "SUCCESS"),
					dal.Orderby("d.finished_date ASC"),
					dal.Limit(1),
				)

				if err == nil && len(deploymentCommits) > 0 {
					deployment.DeploymentId = deploymentCommits[0].DeploymentId
					deployment.FinishedDate = deploymentCommits[0].FinishedDate
					taskCtx.GetLogger().Info("Found deployment for task %s: %s (finished: %v)", task.Id, deployment.DeploymentId, deployment.FinishedDate)
				} else {
					taskCtx.GetLogger().Debug("No deployment found for task %s (merge_commit_sha: %s)", task.Id, prData.MergeCommitSha)
				}
			}
		} else {
			taskCtx.GetLogger().Debug("No merge_commit_sha for PR #%d (task %s), cannot find deployment", task.GitHubPRNumber, task.Id)
		}

		// Create or update relationship record
		relationship := &models.ClickUpTaskDeploymentRelationship{
			TaskId:        task.Id,
			ConnectionId:  task.ConnectionId, // Use connection ID from the task itself
			TaskCreatedAt: task.DateCreated,
			PRCreatedAt:   &prData.CreatedDate,
			PRMergedAt:    prData.MergedDate,
		}

		if prData.FirstCommitDate != nil {
			relationship.FirstCommitAt = prData.FirstCommitDate
		}

		if deployment.DeploymentId != "" {
			relationship.DeploymentId = deployment.DeploymentId
			relationship.DeploymentAt = &deployment.FinishedDate
		}

		// Calculate cycle time metrics (in minutes)
		calculateMetrics(relationship)

		// Upsert the relationship
		err = db.CreateOrUpdate(relationship)
		if err != nil {
			taskCtx.GetLogger().Warn(err, "failed to save relationship", "task_id", task.Id)
		} else {
			processedCount++
			if deployment.DeploymentId != "" {
				deploymentsFound++
			}
		}

		// Report progress for each processed task
		taskCtx.IncProgress(1)
	}

	taskCtx.GetLogger().Info("Business cycle calculation complete: %d tasks processed, %d with deployments", processedCount, deploymentsFound)

	return nil
}

func calculateMetrics(rel *models.ClickUpTaskDeploymentRelationship) {
	// Planning Time: TaskCreated → FirstCommit
	if rel.FirstCommitAt != nil {
		planningMinutes := int(rel.FirstCommitAt.Sub(rel.TaskCreatedAt).Minutes())
		rel.PlanningTime = &planningMinutes
	}

	// Code Time: FirstCommit → PRMerged
	if rel.FirstCommitAt != nil && rel.PRMergedAt != nil {
		codeMinutes := int(rel.PRMergedAt.Sub(*rel.FirstCommitAt).Minutes())
		rel.CodeTime = &codeMinutes
	}

	// Deploy Time: PRMerged → Deployment
	if rel.PRMergedAt != nil && rel.DeploymentAt != nil {
		deployMinutes := int(rel.DeploymentAt.Sub(*rel.PRMergedAt).Minutes())
		rel.DeployTime = &deployMinutes
	}

	// Total Cycle Time: TaskCreated → Deployment
	if rel.DeploymentAt != nil {
		totalMinutes := int(rel.DeploymentAt.Sub(rel.TaskCreatedAt).Minutes())
		rel.TotalCycleTime = &totalMinutes
	}
}
