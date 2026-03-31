package manifest

import (
	"fmt"
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

// ── Adversarial / edge-case tests ────────────────────────────────────────────

func TestDiff_NilServicesMap(t *testing.T) {
	// A manifest with a nil services map should not panic.
	a := &Manifest{Meta: Meta{Version: 1}, Services: nil}
	b := &Manifest{Meta: Meta{Version: 1}, Services: map[string]Service{
		"svc": {Port: 8080},
	}}
	d := Diff(a, b)
	if len(d.Added) != 1 || d.Added[0] != "svc" {
		t.Errorf("expected svc added, got: %v", d.Added)
	}
	if len(d.Removed) != 0 {
		t.Errorf("expected nothing removed, got: %v", d.Removed)
	}

	// Also reverse: b is nil.
	d2 := Diff(b, a)
	if len(d2.Removed) != 1 || d2.Removed[0] != "svc" {
		t.Errorf("expected svc removed, got: %v", d2.Removed)
	}
}

func TestDiff_BothNilServicesMaps(t *testing.T) {
	a := &Manifest{Meta: Meta{Version: 1}, Services: nil}
	b := &Manifest{Meta: Meta{Version: 1}, Services: nil}
	d := Diff(a, b)
	if !d.Empty() {
		t.Error("both nil services maps should produce empty diff")
	}
}

func TestDiff_AllFieldsChanged(t *testing.T) {
	a := &Manifest{Services: map[string]Service{
		"svc": {
			Description: "old desc",
			Host:        "host-a",
			Port:        3001,
			HealthURL:   "http://old/health",
			SystemdUnit: "old.service",
			Repo:        "old/repo",
			Version:     "1.0.0",
			MaxMajor:    1,
			Docs:        "https://old.docs",
			Tags:        []string{"a", "b"},
			Added:       "2024-01-01",
		},
	}}
	b := &Manifest{Services: map[string]Service{
		"svc": {
			Description: "new desc",
			Host:        "host-b",
			Port:        3002,
			HealthURL:   "http://new/health",
			SystemdUnit: "new.service",
			Repo:        "new/repo",
			Version:     "2.0.0",
			MaxMajor:    2,
			Docs:        "https://new.docs",
			Tags:        []string{"c"},
			Added:       "2025-01-01",
		},
	}}
	d := Diff(a, b)
	if len(d.Changed) != 1 {
		t.Fatalf("expected 1 changed service, got %d", len(d.Changed))
	}
	if len(d.Changed[0].Changes) != 11 {
		t.Errorf("expected 11 field changes (all fields), got %d: %v",
			len(d.Changed[0].Changes), d.Changed[0].Changes)
	}
}

func TestDiff_TagOrderInsensitive(t *testing.T) {
	// Tags [b, a] and [a, b] should be identical — sort before comparing.
	a := &Manifest{Services: map[string]Service{
		"svc": {Port: 80, Tags: []string{"b", "a", "z"}},
	}}
	b := &Manifest{Services: map[string]Service{
		"svc": {Port: 80, Tags: []string{"z", "a", "b"}},
	}}
	d := Diff(a, b)
	if !d.Empty() {
		t.Errorf("tag order should not produce diff, got: %v", d.Changed)
	}
}

func TestDiff_TagsNilVsEmpty(t *testing.T) {
	// nil tags and empty slice should be identical.
	a := &Manifest{Services: map[string]Service{
		"svc": {Port: 80, Tags: nil},
	}}
	b := &Manifest{Services: map[string]Service{
		"svc": {Port: 80, Tags: []string{}},
	}}
	d := Diff(a, b)
	if !d.Empty() {
		t.Errorf("nil vs empty tags should not produce diff, got: %v", d.Changed)
	}
}

func TestDiff_MetadataChangesIgnored(t *testing.T) {
	// manifest.host and manifest.ignore_units changes should NOT appear in diff output.
	a := &Manifest{
		Meta:     Meta{Version: 1, Host: "host-a", IgnoreUnits: []string{"foo.service"}},
		Services: map[string]Service{"svc": {Port: 80}},
	}
	b := &Manifest{
		Meta:     Meta{Version: 1, Host: "host-b", IgnoreUnits: nil},
		Services: map[string]Service{"svc": {Port: 80}},
	}
	d := Diff(a, b)
	if !d.Empty() {
		t.Errorf("manifest metadata changes should be ignored, got: %v", d)
	}
}

func TestDiff_MassiveManifest(t *testing.T) {
	// 1000 services — every 7th has a port change, 10 new added.
	aServices := make(map[string]Service, 1000)
	bServices := make(map[string]Service, 1010)
	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("svc-%04d", i)
		aServices[id] = Service{Description: fmt.Sprintf("Service %d", i), Port: 3000 + i}
		port := 3000 + i
		if i%7 == 0 {
			port = 4000 + i
		}
		bServices[id] = Service{Description: fmt.Sprintf("Service %d", i), Port: port}
	}
	for i := 1000; i < 1010; i++ {
		bServices[fmt.Sprintf("new-%04d", i)] = Service{
			Description: fmt.Sprintf("New %d", i), Port: 5000 + i,
		}
	}

	a := &Manifest{Services: aServices}
	b := &Manifest{Services: bServices}
	d := Diff(a, b)

	expectedAdded := 10
	expectedChanged := 143 // every 7th of 0–999 inclusive: 0,7,14,...,994 = 143 values
	if len(d.Added) != expectedAdded {
		t.Errorf("expected %d added, got %d", expectedAdded, len(d.Added))
	}
	if len(d.Removed) != 0 {
		t.Errorf("expected 0 removed, got %d", len(d.Removed))
	}
	if len(d.Changed) != expectedChanged {
		t.Errorf("expected %d changed, got %d", expectedChanged, len(d.Changed))
	}
	if !d.Empty() == d.Empty() {
		t.Error("Empty() logic is broken")
	}
	if d.Empty() {
		t.Error("massive diff should not be Empty()")
	}
}

func TestDiff_PortZeroToNonZero(t *testing.T) {
	// Port 0 (absent) → non-zero should show as port change.
	a := &Manifest{Services: map[string]Service{
		"svc": {Description: "test", HealthURL: "https://example.com/"},
	}}
	b := &Manifest{Services: map[string]Service{
		"svc": {Description: "test", HealthURL: "https://example.com/", Port: 8080},
	}}
	d := Diff(a, b)
	if len(d.Changed) != 1 {
		t.Fatalf("expected 1 changed, got %d", len(d.Changed))
	}
	found := false
	for _, fc := range d.Changed[0].Changes {
		if fc.Field == "port" && fc.From == "" && fc.To == "8080" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected port 0→8080 change, got: %v", d.Changed[0].Changes)
	}
}

func TestDiff_MaxMajorZeroToNonZero(t *testing.T) {
	// max_major 0 (absent) → non-zero should show as change.
	a := &Manifest{Services: map[string]Service{
		"svc": {Description: "test", Port: 80},
	}}
	b := &Manifest{Services: map[string]Service{
		"svc": {Description: "test", Port: 80, MaxMajor: 2},
	}}
	d := Diff(a, b)
	if len(d.Changed) != 1 {
		t.Fatalf("expected 1 changed, got %d", len(d.Changed))
	}
	found := false
	for _, fc := range d.Changed[0].Changes {
		if fc.Field == "max_major" && fc.From == "" && fc.To == "2" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected max_major change, got: %v", d.Changed[0].Changes)
	}
}
