package manifest

import (
	"testing"
)

func TestDiff_Empty(t *testing.T) {
	a := &Manifest{
		Meta:     Meta{Version: 1, Host: "localhost"},
		Services: map[string]Service{},
	}
	b := &Manifest{
		Meta:     Meta{Version: 1, Host: "localhost"},
		Services: map[string]Service{},
	}
	d := Diff(a, b)
	if !d.Empty() {
		t.Error("expected empty diff for identical empty manifests")
	}
}

func TestDiff_Identical(t *testing.T) {
	svc := Service{
		Description: "My service",
		Port:        8080,
		HealthURL:   "http://localhost:8080/health",
		SystemdUnit: "my-service.service",
	}
	a := &Manifest{Services: map[string]Service{"my-service": svc}}
	b := &Manifest{Services: map[string]Service{"my-service": svc}}
	d := Diff(a, b)
	if !d.Empty() {
		t.Errorf("expected empty diff for identical manifests, got: added=%v removed=%v changed=%v",
			d.Added, d.Removed, d.Changed)
	}
}

func TestDiff_Added(t *testing.T) {
	a := &Manifest{Services: map[string]Service{
		"alpha": {Port: 3001},
	}}
	b := &Manifest{Services: map[string]Service{
		"alpha": {Port: 3001},
		"beta":  {Port: 3002},
		"gamma": {Port: 3003},
	}}
	d := Diff(a, b)
	if len(d.Added) != 2 {
		t.Errorf("expected 2 added, got %d: %v", len(d.Added), d.Added)
	}
	if d.Added[0] != "beta" || d.Added[1] != "gamma" {
		t.Errorf("unexpected added IDs: %v", d.Added)
	}
	if len(d.Removed) != 0 {
		t.Errorf("expected 0 removed, got %d", len(d.Removed))
	}
}

func TestDiff_Removed(t *testing.T) {
	a := &Manifest{Services: map[string]Service{
		"alpha": {Port: 3001},
		"beta":  {Port: 3002},
	}}
	b := &Manifest{Services: map[string]Service{
		"alpha": {Port: 3001},
	}}
	d := Diff(a, b)
	if len(d.Removed) != 1 {
		t.Errorf("expected 1 removed, got %d: %v", len(d.Removed), d.Removed)
	}
	if d.Removed[0] != "beta" {
		t.Errorf("expected removed 'beta', got %q", d.Removed[0])
	}
}

func TestDiff_Changed_Port(t *testing.T) {
	a := &Manifest{Services: map[string]Service{
		"my-svc": {Description: "test", Port: 3001},
	}}
	b := &Manifest{Services: map[string]Service{
		"my-svc": {Description: "test", Port: 3002},
	}}
	d := Diff(a, b)
	if len(d.Changed) != 1 {
		t.Fatalf("expected 1 changed service, got %d", len(d.Changed))
	}
	sc := d.Changed[0]
	if sc.ID != "my-svc" {
		t.Errorf("expected changed ID 'my-svc', got %q", sc.ID)
	}
	if len(sc.Changes) != 1 {
		t.Fatalf("expected 1 field change, got %d: %v", len(sc.Changes), sc.Changes)
	}
	fc := sc.Changes[0]
	if fc.Field != "port" {
		t.Errorf("expected field 'port', got %q", fc.Field)
	}
	if fc.From != "3001" || fc.To != "3002" {
		t.Errorf("expected port 3001→3002, got %s→%s", fc.From, fc.To)
	}
}

func TestDiff_Changed_MultipleFields(t *testing.T) {
	a := &Manifest{Services: map[string]Service{
		"my-svc": {Description: "old", Port: 3001, Version: "1.0.0"},
	}}
	b := &Manifest{Services: map[string]Service{
		"my-svc": {Description: "new", Port: 3001, Version: "1.1.0"},
	}}
	d := Diff(a, b)
	if len(d.Changed) != 1 {
		t.Fatalf("expected 1 changed service, got %d", len(d.Changed))
	}
	if len(d.Changed[0].Changes) != 2 {
		t.Errorf("expected 2 field changes, got %d: %v", len(d.Changed[0].Changes), d.Changed[0].Changes)
	}
}

func TestDiff_HealthURLAdded(t *testing.T) {
	a := &Manifest{Services: map[string]Service{
		"dead-drop": {Port: 3001},
	}}
	b := &Manifest{Services: map[string]Service{
		"dead-drop": {Port: 3001, HealthURL: "https://example.com/drop/health"},
	}}
	d := Diff(a, b)
	if len(d.Changed) != 1 {
		t.Fatalf("expected 1 changed, got %d", len(d.Changed))
	}
	fc := d.Changed[0].Changes[0]
	if fc.Field != "health_url" {
		t.Errorf("expected health_url change, got %q", fc.Field)
	}
	if fc.From != "" || fc.To != "https://example.com/drop/health" {
		t.Errorf("unexpected health_url change: %q → %q", fc.From, fc.To)
	}
}

func TestDiff_Tags(t *testing.T) {
	a := &Manifest{Services: map[string]Service{
		"svc": {Port: 80, Tags: []string{"http", "public"}},
	}}
	b := &Manifest{Services: map[string]Service{
		"svc": {Port: 80, Tags: []string{"http", "internal"}},
	}}
	d := Diff(a, b)
	if len(d.Changed) != 1 {
		t.Fatalf("expected 1 changed, got %d", len(d.Changed))
	}
	fc := d.Changed[0].Changes[0]
	if fc.Field != "tags" {
		t.Errorf("expected tags change, got %q", fc.Field)
	}
}

func TestDiff_Empty_Result(t *testing.T) {
	d := &DiffResult{}
	if !d.Empty() {
		t.Error("new empty DiffResult should be Empty()")
	}
	d.Added = []string{"x"}
	if d.Empty() {
		t.Error("DiffResult with Added should not be Empty()")
	}
}

func TestDiff_SortedOutput(t *testing.T) {
	a := &Manifest{Services: map[string]Service{
		"delta": {Port: 1},
		"alpha": {Port: 2},
		"gamma": {Port: 3},
	}}
	b := &Manifest{Services: map[string]Service{
		"zeta":  {Port: 9},
		"beta":  {Port: 8},
		"gamma": {Port: 3},
	}}
	d := Diff(a, b)
	// Added: beta, zeta (sorted)
	if len(d.Added) != 2 || d.Added[0] != "beta" || d.Added[1] != "zeta" {
		t.Errorf("unexpected added order: %v", d.Added)
	}
	// Removed: alpha, delta (sorted)
	if len(d.Removed) != 2 || d.Removed[0] != "alpha" || d.Removed[1] != "delta" {
		t.Errorf("unexpected removed order: %v", d.Removed)
	}
}
