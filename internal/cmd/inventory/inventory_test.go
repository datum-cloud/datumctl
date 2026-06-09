package inventory

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func obj(name string, labels map[string]string, spec, status map[string]interface{}) unstructured.Unstructured {
	o := map[string]interface{}{"metadata": map[string]interface{}{"name": name}}
	if labels != nil {
		l := map[string]interface{}{}
		for k, v := range labels {
			l[k] = v
		}
		o["metadata"].(map[string]interface{})["labels"] = l
	}
	if spec != nil {
		o["spec"] = spec
	}
	if status != nil {
		o["status"] = status
	}
	return unstructured.Unstructured{Object: o}
}

func readyCond(s string) map[string]interface{} {
	return map[string]interface{}{
		"conditions": []interface{}{map[string]interface{}{"type": "Ready", "status": s}},
	}
}

func TestStrAndIntStr(t *testing.T) {
	u := obj("n", nil, map[string]interface{}{
		"hardware": map[string]interface{}{"cpuArchitecture": "arm64", "cpuCores": int64(96)},
	}, nil)
	if got := str(u, "spec", "hardware", "cpuArchitecture"); got != "arm64" {
		t.Errorf("str arch = %q, want arm64", got)
	}
	if got := intStr(u, "spec", "hardware", "cpuCores"); got != "96" {
		t.Errorf("intStr cpu = %q, want 96", got)
	}
	if got := str(u, "spec", "missing"); got != none {
		t.Errorf("str missing = %q, want %s", got, none)
	}
	if got := intStr(u, "spec", "missing"); got != none {
		t.Errorf("intStr missing = %q, want %s", got, none)
	}
}

func TestReady(t *testing.T) {
	cases := map[string]struct {
		status map[string]interface{}
		want   string
	}{
		"true":    {readyCond("True"), "True"},
		"false":   {readyCond("False"), "False"},
		"none":    {nil, none},
		"noReady": {map[string]interface{}{"conditions": []interface{}{map[string]interface{}{"type": "Accepted", "status": "True"}}}, none},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := ready(obj("x", nil, nil, tc.status)); got != tc.want {
				t.Errorf("ready = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSitesViewRow(t *testing.T) {
	u := obj("us-central-2a", map[string]string{labelRegion: "us-central-2"}, map[string]interface{}{
		"regionRef":   map[string]interface{}{"name": "us-central-2"},
		"providerRef": map[string]interface{}{"name": "netactuate"},
		"type":        "Edge",
	}, readyCond("True"))
	got := sitesView.row(u)
	want := []any{"us-central-2a", "us-central-2", "netactuate", "Edge", "True"}
	if len(got) != len(want) {
		t.Fatalf("row len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("col %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestFilterItemsPredicate(t *testing.T) {
	list := &unstructured.UnstructuredList{Items: []unstructured.Unstructured{
		obj("a", nil, map[string]interface{}{"providerRef": map[string]interface{}{"name": "vultr"}}, nil),
		obj("b", nil, map[string]interface{}{"providerRef": map[string]interface{}{"name": "netactuate"}}, nil),
	}}
	filterItems(list, []func(u unstructured.Unstructured) bool{
		func(u unstructured.Unstructured) bool { return str(u, "spec", "providerRef", "name") == "netactuate" },
	})
	if len(list.Items) != 1 || list.Items[0].GetName() != "b" {
		t.Fatalf("filterItems kept %v, want [b]", names(list))
	}
}

func TestFilterItemsNoPredicateKeepsAll(t *testing.T) {
	list := &unstructured.UnstructuredList{Items: []unstructured.Unstructured{obj("a", nil, nil, nil), obj("b", nil, nil, nil)}}
	filterItems(list, nil)
	if len(list.Items) != 2 {
		t.Fatalf("filterItems dropped items without predicate: %v", names(list))
	}
}

func TestPrintTree(t *testing.T) {
	regions := &unstructured.UnstructuredList{Items: []unstructured.Unstructured{
		obj("us-central-2", nil, nil, nil),
		obj("eu-west-1", nil, nil, nil),
	}}
	sites := &unstructured.UnstructuredList{Items: []unstructured.Unstructured{
		obj("us-central-2a", nil, map[string]interface{}{"regionRef": map[string]interface{}{"name": "us-central-2"}}, nil),
	}}
	clusters := &unstructured.UnstructuredList{Items: []unstructured.Unstructured{
		obj("edge-1", map[string]string{labelRegion: "us-central-2"}, nil, nil),
	}}
	nodes := &unstructured.UnstructuredList{Items: []unstructured.Unstructured{
		obj("node-1", nil, map[string]interface{}{"siteRef": map[string]interface{}{"name": "us-central-2a"}}, nil),
	}}

	var buf bytes.Buffer
	printTree(&buf, "", regions, sites, clusters, nodes)
	out := buf.String()
	for _, want := range []string{"us-central-2", "eu-west-1", "clusters: edge-1", "  us-central-2a", "    node-1"} {
		if !strings.Contains(out, want) {
			t.Errorf("tree output missing %q\n%s", want, out)
		}
	}

	buf.Reset()
	printTree(&buf, "us-central-2", regions, sites, clusters, nodes)
	if strings.Contains(buf.String(), "eu-west-1") {
		t.Errorf("--region filter leaked other region:\n%s", buf.String())
	}
}

func TestTallyAndUnion(t *testing.T) {
	items := []unstructured.Unstructured{
		obj("a", nil, map[string]interface{}{"regionRef": map[string]interface{}{"name": "r1"}}, nil),
		obj("b", nil, map[string]interface{}{"regionRef": map[string]interface{}{"name": "r1"}}, nil),
		obj("c", nil, map[string]interface{}{"regionRef": map[string]interface{}{"name": "r2"}}, nil),
	}
	got := tally(items, func(u unstructured.Unstructured) string { return str(u, "spec", "regionRef", "name") })
	if got["r1"] != 2 || got["r2"] != 1 {
		t.Errorf("tally = %v, want r1:2 r2:1", got)
	}
	u := union(map[string]int{"r1": 1}, map[string]int{"r2": 1, "r1": 3})
	if strings.Join(u, ",") != "r1,r2" {
		t.Errorf("union = %v, want [r1 r2] sorted", u)
	}
}

func TestRenderJSONYAMLUnstructured(t *testing.T) {
	list := &unstructured.UnstructuredList{}
	list.SetAPIVersion("inventory.miloapis.com/v1alpha1")
	list.SetKind("SiteList")
	list.Items = []unstructured.Unstructured{obj("s1", nil, map[string]interface{}{"type": "Edge"}, nil)}
	for _, f := range []string{"json", "yaml"} {
		var buf bytes.Buffer
		c := &cobra.Command{}
		c.SetOut(&buf)
		if err := render(c, f, list, sitesView.headers, sitesView.row); err != nil {
			t.Fatalf("%s render err: %v", f, err)
		}
		if !strings.Contains(buf.String(), "s1") {
			t.Errorf("%s output missing name s1:\n%s", f, buf.String())
		}
	}
}

func TestRenderInvalidFormat(t *testing.T) {
	c := &cobra.Command{}
	c.SetOut(&bytes.Buffer{})
	if err := render(c, "xml", &unstructured.UnstructuredList{}, nil, nil); err == nil {
		t.Fatal("render with invalid format should error")
	}
}

func names(list *unstructured.UnstructuredList) []string {
	var out []string
	for _, i := range list.Items {
		out = append(out, i.GetName())
	}
	return out
}
