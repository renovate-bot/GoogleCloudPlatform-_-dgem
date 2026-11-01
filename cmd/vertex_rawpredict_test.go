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

import "testing"

// Phase 2 (private Vertex): a PSC dedicated endpoint is called with :rawPredict on the same *.prediction.vertexai.goog
// name; the gateway must keep that URL (not rewrite it to /invoke) and still find the endpoint ID for status checks.
func TestPrivateVertexRawPredictURL(t *testing.T) {
	const priv = "https://123.us-central1-456.prediction.vertexai.goog/v1/projects/456/locations/us-central1/endpoints/123:rawPredict"
	got, err := expandAndValidateVertexURL(priv)
	if err != nil || got != priv {
		t.Fatalf("expand(%s) = %q, %v; want unchanged", priv, got, err)
	}
	if id := extractEndpointIDFromURL(priv); id != "123" {
		t.Fatalf("endpoint id %q", id)
	}
	// Public invoke URLs are unchanged by this.
	const pub = "https://123.us-central1-456.prediction.vertexai.goog/v1/projects/456/locations/us-central1/endpoints/123/invoke/v1/chat/completions"
	if got, err := expandAndValidateVertexURL(pub); err != nil || got != pub {
		t.Fatalf("expand(pub) = %q, %v", got, err)
	}
	// Arbitrary hosts are still refused.
	if _, err := expandAndValidateVertexURL("https://10.0.0.5/v1/projects/456/locations/us-central1/endpoints/123:rawPredict"); err == nil {
		t.Fatal("non-Vertex host accepted")
	}
}
