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

package impl

import (
	"encoding/json"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	coreModels "github.com/apache/incubator-devlake/core/models"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/clickup_metrics/tasks"
)

var _ interface {
	plugin.PluginMeta
	plugin.PluginTask
	plugin.PluginModel
	plugin.MetricPluginBlueprintV200
} = (*ClickUpMetrics)(nil)

type ClickUpMetrics struct{}

func (p ClickUpMetrics) Description() string {
	return "Calculate ClickUp business cycle time metrics"
}

func (p ClickUpMetrics) Name() string {
	return "clickup_metrics"
}

func (p ClickUpMetrics) IsProjectMetric() bool {
	return true
}

func (p ClickUpMetrics) RunAfter() ([]string, errors.Error) {
	// Must run after both DORA and refdiff to ensure commits_diffs table exists
	return []string{"dora", "refdiff"}, nil
}

func (p ClickUpMetrics) RequiredDataEntities() (data []map[string]interface{}, err errors.Error) {
	return []map[string]interface{}{
		{
			"model": "cicd_deployments",
			"requiredFields": map[string]string{
				"column":        "result",
				"expectedValue": "SUCCESS",
			},
		},
		{
			"model": "pull_requests",
			"requiredFields": map[string]string{
				"column":        "merged_date",
				"expectedValue": "not null",
			},
		},
	}, nil
}

func (p ClickUpMetrics) GetTablesInfo() []dal.Tabler {
	// This plugin doesn't create its own tables, it updates ClickUp plugin tables
	return []dal.Tabler{}
}

func (p ClickUpMetrics) SubTaskMetas() []plugin.SubTaskMeta {
	return []plugin.SubTaskMeta{
		tasks.CalculateClickUpBusinessCycleTimeMeta,
	}
}

func (p ClickUpMetrics) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
	op, err := tasks.DecodeAndValidateTaskOptions(options)
	if err != nil {
		return nil, err
	}
	return &tasks.ClickUpMetricsTaskData{
		Options: op,
	}, nil
}

func (p ClickUpMetrics) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/clickup_metrics"
}

func (p ClickUpMetrics) MigrationScripts() []plugin.MigrationScript {
	// No migrations needed - uses tables from clickup plugin
	return []plugin.MigrationScript{}
}

func (p ClickUpMetrics) MakeMetricPluginPipelinePlanV200(projectName string, options json.RawMessage) (coreModels.PipelinePlan, errors.Error) {
	// clickup_metrics must run AFTER dora completes (after stage 6 in DORA's plan)
	// DORA returns a 3-stage plan, so we add 3 empty stages before our actual stage
	// to ensure proper alignment when ParallelizePipelinePlans merges them
	plan := coreModels.PipelinePlan{
		{}, // Stage 1 - empty (aligns with DORA stage 1: generateDeployments)
		{}, // Stage 2 - empty (aligns with DORA stage 2: refdiff calculateDeploymentCommitsDiff)
		{}, // Stage 3 - empty (aligns with DORA stage 3: calculateChangeLeadTime)
		{   // Stage 4 - our actual work (runs AFTER all DORA stages)
			{
				Plugin: "clickup_metrics",
				Options: map[string]interface{}{
					"projectName": projectName,
				},
			},
		},
	}
	return plan, nil
}
