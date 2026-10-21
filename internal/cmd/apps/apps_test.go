package apps

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"k8s.io/client-go/rest"
)

const currentDefinition = `{"description":"current","inputSchema":{"additionalProperties":false,"properties":{"a":{"const":1,"type":"number"},"b":{"const":2,"type":"number"},"c":{"const":3,"type":"number"},"d":{"const":4,"type":"number"}},"type":"object"}}`
const historyDefinition = `{"description":"old","inputSchema":{"additionalProperties":false,"properties":{"a":{"const":9007199254740993,"type":"number"},"b":{"const":18446744073709551615,"type":"number"},"c":{"const":123456789012345678901e-21,"type":"number"},"d":{"const":9007199254740993e1,"type":"number"}},"type":"object"}}`

func digest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return "sha256:" + hex.EncodeToString(sum[:])
}
func quoted(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

var currentJSON = `{"apiVersion":"apps.miloapis.com/v1alpha1","kind":"AppTemplate","metadata":{"name":"web-api","uid":"template-uid","resourceVersion":"opaque-rv-a","generation":2},"spec":{"definitionJSON":` + quoted(currentDefinition) + `,"digest":"` + digest(currentDefinition) + `"}}`
var historyJSON = `{"metadata":{"name":"template-uid.revision-uid","uid":"revision-uid"},"spec":{"templateUID":"template-uid","sequence":1,"digest":"` + digest(historyDefinition) + `","effectiveActorUID":"actor-uid","acceptedAt":"2026-10-20T00:00:00Z","definitionJSON":` + quoted(historyDefinition) + `}}`

func execute(t *testing.T, h http.HandlerFunc, args ...string) (string, error) {
	t.Helper()
	server := httptest.NewServer(h)
	defer server.Close()
	var out bytes.Buffer
	cmd := Command(Options{RESTConfig: func() (*rest.Config, error) {
		return &rest.Config{Host: server.URL + "/projects/project-a/control-plane", BearerToken: "fixture-token"}, nil
	}, Out: &out})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}
func checkRequest(t *testing.T, r *http.Request, method, suffix string) {
	t.Helper()
	if r.Method != method || r.URL.Path != "/projects/project-a/control-plane"+apiPath+suffix {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
	}
	if r.Header.Get("Authorization") != "Bearer fixture-token" {
		t.Error("configured identity missing")
	}
}
func checkHistoryQuery(t *testing.T, r *http.Request, uid string, token string) {
	t.Helper()
	query := r.URL.Query()
	if query.Get("limit") != "1" || query.Get("labelSelector") != "apps.miloapis.com/template-uid="+uid || query.Get("continue") != token || len(query) != 2+boolInt(token != "") {
		t.Errorf("unexpected history query: %v", query)
	}
}
func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
func revisionJSON(name, uid, templateUID string, sequence uint64) string {
	return fmt.Sprintf(`{"metadata":{"name":%q,"uid":%q},"spec":{"templateUID":%q,"sequence":%d,"digest":%q,"effectiveActorUID":"actor-uid","acceptedAt":"2026-10-20T00:00:00Z","definitionJSON":%s}}`, name, uid, templateUID, sequence, digest(historyDefinition), quoted(historyDefinition))
}
func reply(w http.ResponseWriter, s string) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, s)
}

func TestHistoryPinsCurrentIncarnationAndOmitsDefinitions(t *testing.T) {
	calls := 0
	out, err := execute(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch calls {
		case 1:
			checkRequest(t, r, http.MethodGet, "apptemplates/web-api")
			reply(w, currentJSON)
		case 2:
			checkRequest(t, r, http.MethodGet, "apptemplaterevisions")
			checkHistoryQuery(t, r, "template-uid", "")
			reply(w, `{"items":[`+historyJSON+`]}`)
		default:
			t.Error("unexpected request")
		}
	}, "templates", "history", "web-api")
	if err != nil || calls != 2 {
		t.Fatalf("history: calls=%d err=%v", calls, err)
	}
	if !strings.Contains(out, "template-uid.revision-uid") || strings.Contains(out, "definition") || strings.Contains(out, "MUST_NOT_PRINT") || strings.Contains(out, "different-uid") {
		t.Fatalf("history leaked or omitted data: %s", out)
	}
}
func TestDeletedHistoryUsesExplicitUIDWithoutNameLookup(t *testing.T) {
	calls := 0
	_, err := execute(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		checkRequest(t, r, http.MethodGet, "apptemplaterevisions")
		checkHistoryQuery(t, r, "old-uid", "")
		reply(w, `{"items":[]}`)
	}, "templates", "history", "--uid", "old-uid")
	if err != nil || calls != 1 {
		t.Fatalf("history: %v calls=%d", err, calls)
	}
}

func TestHistoryFollowsFilteredContinuationAndSortsCompleteResult(t *testing.T) {
	calls := 0
	out, err := execute(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		checkRequest(t, r, http.MethodGet, "apptemplaterevisions")
		switch calls {
		case 1:
			checkHistoryQuery(t, r, "template-uid", "")
			reply(w, `{"items":[`+revisionJSON("template-uid.second", "revision-second", "template-uid", 2)+`],"metadata":{"continue":"opaque+/="}}`)
		case 2:
			checkHistoryQuery(t, r, "template-uid", "opaque+/=")
			reply(w, `{"items":[`+historyJSON+`]}`)
		default:
			t.Error("unexpected history page")
		}
	}, "templates", "history", "--uid", "template-uid")
	if err != nil || calls != 2 {
		t.Fatalf("history pagination: calls=%d err=%v", calls, err)
	}
	var got struct {
		Revisions []struct {
			Name     string `json:"name"`
			Sequence uint64 `json:"sequence"`
		} `json:"revisions"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Revisions) != 2 || got.Revisions[0].Sequence != 1 || got.Revisions[1].Sequence != 2 {
		t.Fatalf("incomplete or unsorted history: %s", out)
	}
}

func TestHistoryRejectsWrongUIDAndDuplicateRevisionIdentityWithoutOutput(t *testing.T) {
	for _, tc := range []struct {
		name  string
		page  func(int) string
		calls int
	}{
		{"wrong UID despite server selector", func(int) string {
			return `{"items":[` + revisionJSON("foreign.revision", "foreign-revision", "foreign", 1) + `]}`
		}, 1},
		{"sequence above bound", func(int) string {
			return `{"items":[` + revisionJSON("template-uid.too-large", "revision-too-large", "template-uid", 1001) + `]}`
		}, 1},
		{"duplicate UID across pages", func(call int) string {
			if call == 1 {
				return `{"items":[` + historyJSON + `],"metadata":{"continue":"next"}}`
			}
			return `{"items":[` + revisionJSON("another-name", "revision-uid", "template-uid", 2) + `]}`
		}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			out, err := execute(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				token := ""
				if calls == 2 {
					token = "next"
				}
				checkHistoryQuery(t, r, "template-uid", token)
				reply(w, tc.page(calls))
			}, "templates", "history", "--uid", "template-uid")
			if err == nil || calls != tc.calls || out != "" {
				t.Fatalf("invalid history yielded output: calls=%d out=%q err=%v", calls, out, err)
			}
		})
	}
}

func TestHistoryRejectsRepeatedContinuationAndLaterPageFailureWithoutOutput(t *testing.T) {
	for _, tc := range []struct {
		name   string
		second func(http.ResponseWriter)
		want   string
	}{
		{"repeated token", func(w http.ResponseWriter) {
			reply(w, `{"items":[`+revisionJSON("template-uid.second", "revision-second", "template-uid", 2)+`],"metadata":{"continue":"same"}}`)
		}, "repeated a continuation token"},
		{"later page unavailable", func(w http.ResponseWriter) {
			w.WriteHeader(http.StatusServiceUnavailable)
			reply(w, `{"message":"temporarily unavailable"}`)
		}, "HTTP 503"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			out, err := execute(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				token := ""
				if calls > 1 {
					token = "same"
				}
				checkHistoryQuery(t, r, "template-uid", token)
				if calls == 1 {
					reply(w, `{"items":[`+historyJSON+`],"metadata":{"continue":"same"}}`)
					return
				}
				tc.second(w)
			}, "templates", "history", "--uid", "template-uid")
			if err == nil || out != "" || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("partial history escaped: calls=%d out=%q err=%v", calls, out, err)
			}
		})
	}
}

func TestExpiredHistoryContinuationRequestsFreshRead(t *testing.T) {
	calls := 0
	out, err := execute(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			reply(w, `{"items":[],"metadata":{"continue":"expired"}}`)
			return
		}
		w.WriteHeader(http.StatusGone)
		reply(w, `{"message":"expired token"}`)
	}, "templates", "history", "--uid", "template-uid")
	if err == nil || out != "" || calls != 2 || !strings.Contains(err.Error(), "Run the history command again") {
		t.Fatalf("expired pagination: calls=%d out=%q err=%v", calls, out, err)
	}
}

func TestHistoryRejectsOversizedContinuationBeforeAnotherRequest(t *testing.T) {
	calls := 0
	out, err := execute(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		reply(w, `{"items":[],"metadata":{"continue":`+quoted(strings.Repeat("x", maxContinueTokenBytes+1))+`}}`)
	}, "templates", "history", "--uid", "template-uid")
	if err == nil || calls != 1 || out != "" || !strings.Contains(err.Error(), "oversized continuation") {
		t.Fatalf("oversized token: calls=%d out=%q err=%v", calls, out, err)
	}
}

func TestHistoryStopsAtOneThousandPagesWithoutPartialOutput(t *testing.T) {
	calls := 0
	out, err := execute(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		item := revisionJSON(fmt.Sprintf("template-uid.revision-%d", calls),
			fmt.Sprintf("revision-%d", calls), "template-uid", uint64(calls))
		reply(w, fmt.Sprintf(`{"items":[%s],"metadata":{"continue":"page-%d"}}`, item, calls))
	}, "templates", "history", "--uid", "template-uid")
	if err == nil || calls != maxTemplateRevisions || out != "" ||
		!strings.Contains(err.Error(), "exceeds the supported 1000 revisions") {
		t.Fatalf("history bound: calls=%d out=%q err=%v", calls, out, err)
	}
}

func TestHistoryBufferDropsDefinitionsAndUnneededMetadata(t *testing.T) {
	largeDefinition := `{"description":"` + strings.Repeat("d", 1<<20) + `"}`
	largeAnnotation := strings.Repeat("a", 1<<20)
	response := fmt.Sprintf(`{"items":[{"metadata":{"name":"template-uid.revision-uid","uid":"revision-uid","annotations":{"large":%q},"labels":{"extra":"discard"},"managedFields":[{"manager":"discard"}]},"spec":{"templateUID":"template-uid","sequence":1,"digest":%q,"effectiveActorUID":"actor-uid","acceptedAt":"2026-10-20T00:00:00Z","definitionJSON":%q}}]}`,
		largeAnnotation, digest(largeDefinition), largeDefinition)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkRequest(t, r, http.MethodGet, "apptemplaterevisions")
		checkHistoryQuery(t, r, "template-uid", "")
		reply(w, response)
	}))
	defer server.Close()
	client, err := newClient(Options{RESTConfig: func() (*rest.Config, error) {
		return &rest.Config{Host: server.URL + "/projects/project-a/control-plane",
			BearerToken: "fixture-token"}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := readRevisionHistory(context.Background(), client, "template-uid")
	want := []historyEntry{{Name: "template-uid.revision-uid", Sequence: 1,
		Digest: digest(largeDefinition), ActorUID: "actor-uid", AcceptedAt: "2026-10-20T00:00:00Z"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("history buffer retained or changed non-output fields: %#v err=%v", got, err)
	}
}
func TestRestoreUsesCurrentPreconditionsAndPreservesExactJSONNumber(t *testing.T) {
	for _, preview := range []bool{false, true} {
		t.Run(fmt.Sprint(preview), func(t *testing.T) {
			calls := 0
			args := []string{"templates", "restore", "web-api", "--revision", "template-uid.revision-uid"}
			if preview {
				args = append(args, "--dry-run=server")
			}
			out, err := execute(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				switch calls {
				case 1:
					reply(w, currentJSON)
				case 2:
					checkRequest(t, r, http.MethodGet, "apptemplaterevisions/template-uid.revision-uid")
					reply(w, historyJSON)
				case 3:
					checkRequest(t, r, http.MethodPut, "apptemplates/web-api")
					if (r.URL.Query().Get("dryRun") == "All") != preview {
						t.Error("wrong restore dry-run")
					}
					var got template
					if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
						t.Fatal(err)
					}
					if got.Metadata.UID != "template-uid" || got.Metadata.ResourceVersion != "opaque-rv-a" {
						t.Error("lost restore UID/version")
					}
					if got.Spec.DefinitionJSON != historyDefinition || got.Spec.Digest != digest(historyDefinition) {
						t.Errorf("restore changed exact definition envelope: %#v", got.Spec)
					}
					reply(w, currentJSON)
				default:
					t.Error("unexpected request")
				}
			}, args...)
			if err != nil || calls != 3 || strings.Contains(out, "9007199254740993") {
				t.Fatalf("restore: %v calls=%d out=%s", err, calls, out)
			}
		})
	}
}
func TestRestoreRefusesOldTemplateUIDAfterNameReuse(t *testing.T) {
	calls := 0
	_, err := execute(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			reply(w, strings.ReplaceAll(currentJSON, "template-uid", "replacement-uid"))
		} else if calls == 2 {
			reply(w, historyJSON)
		} else {
			t.Error("wrote after UID mismatch")
		}
	}, "templates", "restore", "web-api", "--revision", "template-uid.revision-uid")
	if err == nil || calls != 2 {
		t.Fatalf("UID guard: %v calls=%d", err, calls)
	}
}
func TestRestoreConflictNeverRetries(t *testing.T) {
	calls := 0
	_, err := execute(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch calls {
		case 1:
			reply(w, currentJSON)
		case 2:
			reply(w, historyJSON)
		case 3:
			w.WriteHeader(409)
			reply(w, `{"message":"server text MUST_NOT_PRINT"}`)
		default:
			t.Error("retried conflict")
		}
	}, "templates", "restore", "web-api", "--revision", "template-uid.revision-uid")
	if err == nil || calls != 3 || strings.Contains(err.Error(), "MUST_NOT_PRINT") {
		t.Fatalf("conflict: %v calls=%d", err, calls)
	}
}
func TestDiffMakesOnlyReads(t *testing.T) {
	calls := 0
	out, err := execute(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" {
			t.Error("diff wrote")
		}
		if calls == 1 {
			reply(w, currentJSON)
		} else {
			reply(w, historyJSON)
		}
	}, "templates", "diff", "web-api", "--revision", "template-uid.revision-uid")
	if err != nil || calls != 2 || !strings.Contains(out, "9007199254740993") || !strings.Contains(out, "18446744073709551615") || !strings.Contains(out, "123456789012345678901e-21") || !strings.Contains(out, "9007199254740993e1") || !strings.Contains(out, "--- current") {
		t.Fatalf("diff: %v %s", err, out)
	}
}
func TestDeleteUsesBodyDryRunAndExactPreconditions(t *testing.T) {
	for _, preview := range []bool{false, true} {
		t.Run(fmt.Sprint(preview), func(t *testing.T) {
			calls := 0
			args := []string{"templates", "delete", "web-api", "--uid", "template-uid", "--resource-version", "opaque-rv-a"}
			if preview {
				args = append(args, "--dry-run=server")
			}
			out, err := execute(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				checkRequest(t, r, "DELETE", "apptemplates/web-api")
				if r.URL.RawQuery != "" {
					t.Fatal("DELETE options put in query")
				}
				var fields map[string]json.RawMessage
				if e := json.NewDecoder(r.Body).Decode(&fields); e != nil {
					t.Fatal(e)
				}
				if string(fields["preconditions"]) != `{"uid":"template-uid","resourceVersion":"opaque-rv-a"}` {
					t.Errorf("wrong preconditions %s", fields["preconditions"])
				}
				if preview && string(fields["dryRun"]) != `["All"]` {
					t.Error("dry-run body absent")
				}
				if !preview && fields["dryRun"] != nil {
					t.Error("unexpected dry-run field")
				}
				reply(w, currentJSON)
			}, args...)
			if err != nil || calls != 1 || strings.Contains(out, "inputSchema") {
				t.Fatalf("delete: %v calls=%d out=%s", err, calls, out)
			}
		})
	}
}
func TestBadSelectionsMakeNoRequests(t *testing.T) {
	cases := [][]string{
		{"templates", "history"}, {"templates", "history", "web-api", "--uid", "uid"},
		{"templates", "history", "--uid", "../escape"},
		{"templates", "history", "--uid", "template-uid,other=uid"},
		{"templates", "delete", "web-api", "--uid", "uid"},
		{"templates", "delete", "web-api", "--uid", "uid", "--resource-version", ""},
		{"templates", "delete", "../escape", "--uid", "uid", "--resource-version", "42"},
		{"templates", "delete", "web-api", "--uid", "uid", "--resource-version", "42", "--dry-run=client"},
		{"templates", "restore", "web-api"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, err := execute(t, func(http.ResponseWriter, *http.Request) { t.Error("invalid command made network request") }, args...)
			if err == nil {
				t.Fatal("accepted invalid selection")
			}
		})
	}
}

func TestReadRejectsLegacyOrInvalidDefinitionEnvelope(t *testing.T) {
	badTemplateSpecs := []struct {
		name, spec string
	}{
		{"legacy definition object", `{"description":"private marker"}`},
		{"missing digest", `{"definitionJSON":"{}"}`},
		{"malformed embedded JSON", `{"definitionJSON":"{","digest":"sha256:bad"}`},
		{"wrong digest", `{"definitionJSON":"{\"x\":1}","digest":"sha256:0000000000000000000000000000000000000000000000000000000000000000"}`},
	}
	for _, tc := range badTemplateSpecs {
		t.Run("template/"+tc.name, func(t *testing.T) {
			calls := 0
			bad := `{"apiVersion":"apps.miloapis.com/v1alpha1","kind":"AppTemplate","metadata":{"name":"web-api","uid":"template-uid","resourceVersion":"opaque-rv-a"},"spec":` + tc.spec + `}`
			_, err := execute(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				reply(w, bad)
			}, "templates", "diff", "web-api", "--revision", "template-uid.revision-uid")
			if err == nil || calls != 1 || strings.Contains(err.Error(), "private marker") {
				t.Fatalf("accepted invalid template envelope or leaked content: calls=%d err=%v", calls, err)
			}
		})
	}
	badRevisions := []struct {
		name, spec string
	}{
		{"legacy definition object", `{"templateUID":"template-uid","definition":{"private":"marker"}}`},
		{"missing digest", `{"templateUID":"template-uid","definitionJSON":"{}"}`},
		{"wrong digest", `{"templateUID":"template-uid","definitionJSON":"{}","digest":"sha256:0000000000000000000000000000000000000000000000000000000000000000"}`},
	}
	for _, tc := range badRevisions {
		t.Run("revision/"+tc.name, func(t *testing.T) {
			calls := 0
			_, err := execute(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls == 1 {
					reply(w, currentJSON)
					return
				}
				reply(w, `{"metadata":{"name":"template-uid.revision-uid"},"spec":`+tc.spec+`}`)
			}, "templates", "diff", "web-api", "--revision", "template-uid.revision-uid")
			if err == nil || calls != 2 || strings.Contains(err.Error(), "marker") {
				t.Fatalf("accepted invalid revision envelope or leaked content: calls=%d err=%v", calls, err)
			}
		})
	}
}
func TestRedirectDoesNotForwardCredentials(t *testing.T) {
	got := 0
	dest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { got++ }))
	defer dest.Close()
	_, err := execute(t, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, dest.URL, 307) }, "templates", "history", "web-api")
	if err == nil || got != 0 {
		t.Fatalf("redirect followed: %v calls=%d", err, got)
	}
}
func TestDenialDoesNotExposeServerResponse(t *testing.T) {
	_, err := execute(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		reply(w, `{"message":"secret-input-value"}`)
	}, "templates", "history", "web-api")
	if err == nil || strings.Contains(err.Error(), "secret-input-value") {
		t.Fatalf("error: %v", err)
	}
}

func TestAmbiguousRestoreResponseRequiresInspectionWithoutRetry(t *testing.T) {
	for _, failure := range []struct {
		name   string
		status int
		body   string
	}{
		{"post-commit storage read failure", 503, `{"message":"storage unavailable"}`},
		{"truncated response", 200, `{"metadata":`},
		{"missing identity", 200, `{}`},
	} {
		t.Run(failure.name, func(t *testing.T) {
			calls, writes := 0, 0
			_, err := execute(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				switch calls {
				case 1:
					reply(w, currentJSON)
				case 2:
					reply(w, historyJSON)
				case 3:
					writes++
					w.WriteHeader(failure.status)
					reply(w, failure.body)
				default:
					t.Error("retried ambiguous write")
				}
			}, "templates", "restore", "web-api", "--revision", "template-uid.revision-uid")
			if err == nil || !strings.Contains(err.Error(), "outcome is unknown") || !strings.Contains(err.Error(), "before retrying") || writes != 1 || calls != 3 {
				t.Fatalf("ambiguous result: calls=%d writes=%d err=%v", calls, writes, err)
			}
		})
	}
}
