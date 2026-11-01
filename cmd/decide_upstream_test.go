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
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/dgem/pkg/client"
)

// /api/decide: a request the serving image refused is the caller's (4xx, WARNING) and is never cold-start retried;
// unreachable or 5xx upstreams stay 502.
func TestDecideUpstreamStatus(t *testing.T) {
	refused := &client.HTTPError{StatusCode: 400, Body: `{"error": {"message": "question 'wide_choice': at most 26 alternatives", "type": "invalid_request_error"}}`}
	for _, tc := range []struct {
		name  string
		err   error
		want  int
		retry bool
	}{
		{"server refused (>26 options)", refused, 422, false},
		{"server refused, wrapped", fmt.Errorf("decide: %w", refused), 422, false},
		{"server 422", &client.HTTPError{StatusCode: 422, Body: `{"error": {"message": "Failed to fetch media"}}`}, 422, false},
		{"rate limited", &client.HTTPError{StatusCode: 429, Body: `slow down`}, 502, true},
		{"server 503 (cold start)", &client.HTTPError{StatusCode: 503, Body: `Service Unavailable`}, 502, true},
		{"server crash", &client.HTTPError{StatusCode: 500, Body: `boom`}, 502, false},
		{"unreachable", errors.New("dial tcp: Connection refused"), 502, true},
	} {
		if got := decideUpstreamStatus(tc.err); got != tc.want {
			t.Errorf("%s: status %d, want %d", tc.name, got, tc.want)
		}
		if got := isColdStartRetryable(tc.err); got != tc.retry {
			t.Errorf("%s: retryable %v, want %v", tc.name, got, tc.retry)
		}
	}
}

func TestSystemOneRawCallerAndForwardBody(t *testing.T) {
	old := systemOneRawUsers
	defer func() { systemOneRawUsers = old }()
	systemOneRawUsers = []string{"client-sa@example.iam.gserviceaccount.com"}
	if !systemOneRawCaller("Client-SA@example.iam.gserviceaccount.com") || systemOneRawCaller("other@example.com") || systemOneRawCaller("") {
		t.Error("caller list match")
	}
	systemOneRawUsers = []string{"*"}
	if !systemOneRawCaller("anyone@example.com") {
		t.Error("* should match")
	}
	systemOneRawUsers = nil
	if systemOneRawCaller("client-sa@example.iam.gserviceaccount.com") {
		t.Error("empty list must match nobody")
	}

	plain := []byte(`{"state": "board\nrow 2", "questions": {"q": {"type": "noul"}}}`)
	req, _, err := decodeSystemOneAdapterRequest(plain)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := systemOneForwardBody(plain, req); err != nil || string(got) != string(plain) {
		t.Errorf("body without media must be forwarded byte for byte: %q %v", got, err)
	}
	withMedia := []byte(`{"state": "s", "questions": {"q": {"type": "noul"}}, "media": [{"mime_type": "image/png", "data": "AAAA", "name": "f.png"}]}`)
	req, _, err = decodeSystemOneAdapterRequest(withMedia)
	if err != nil {
		t.Fatal(err)
	}
	got, err := systemOneForwardBody(withMedia, req)
	if err != nil || strings.Contains(string(got), `"media"`) || !strings.Contains(string(got), `"images":["data:image/png;base64,AAAA"]`) {
		t.Errorf("media not converted: %s %v", got, err)
	}
}
