package manifest

import (
	"fmt"
	"sort"
	"strings"
)

// DiffResult holds the result of comparing two manifests.
type DiffResult struct {
	Added   []string        // service IDs present in b but not a
	Removed []string        // service IDs present in a but not b
	Changed []ServiceChange // services present in both with field differences
}

// ServiceChange describes what changed for a specific service.
type ServiceChange struct {
	ID      string
	Changes []FieldChange
}

// FieldChange describes a single field that changed between two service definitions.
type FieldChange struct {
	Field string
	From  string
	To    string
}

// Empty returns true if there are no differences between the two manifests.
func (d *DiffResult) Empty() bool {
	return len(d.Added) == 0 && len(d.Removed) == 0 && len(d.Changed) == 0
}

// Diff compares two manifests and returns the differences in service definitions.
// Only service-level changes are reported; manifest metadata (host, ignore_units) is ignored.
func Diff(a, b *Manifest) *DiffResult {
	result := &DiffResult{}

	// Services in a but not b → removed.
	for id := range a.Services {
		if _, ok := b.Services[id]; !ok {
			result.Removed = append(result.Removed, id)
		}
	}
	sort.Strings(result.Removed)

	// Services in b but not a → added.
	for id := range b.Services {
		if _, ok := a.Services[id]; !ok {
			result.Added = append(result.Added, id)
		}
	}
	sort.Strings(result.Added)

	// Services in both → compare fields.
	var changedIDs []string
	for id := range a.Services {
		if _, ok := b.Services[id]; ok {
			changedIDs = append(changedIDs, id)
		}
	}
	sort.Strings(changedIDs)

	for _, id := range changedIDs {
		svcA := a.Services[id]
		svcB := b.Services[id]
		changes := diffService(svcA, svcB)
		if len(changes) > 0 {
			result.Changed = append(result.Changed, ServiceChange{ID: id, Changes: changes})
		}
	}

	return result
}

// diffService compares two Service structs and returns a list of field-level changes.
func diffService(a, b Service) []FieldChange {
	var changes []FieldChange

	check := func(field, from, to string) {
		if from != to {
			changes = append(changes, FieldChange{Field: field, From: from, To: to})
		}
	}

	check("description", a.Description, b.Description)
	check("host", a.Host, b.Host)
	check("port", portStr(a.Port), portStr(b.Port))
	check("health_url", a.HealthURL, b.HealthURL)
	check("systemd_unit", a.SystemdUnit, b.SystemdUnit)
	check("repo", a.Repo, b.Repo)
	check("version", a.Version, b.Version)
	check("max_major", maxMajorStr(a.MaxMajor), maxMajorStr(b.MaxMajor))
	check("docs", a.Docs, b.Docs)
	check("added", a.Added, b.Added)

	// Tags: compare sorted join.
	aTags := sortedJoin(a.Tags)
	bTags := sortedJoin(b.Tags)
	check("tags", aTags, bTags)

	return changes
}

func portStr(p int) string {
	if p == 0 {
		return ""
	}
	return fmt.Sprintf("%d", p)
}

func maxMajorStr(m int) string {
	if m == 0 {
		return ""
	}
	return fmt.Sprintf("%d", m)
}

func sortedJoin(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	cp := make([]string, len(tags))
	copy(cp, tags)
	sort.Strings(cp)
	return strings.Join(cp, ",")
}
