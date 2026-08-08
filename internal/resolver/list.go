package resolver

import (
	"os"
	"sort"
)

// Entry summarizes a resolver file for listing.
type Entry struct {
	Name        string   `json:"name"`
	Domain      string   `json:"domain"`
	Nameservers []string `json:"nameservers"`
	Port        string   `json:"port,omitempty"`
	Managed     bool     `json:"managed"`
	Warning     string   `json:"warning,omitempty"`
}

// List returns all readable regular resolver files, sorted by name. Files that
// cannot be read or are irregular are reported with a Warning rather than
// omitted.
func List(resolverDir string) ([]Entry, error) {
	dirEntries, err := os.ReadDir(resolverDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var entries []Entry
	for _, de := range dirEntries {
		name := de.Name()
		e := Entry{Name: name}

		exists, existErr := Exists(resolverDir, name)
		if existErr != nil {
			e.Warning = existErr.Error()
			entries = append(entries, e)
			continue
		}
		if !exists {
			continue
		}
		cfg, _, readErr := Read(resolverDir, name)
		if readErr != nil {
			e.Warning = readErr.Error()
			entries = append(entries, e)
			continue
		}
		e.Domain = cfg.Domain
		if e.Domain == "" {
			e.Domain = name
		}
		e.Nameservers = cfg.Nameservers()
		if port, ok := cfg.Port(); ok {
			e.Port = port
		}
		e.Managed = cfg.Managed
		entries = append(entries, e)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries, nil
}
