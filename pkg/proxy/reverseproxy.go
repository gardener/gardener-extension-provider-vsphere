/*
 * Copyright (c) 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *
 *  You may obtain a copy of the License at
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"golang.org/x/oauth2/google"
)

// ReverseProxy provides the runtime configuration of the Reverse Proxy
type ReverseProxy struct {
	Debug           bool
	Port            int
	ProjectID       string
	KeyFile         string
	CertificateFile string
	clusterInfo     *Cache
}

func (p *ReverseProxy) retrieveClusterInfo(ctx context.Context) error {
	credentials, err := google.FindDefaultCredentials(ctx,
		"https://www.googleapis.com/auth/cloud-platform.read-only")
	if err != nil {
		return err
	}
	if p.ProjectID == "" {
		p.ProjectID = credentials.ProjectID
	}
	if p.ProjectID == "" {
		return fmt.Errorf("specify a --project as there is no default one")
	}

	p.clusterInfo, err = NewCache(ctx, p.ProjectID, credentials, 5*time.Minute)
	return err
}

func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintf(w, "service is healthy\n")
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clusterInfo := p.clusterInfo.GetConnectInfoForEndpoint(r.Host)
	if clusterInfo == nil {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(fmt.Sprintf("%s is not a cluster endpoint", r.Host)))
		return
	}

	targetURL, err := url.Parse(fmt.Sprintf("https://%s", r.Host))
	if clusterInfo == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf("failed to parse URL https://%s, %s", r.Host, err)))
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: clusterInfo.RootCAs,
		},
	}

	proxy.ServeHTTP(w, r)
}

// Run the reverse proxy until stopped
func (p *ReverseProxy) Run() error {
	var err error

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err = p.retrieveClusterInfo(ctx); err != nil {
		return fmt.Errorf("failed to retrieve cluster information, %s", err)
	}

	http.Handle("/", p)
	http.HandleFunc("/__health", healthCheckHandler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", p.Port),
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	err = srv.ListenAndServeTLS(p.CertificateFile, p.KeyFile)
	return err
}
