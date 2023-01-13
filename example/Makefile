# Copyright 2021 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# Creates a CA and serving certificate valid for webhook.pod-security-webhook.svc
.PHONY: certs
certs:
	rm -fr pki
	mkdir -p pki
	openssl genrsa -out /tmp/ca.key 2048
	openssl req -sha256 -new -x509 -days 3650 -key /tmp/ca.key -subj "/CN=validator-vsphere-webhook-ca-$(shell date +%s)" -out validator-vsphere-certs/ca.crt

	openssl req -sha256 -newkey rsa:2048 -nodes -keyout validator-vsphere-certs/tls.key -subj "/CN=gardener-extension-validator-vsphere.garden.svc" -out /tmp/tls.csr

	echo "subjectAltName=DNS:gardener-extension-validator-vsphere.garden.svc" > /tmp/extensions.txt
	echo "extendedKeyUsage=serverAuth" >> /tmp/extensions.txt
	openssl x509 -sha256 -req -extfile /tmp/extensions.txt -days 730 -in /tmp/tls.csr -CA validator-vsphere-certs/ca.crt -CAkey /tmp/ca.key -CAcreateserial -out validator-vsphere-certs/tls.crt
	yq e -i '.global.webhookConfig.caBundle |= load_str("validator-vsphere-certs/ca.crt")' gardener-local/validator-vsphere/values.yaml
	yq e -i '.global.webhookConfig.tls.crt |= load_str("validator-vsphere-certs/tls.crt")' gardener-local/validator-vsphere/values.yaml
	yq e -i '.global.webhookConfig.tls.key |= load_str("validator-vsphere-certs/tls.key")' gardener-local/validator-vsphere/values.yaml