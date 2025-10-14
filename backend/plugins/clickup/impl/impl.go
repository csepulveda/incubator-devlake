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
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	coreModels "github.com/apache/incubator-devlake/core/models"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/clickup/api"
	"github.com/apache/incubator-devlake/plugins/clickup/models"
	"github.com/apache/incubator-devlake/plugins/clickup/models/migrationscripts"
	"github.com/apache/incubator-devlake/plugins/clickup/tasks"
)

var _ interface {
	plugin.PluginMeta
	plugin.PluginInit
	plugin.PluginTask
	plugin.PluginModel
	plugin.PluginApi
	plugin.DataSourcePluginBlueprintV200
	plugin.CloseablePluginTask
} = (*ClickUp)(nil)

type ClickUp struct{}

func (p ClickUp) Connection() dal.Tabler {
	return &models.ClickUpConnection{}
}

func (p ClickUp) Description() string {
	return "Collect ClickUp tasks to track full business cycle time from task creation to deployment"
}

func (p ClickUp) Name() string {
	return "clickup"
}

func (p ClickUp) RunAfter() ([]string, errors.Error) {
	// Business cycle calculation depends on commits_diffs from refdiff
	// which runs after DORA generates deployment commits
	return []string{"refdiff"}, nil
}

func (p ClickUp) Init(basicRes context.BasicRes) errors.Error {
	api.Init(basicRes, p)
	return nil
}

func (p ClickUp) ApiResources() map[string]map[string]plugin.ApiResourceHandler {
	return map[string]map[string]plugin.ApiResourceHandler{
		"test": {
			"POST": api.TestConnection,
		},
		"connections": {
			"POST": api.PostConnections,
			"GET":  api.ListConnections,
		},
		"connections/:connectionId": {
			"GET":    api.GetConnection,
			"PATCH":  api.PatchConnection,
			"DELETE": api.DeleteConnection,
		},
		"connections/:connectionId/test": {
			"POST": api.TestExistingConnection,
		},
		"connections/:connectionId/scopes": {
			"PUT": api.PutScopes,
			"GET": api.GetScopeList,
		},
		"connections/:connectionId/scopes/:scopeId": {
			"GET":    api.GetScope,
			"PATCH":  api.PatchScope,
			"DELETE": api.DeleteScope,
		},
		"connections/:connectionId/scope-configs": {
			"POST": api.PostScopeConfig,
			"GET":  api.GetScopeConfigList,
		},
		"connections/:connectionId/scope-configs/:scopeConfigId": {
			"GET":    api.GetScopeConfig,
			"PATCH":  api.PatchScopeConfig,
			"DELETE": api.DeleteScopeConfig,
		},
		"connections/:connectionId/remote-scopes": {
			"GET": api.RemoteScopes,
		},
		"connections/:connectionId/search-remote-scopes": {
			"GET": api.SearchRemoteScopes,
		},
		"connections/:connectionId/proxy/*path": {
			"GET": api.Proxy,
		},
	}
}

func (p ClickUp) GetTablesInfo() []dal.Tabler {
	return []dal.Tabler{
		&models.ClickUpConnection{},
		&models.ClickUpTask{},
		&models.ClickUpTaskDeploymentRelationship{},
	}
}

func (p ClickUp) SubTaskMetas() []plugin.SubTaskMeta {
	return []plugin.SubTaskMeta{
		// Simplified: single task collector that handles folders/lists internally
		plugin.SubTaskMeta{
			Name:             "collectTasksV2",
			EntryPoint:       tasks.CollectTasksV2,
			EnabledByDefault: true,
			Description:      "Collect all tasks from ClickUp space (including folders)",
			DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
		},
		tasks.ExtractTasksMeta,
		tasks.CollectCommentsMeta,
		tasks.ExtractCommentsMeta,
		tasks.EnrichTasksWithGitHubMeta,
		tasks.CalculateBusinessCycleTimeMeta,
	}
}

func (p ClickUp) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
	var op tasks.ClickUpOptions
	err := helper.Decode(options, &op, nil)
	if err != nil {
		return nil, err
	}

	connectionHelper := helper.NewConnectionHelper(
		taskCtx,
		nil,
		p.Name(),
	)

	connection := &models.ClickUpConnection{}
	err = connectionHelper.FirstById(connection, op.ConnectionId)
	if err != nil {
		return nil, errors.Default.Wrap(err, "unable to get ClickUp connection by the given connection ID")
	}

	apiClient, err := tasks.CreateApiClient(taskCtx, connection)
	if err != nil {
		return nil, errors.Default.Wrap(err, "unable to get ClickUp API client instance")
	}

	return &tasks.ClickUpTaskData{
		Options:   &op,
		ApiClient: apiClient,
	}, nil
}

func (p ClickUp) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/clickup"
}

func (p ClickUp) MigrationScripts() []plugin.MigrationScript {
	return migrationscripts.All()
}

func (p ClickUp) MakeDataSourcePipelinePlanV200(
	connectionId uint64,
	scopes []*coreModels.BlueprintScope,
) (coreModels.PipelinePlan, []plugin.Scope, errors.Error) {
	return api.MakeDataSourcePipelinePlanV200(p.SubTaskMetas(), connectionId, scopes)
}

func (p ClickUp) Close(taskCtx plugin.TaskContext) errors.Error {
	data, ok := taskCtx.GetData().(*tasks.ClickUpTaskData)
	if !ok {
		return errors.Default.New("GetData failed when try to close")
	}
	data.ApiClient.Release()
	return nil
}
