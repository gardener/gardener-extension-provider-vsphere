/*
 * Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package ensurer

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/pkg/errors"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/bindings"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/core"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/protocol"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/protocol/client"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/security"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra"

	vinfra "github.com/gardener/gardener-extension-provider-vsphere/pkg/vsphere/infrastructure"
)

func Process(req *http.Request) error {
	fmt.Print(httputil.DumpRequest(req, true))
	oldAuthHeader := req.Header.Get("Authorization")
	newAuthHeader := strings.Replace(oldAuthHeader, "Basic", "Remote", 1)
	req.Header.Set("Authorization", newAuthHeader)
	return nil
}

func createHttpClient(nsxtConfig *vinfra.NSXTConfig) (*string, *http.Client, error) {
	url := fmt.Sprintf("https://%s", nsxtConfig.Host)

	tlsConfig, err := getConnectorTLSConfig(nsxtConfig.InsecureFlag, nsxtConfig.ClientAuthCertFile, nsxtConfig.ClientAuthKeyFile, nsxtConfig.CAFile)
	if err != nil {
		return nil, nil, err
	}
	httpClient := http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: tlsConfig,
		},
	}
	return &url, &httpClient, nil
}

func addSecurityContext(connector client.Connector, nsxtConfig *vinfra.NSXTConfig) error {
	if len(nsxtConfig.ClientAuthCertFile) > 0 {
		securityCtx := core.NewSecurityContextImpl()
		if len(nsxtConfig.VMCAccessToken) > 0 {
			if nsxtConfig.VMCAuthHost == "" {
				return fmt.Errorf("vmc auth host must be provided if auth token is provided")
			}

			apiToken, err := getAPIToken(nsxtConfig.VMCAuthHost, nsxtConfig.VMCAccessToken)
			if err != nil {
				return err
			}

			securityCtx.SetProperty(security.AUTHENTICATION_SCHEME_ID, security.OAUTH_SCHEME_ID)
			securityCtx.SetProperty(security.ACCESS_TOKEN, apiToken)
		} else {
			if nsxtConfig.User == "" {
				return fmt.Errorf("username must be provided")
			}

			if nsxtConfig.Password == "" {
				return fmt.Errorf("password must be provided")
			}

			securityCtx.SetProperty(security.AUTHENTICATION_SCHEME_ID, security.USER_PASSWORD_SCHEME_ID)
			securityCtx.SetProperty(security.USER_KEY, nsxtConfig.User)
			securityCtx.SetProperty(security.PASSWORD_KEY, nsxtConfig.Password)
		}
		connector.SetSecurityContext(securityCtx)
	}
	return nil
}

func createConnectorNiceError(nsxtConfig *vinfra.NSXTConfig) (client.Connector, error) {
	connector, err := createConnector(nsxtConfig)
	if err != nil {
		submsg := ""
		if strings.Contains(err.Error(), "com.vmware.vapi.std.errors.unauthorized") {
			submsg = ". Please check credentials in provider-specific secret"
		}
		return nil, errors.Wrapf(err, "creating NSX-T connector failed%s", submsg)
	}
	return connector, nil
}

func buildRestMetadata() protocol.OperationRestMetadata {
	fields := map[string]bindings.BindingType{}
	fieldNameMap := map[string]string{}
	paramsTypeMap := map[string]bindings.BindingType{}
	pathParams := map[string]string{}
	queryParams := map[string]string{}
	headerParams := map[string]string{}
	dispatchHeaderParams := map[string]string{}
	bodyFieldsMap := map[string]string{}
	resultHeaders := map[string]string{}
	errorHeaders := map[string]map[string]string{}
	return protocol.NewOperationRestMetadata(
		fields,
		fieldNameMap,
		paramsTypeMap,
		pathParams,
		queryParams,
		headerParams,
		dispatchHeaderParams,
		bodyFieldsMap,
		"",
		"",
		"GET",
		"/api/v1/node/version",
		"",
		resultHeaders,
		200,
		"",
		errorHeaders,
		map[string]int{"InvalidRequest": 400, "Unauthorized": 403, "ServiceUnavailable": 503, "InternalServerError": 500, "NotFound": 404})
}

func createConnector(nsxtConfig *vinfra.NSXTConfig) (client.Connector, error) {
	url, httpClient, err := createHttpClient(nsxtConfig)
	if err != nil {
		return nil, err
	}
	connectorOptions := []client.ConnectorOption{
		client.UsingRest(nil),
		client.WithHttpClient(httpClient),
	}
	if nsxtConfig.RemoteAuth {
		connectorOptions = append(connectorOptions, client.WithRequestProcessors(Process))
	}
	connector := client.NewConnector(*url, connectorOptions...)
	err = addSecurityContext(connector, nsxtConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to addSecurityContext: %v", err)
	}

	// perform API call to check connector
	_, err = infra.NewTier0sClient(connector).List(nil, nil, nil, nil, nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "Connection to NSX-T API failed (cannot list tier-0 gateways). Please check your connection settings.")
	}

	return connector, nil
}

func getConnectorTLSConfig(insecure bool, clientCertFile string, clientKeyFile string, caFile string) (*tls.Config, error) {
	tlsConfig := tls.Config{InsecureSkipVerify: insecure}

	if len(clientCertFile) > 0 {
		if len(clientKeyFile) == 0 {
			return nil, fmt.Errorf("Please provide key file for client certificate")
		}

		cert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("Failed to load client cert/key pair: %v", err)
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if len(caFile) > 0 {
		caCert, err := ioutil.ReadFile(caFile)
		if err != nil {
			return nil, err
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		tlsConfig.RootCAs = caCertPool
	}

	return &tlsConfig, nil
}

type jwtToken struct {
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    string `json:"expires_in"`
	Scope        string `json:"scope"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func getAPIToken(vmcAuthHost string, vmcAccessToken string) (string, error) {

	payload := strings.NewReader("refresh_token=" + vmcAccessToken)
	req, _ := http.NewRequest("POST", "https://"+vmcAuthHost, payload)

	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return "", err
	}

	if res.StatusCode != 200 {
		b, _ := ioutil.ReadAll(res.Body)
		return "", fmt.Errorf("Unexpected status code %d trying to get auth token. %s", res.StatusCode, string(b))
	}

	defer res.Body.Close()
	token := jwtToken{}
	err = json.NewDecoder(res.Body).Decode(&token)
	if err != nil {
		return "", errors.Wrapf(err, "Decoding token failed with")
	}

	return token.AccessToken, nil
}
