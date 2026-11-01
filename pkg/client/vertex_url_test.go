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

package client

import "testing"

func TestNormalizeVertexEndpointURL(t *testing.T) {
	const ded = "https://123.us-central1-456.prediction.vertexai.goog/v1/projects/456/locations/us-central1/endpoints/123"
	const reg = "https://us-central1-aiplatform.googleapis.com/v1/projects/p/locations/us-central1/endpoints/123"
	for in, want := range map[string]string{
		ded + "/invoke/v1/chat/completions": ded + "/invoke/v1/chat/completions", // public dedicated, invoke routes
		ded + "/invoke":                     ded + "/invoke/v1/chat/completions",
		ded + "/invoke/v1/systemone":        ded + "/invoke/v1/systemone",
		ded + ":rawPredict":                 ded + ":rawPredict", // private (PSC) dedicated: kept, not rewritten to /invoke
		reg:                                 reg + ":rawPredict",
		reg + ":rawPredict":                 reg + ":rawPredict",
		reg + "/v1/chat/completions":        reg + ":rawPredict",
	} {
		if got := NormalizeVertexEndpointURL(in); got != want {
			t.Errorf("%s\n got  %s\n want %s", in, got, want)
		}
	}
	if !IsVertexEndpointURL(ded + ":rawPredict") {
		t.Error("private rawPredict URL not recognised as Vertex")
	}
}
