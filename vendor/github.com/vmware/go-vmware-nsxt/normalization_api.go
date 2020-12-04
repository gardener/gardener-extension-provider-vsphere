/* Copyright © 2017 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: BSD-2-Clause

   Generated by: https://github.com/swagger-api/swagger-codegen.git */

package nsxt

import (
	"context"
	"encoding/json"
	"github.com/vmware/go-vmware-nsxt/normalization"
	"net/http"
	"net/url"
	"strings"
)

// Linger please
var (
	_ context.Context
)

type NormalizationApiService service

/* NormalizationApiService Get normalizations based on the query parameters
Returns the list of normalized resources based on the query parameters. Id and Type of the resource on which the normalizations is to be performed, are to be specified as query parameters in the URI. The target resource types to which normalization is to be done should also be specified as query parameter.
* @param ctx context.Context Authentication Context
@param preferredNormalizationType Resource type valid for use as target in normalization API.
@param resourceId Identifier of the resource on which normalization is to be performed
@param resourceType Resource type valid for use as source in normalization API.
@param optional (nil or map[string]interface{}) with one or more of:
    @param "cursor" (string) Opaque cursor to be used for getting next page of records (supplied by current result page)
    @param "includedFields" (string) Comma separated list of fields that should be included to result of query
    @param "pageSize" (int64) Maximum number of results to return in this page (server may return fewer)
    @param "sortAscending" (bool)
    @param "sortBy" (string) Field by which records are sorted
@return normalization.NormalizedResourceListResult*/
func (a *NormalizationApiService) GetNormalizations(ctx context.Context, preferredNormalizationType string, resourceId string, resourceType string, localVarOptionals map[string]interface{}) (normalization.NormalizedResourceListResult, *http.Response, error) {
	var (
		localVarHttpMethod = strings.ToUpper("Get")
		localVarPostBody   interface{}
		localVarFileName   string
		localVarFileBytes  []byte
		successPayload     normalization.NormalizedResourceListResult
	)

	// create path and map variables
	localVarPath := a.client.cfg.BasePath + "/normalizations"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := url.Values{}
	localVarFormParams := url.Values{}

	if err := typeCheckParameter(localVarOptionals["cursor"], "string", "cursor"); err != nil {
		return successPayload, nil, err
	}
	if err := typeCheckParameter(localVarOptionals["includedFields"], "string", "includedFields"); err != nil {
		return successPayload, nil, err
	}
	if err := typeCheckParameter(localVarOptionals["pageSize"], "int64", "pageSize"); err != nil {
		return successPayload, nil, err
	}
	if err := typeCheckParameter(localVarOptionals["sortAscending"], "bool", "sortAscending"); err != nil {
		return successPayload, nil, err
	}
	if err := typeCheckParameter(localVarOptionals["sortBy"], "string", "sortBy"); err != nil {
		return successPayload, nil, err
	}

	if localVarTempParam, localVarOk := localVarOptionals["cursor"].(string); localVarOk {
		localVarQueryParams.Add("cursor", parameterToString(localVarTempParam, ""))
	}
	if localVarTempParam, localVarOk := localVarOptionals["includedFields"].(string); localVarOk {
		localVarQueryParams.Add("included_fields", parameterToString(localVarTempParam, ""))
	}
	if localVarTempParam, localVarOk := localVarOptionals["pageSize"].(int64); localVarOk {
		localVarQueryParams.Add("page_size", parameterToString(localVarTempParam, ""))
	}
	localVarQueryParams.Add("preferred_normalization_type", parameterToString(preferredNormalizationType, ""))
	localVarQueryParams.Add("resource_id", parameterToString(resourceId, ""))
	localVarQueryParams.Add("resource_type", parameterToString(resourceType, ""))
	if localVarTempParam, localVarOk := localVarOptionals["sortAscending"].(bool); localVarOk {
		localVarQueryParams.Add("sort_ascending", parameterToString(localVarTempParam, ""))
	}
	if localVarTempParam, localVarOk := localVarOptionals["sortBy"].(string); localVarOk {
		localVarQueryParams.Add("sort_by", parameterToString(localVarTempParam, ""))
	}
	// to determine the Content-Type header
	localVarHttpContentTypes := []string{"application/json"}

	// set Content-Type header
	localVarHttpContentType := selectHeaderContentType(localVarHttpContentTypes)
	if localVarHttpContentType != "" {
		localVarHeaderParams["Content-Type"] = localVarHttpContentType
	}

	// to determine the Accept header
	localVarHttpHeaderAccepts := []string{
		"application/json",
	}

	// set Accept header
	localVarHttpHeaderAccept := selectHeaderAccept(localVarHttpHeaderAccepts)
	if localVarHttpHeaderAccept != "" {
		localVarHeaderParams["Accept"] = localVarHttpHeaderAccept
	}
	r, err := a.client.prepareRequest(ctx, localVarPath, localVarHttpMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, localVarFileName, localVarFileBytes)
	if err != nil {
		return successPayload, nil, err
	}

	localVarHttpResponse, err := a.client.callAPI(r)
	if err != nil || localVarHttpResponse == nil {
		return successPayload, localVarHttpResponse, err
	}
	defer localVarHttpResponse.Body.Close()
	if localVarHttpResponse.StatusCode >= 300 {
		return successPayload, localVarHttpResponse, reportError(localVarHttpResponse.Status)
	}

	if err = json.NewDecoder(localVarHttpResponse.Body).Decode(&successPayload); err != nil {
		return successPayload, localVarHttpResponse, err
	}

	return successPayload, localVarHttpResponse, err
}