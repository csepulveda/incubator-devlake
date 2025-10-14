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

package api

import (
	"net/http"
	"strings"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	dsmodels "github.com/apache/incubator-devlake/helpers/pluginhelper/api/models"
	"github.com/apache/incubator-devlake/plugins/clickup/models"
)

type ClickUpRemotePagination struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

type TeamResponse struct {
	Teams []struct {
		Id     string `json:"id"`
		Name   string `json:"name"`
		Color  string `json:"color"`
		Avatar string `json:"avatar"`
	} `json:"teams"`
}

type SpacesResponse struct {
	Spaces []struct {
		Id       string `json:"id"`
		Name     string `json:"name"`
		Private  bool   `json:"private"`
		Archived bool   `json:"archived"`
	} `json:"spaces"`
}

func queryClickUpRemoteScopes(
	apiClient plugin.ApiClient,
	groupId string,
	page ClickUpRemotePagination,
	search string,
) (
	children []dsmodels.DsRemoteApiScopeListEntry[models.ClickUpScope],
	nextPage *ClickUpRemotePagination,
	err errors.Error,
) {
	// If groupId is empty, list teams and spaces
	if groupId == "" {
		return listTeamsAndSpaces(apiClient, search)
	}

	// If groupId is provided, it's a space ID - list folders in that space
	return listFoldersInSpace(apiClient, groupId, search)
}

func listTeamsAndSpaces(
	apiClient plugin.ApiClient,
	search string,
) (
	children []dsmodels.DsRemoteApiScopeListEntry[models.ClickUpScope],
	nextPage *ClickUpRemotePagination,
	err errors.Error,
) {
	// Get authorized teams (workspaces)
	var teamsRes *http.Response
	teamsRes, err = apiClient.Get("team", nil, nil)
	if err != nil {
		return
	}

	teamsResponse := &TeamResponse{}
	err = api.UnmarshalResponse(teamsRes, teamsResponse)
	if err != nil {
		return
	}

	// For each team, get spaces
	for _, team := range teamsResponse.Teams {
		var spacesRes *http.Response
		spacesRes, err = apiClient.Get("team/"+team.Id+"/space", nil, nil)
		if err != nil {
			continue
		}

		spacesResponse := &SpacesResponse{}
		err = api.UnmarshalResponse(spacesRes, spacesResponse)
		if err != nil {
			continue
		}

		// Add each space as a scope
		for _, space := range spacesResponse.Spaces {
			if space.Archived {
				continue
			}

			if search != "" && !strings.Contains(strings.ToLower(space.Name), strings.ToLower(search)) {
				continue
			}

			fullName := team.Name + " / " + space.Name

			children = append(children, dsmodels.DsRemoteApiScopeListEntry[models.ClickUpScope]{
				Type:     api.RAS_ENTRY_TYPE_SCOPE,
				Id:       space.Id,
				Name:     space.Name,
				FullName: fullName,
				Data: &models.ClickUpScope{
					Scope: common.Scope{
						NoPKModel: common.NoPKModel{},
					},
					Id:   space.Id,
					Name: fullName,
				},
			})
		}
	}

	if len(children) > 0 {
		err = nil
	}

	return
}

type FoldersResponse struct {
	Folders []struct {
		Id       string `json:"id"`
		Name     string `json:"name"`
		Hidden   bool   `json:"hidden"`
		SpaceId  string `json:"space"`
		Archived bool   `json:"archived"`
	} `json:"folders"`
}

func listFoldersInSpace(
	apiClient plugin.ApiClient,
	spaceId string,
	search string,
) (
	children []dsmodels.DsRemoteApiScopeListEntry[models.ClickUpScope],
	nextPage *ClickUpRemotePagination,
	err errors.Error,
) {
	// This would be used if we want hierarchical folder selection in UI
	// For now, we'll handle folders in the collection logic
	return
}

func listClickUpRemoteScopes(
	connection *models.ClickUpConnection,
	apiClient plugin.ApiClient,
	groupId string,
	page ClickUpRemotePagination,
) (
	[]dsmodels.DsRemoteApiScopeListEntry[models.ClickUpScope],
	*ClickUpRemotePagination,
	errors.Error,
) {
	return queryClickUpRemoteScopes(apiClient, groupId, page, "")
}

func searchClickUpRemoteScopes(
	apiClient plugin.ApiClient,
	params *dsmodels.DsRemoteApiScopeSearchParams,
) (
	children []dsmodels.DsRemoteApiScopeListEntry[models.ClickUpScope],
	err errors.Error,
) {
	children, _, err = queryClickUpRemoteScopes(apiClient, "", ClickUpRemotePagination{
		Page:     params.Page,
		PageSize: params.PageSize,
	}, params.Search)
	return
}

// RemoteScopes list all available scopes (spaces) for this connection
func RemoteScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return raScopeList.Get(input)
}

// SearchRemoteScopes search remote scopes
func SearchRemoteScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return raScopeSearch.Get(input)
}

// Proxy forwards API requests to ClickUp
func Proxy(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return raProxy.Proxy(input)
}
