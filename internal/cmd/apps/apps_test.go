package apps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"k8s.io/client-go/rest"
)

const currentJSON = `{"apiVersion":"apps.miloapis.com/v1alpha1","kind":"AppTemplate","metadata":{"name":"web-api","uid":"template-uid","resourceVersion":"42","generation":2},"spec":{"description":"current","inputSchema":{"type":"object","additionalProperties":false,"properties":{}}}}`
const historyJSON = `{"metadata":{"name":"template-uid.revision-uid"},"spec":{"templateUID":"template-uid","sequence":1,"digest":"sha256:abc","effectiveActorUID":"actor-uid","acceptedAt":"2026-10-20T00:00:00Z","definition":{"description":"old","large":9007199254740993}}}`

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
			reply(w, `{"items":[`+historyJSON+`,{"metadata":{"name":"different-uid.rev"},"spec":{"templateUID":"different-uid","definition":{"sensitive":"MUST_NOT_PRINT"}}}]}`)
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
		reply(w, `{"items":[]}`)
	}, "templates", "history", "--uid", "old-uid")
	if err != nil || calls != 1 {
		t.Fatalf("history: %v calls=%d", err, calls)
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
					if got.Metadata.UID != "template-uid" || got.Metadata.ResourceVersion != "42" {
						t.Error("lost restore UID/version")
					}
					if !strings.Contains(string(got.Spec), "9007199254740993") {
						t.Errorf("number lost precision: %s", got.Spec)
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
	if err != nil || calls != 2 || !strings.Contains(out, "9007199254740993") || !strings.Contains(out, "--- current") {
		t.Fatalf("diff: %v %s", err, out)
	}
}
func TestDeleteUsesBodyDryRunAndExactPreconditions(t *testing.T) {
	for _, preview := range []bool{false, true} {
		t.Run(fmt.Sprint(preview), func(t *testing.T) {
			calls := 0
			args := []string{"templates", "delete", "web-api", "--uid", "template-uid", "--resource-version", "42"}
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
				if string(fields["preconditions"]) != `{"uid":"template-uid","resourceVersion":"42"}` {
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
		{"templates", "delete", "web-api", "--uid", "uid"},
		{"templates", "delete", "web-api", "--uid", "uid", "--resource-version", "+42"},
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
