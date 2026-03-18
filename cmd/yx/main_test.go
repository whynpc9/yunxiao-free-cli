package main

import (
	"testing"

	"github.com/wanghongyi/yunxiao-free-cli/internal/config"
)

func TestParseConditions(t *testing.T) {
	conds, err := parseConditions(`{"name":"登录","priority":"P1"}`)
	if err != nil {
		t.Fatalf("parseConditions error: %v", err)
	}
	if conds["name"] != "登录" {
		t.Fatalf("unexpected name: %v", conds["name"])
	}
	if conds["priority"] != "P1" {
		t.Fatalf("unexpected priority: %v", conds["priority"])
	}
}

func TestParseConditionsInvalid(t *testing.T) {
	if _, err := parseConditions(`[]`); err == nil {
		t.Fatal("expected error for non-object json")
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
