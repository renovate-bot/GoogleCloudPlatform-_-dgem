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

package decisionindex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/dgem/pkg/client"
)

// timedServer answers every pass with the first option and reports diagnostics.timing.total_ms = ms.
func timedServer(t *testing.T, ms float64, passes *int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(passes, 1)
		var body struct {
			Messages []struct{ Role, Content string } `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		var sys struct {
			Questions []struct {
				ID      string                  `json:"id"`
				Type    string                  `json:"type"`
				Options []struct{ Name string } `json:"options"`
			} `json:"questions"`
		}
		for _, m := range body.Messages {
			if m.Role == "system" {
				_ = json.Unmarshal([]byte(m.Content), &sys)
			}
		}
		answers := map[string]client.QuestionAnswer{}
		for _, q := range sys.Questions {
			if q.Type == "boolean" {
				answers[q.ID] = client.QuestionAnswer{Noul: 0.7}
				continue
			}
			p := map[string]float64{}
			for _, o := range q.Options {
				p[o.Name] = 0.1 / float64(maxInt(1, len(q.Options)-1))
			}
			p[q.Options[0].Name] = 0.9
			answers[q.ID] = client.QuestionAnswer{Choice: q.Options[0].Name, Probabilities: p}
		}
		resp := client.StructuredDecisionResponse{Answers: answers}
		resp.Diagnostics.Timing.TotalMs = ms
		env, _ := json.Marshal(resp)
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": map[string]any{"role": "assistant", "content": string(env)}}}})
	}))
}

func TestEvaluationTraceServerMs(t *testing.T) {
	var passes int32
	srv := timedServer(t, 40, &passes)
	defer srv.Close()
	cli := client.NewClient(srv.URL, "dgemma", 10*time.Second)

	// One pass: server time is that pass's total_ms.
	one := SystemOneRequest{State: "x", Questions: map[string]SystemOneQuestion{
		"u": {Type: "noul", Instructions: "Urgent?"}}}
	resp, err := ExecuteSystemOne(context.Background(), cli, one, DefaultEngineOptions())
	if err != nil {
		t.Fatal(err)
	}
	if resp.EvaluationTrace.ServerMs != 40 || resp.EvaluationTrace.ForwardPasses != 1 {
		t.Fatalf("one pass: server_ms %v passes %d", resp.EvaluationTrace.ServerMs, resp.EvaluationTrace.ForwardPasses)
	}

	// A wide question (bracket rounds): server time sums every pass the adapter made.
	crit := map[string]string{}
	for i := 0; i < 60; i++ {
		crit[fmt.Sprintf("opt_%02d", i)] = fmt.Sprintf("option %d", i)
	}
	atomic.StoreInt32(&passes, 0)
	wide := SystemOneRequest{State: "x", Questions: map[string]SystemOneQuestion{
		"w": {Type: "choice", Instructions: "Which?", Criteria: crit}}}
	resp, err = ExecuteSystemOne(context.Background(), cli, wide, DefaultEngineOptions())
	if err != nil {
		t.Fatal(err)
	}
	n := atomic.LoadInt32(&passes)
	if n < 2 {
		t.Fatalf("expected bracket passes, got %d", n)
	}
	if got, want := resp.EvaluationTrace.ServerMs, 40*float64(n); got != want {
		t.Fatalf("wide: server_ms %v, want %v over %d passes", got, want, n)
	}

	// A server without timing: server_ms omitted (0).
	atomic.StoreInt32(&passes, 0)
	srv0 := timedServer(t, 0, &passes)
	defer srv0.Close()
	resp, err = ExecuteSystemOne(context.Background(), client.NewClient(srv0.URL, "dgemma", 10*time.Second), one, DefaultEngineOptions())
	if err != nil || resp.EvaluationTrace.ServerMs != 0 {
		t.Fatalf("no timing: %v %v", err, resp.EvaluationTrace.ServerMs)
	}
	b, _ := json.Marshal(resp.EvaluationTrace)
	if json.Valid(b) && strings.Contains(string(b), "server_ms") {
		t.Fatalf("server_ms should be omitted when zero: %s", b)
	}
}
