// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"testing"

	"github.com/GoogleCloudPlatform/dgem/pkg/client"
)

// A private (PSC) endpoint host is <ID>-<REGION>-<PROJECT_NUMBER>.prediction.p.vertexai.goog, reached over
// http on port 80 inside the VPC (port 443 serves a self-signed certificate).
func TestPrivatePSCVertexURL(t *testing.T) {
	const path = "/v1/projects/456/locations/us-central1/endpoints/123:rawPredict"
	const host = "123-us-central1-456.prediction.p.vertexai.goog"
	for _, u := range []string{"http://" + host + path, "https://" + host + path, "http://" + host + ":80" + path} {
		got, err := expandAndValidateVertexURL(u)
		if err != nil || got != u {
			t.Errorf("expand(%s) = %q, %v; want unchanged", u, got, err)
		}
		if id := extractEndpointIDFromURL(u); id != "123" {
			t.Errorf("%s: endpoint id %q", u, id)
		}
		if !client.IsVertexEndpointURL(u) || client.NormalizeVertexEndpointURL(u) != u {
			t.Errorf("%s: client must treat it as Vertex and keep it unchanged (got %q)", u, client.NormalizeVertexEndpointURL(u))
		}
		if !isRemoteGCPURL(u) {
			t.Errorf("%s: not treated as a remote GCP URL (no token would be minted)", u)
		}
	}
	for _, bad := range []string{
		"http://" + host + "/v1/projects/456/locations/us-central1/endpoints/123/invoke/v1/chat/completions", // PSC has no invoke routes
		"http://" + host + "/", // no :rawPredict
		"http://123.us-central1-456.prediction.vertexai.goog" + path, // http on the public host
		"http://us-central1-aiplatform.googleapis.com/v1/projects/p/locations/us-central1/endpoints/123:rawPredict",
		"http://evil.example/x.prediction.p.vertexai.goog" + path, // suffix in the path, not the host
		"http://prediction.p.vertexai.goog.evil.example" + path,
	} {
		if got, err := expandAndValidateVertexURL(bad); err == nil {
			t.Errorf("accepted %s -> %s", bad, got)
		}
	}
}
