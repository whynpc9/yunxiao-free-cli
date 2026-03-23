package main

import (
	"reflect"
	"testing"

	"github.com/wanghongyi/yunxiao-free-cli/internal/config"
)

func TestValidateJSONObjectString(t *testing.T) {
	value, err := validateJSONObjectString(`{"name":"登录","priority":"P1"}`, "--conditions")
	if err != nil {
		t.Fatalf("validateJSONObjectString error: %v", err)
	}
	if value != `{"name":"登录","priority":"P1"}` {
		t.Fatalf("unexpected value: %s", value)
	}
}

func TestValidateJSONObjectStringInvalid(t *testing.T) {
	if _, err := validateJSONObjectString(`[]`, "--conditions"); err == nil {
		t.Fatal("expected error for non-object json")
	}
}

func TestSplitCSV(t *testing.T) {
	got := splitCSV("Req, Bug ,Task,,")
	want := []string{"Req", "Bug", "Task"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected split result: %#v", got)
	}
}

func TestSplitCategoriesDefault(t *testing.T) {
	got := splitCategories("")
	want := []string{"Req", "Bug", "Task"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected categories: %#v", got)
	}
}

func TestPlainWorkitemDescription(t *testing.T) {
	raw := `{"htmlValue":"<h1>需求</h1><p>第一段</p><ul><li>项目A</li><li>项目B</li></ul>"}`
	got := plainWorkitemDescription(raw)
	want := "需求 第一段 项目A 项目B"
	if got != want {
		t.Fatalf("unexpected description: %q", got)
	}
}

func TestPickOrgID(t *testing.T) {
	cfg := config.Config{DefaultOrganizationID: "org-default"}
	if got := pickOrgID(cfg, ""); got != "org-default" {
		t.Fatalf("expected default org, got %s", got)
	}
	if got := pickOrgID(cfg, "org-explicit"); got != "org-explicit" {
		t.Fatalf("expected explicit org, got %s", got)
	}
}
