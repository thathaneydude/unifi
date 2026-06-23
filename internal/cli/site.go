package cli

import (
	"encoding/json"
	"strconv"
	"strings"
)

// site is the minimal identity of a UniFi site used to resolve a siteId path
// parameter from a friendlier --site value.
type site struct {
	ID                string
	Name              string
	InternalReference string
}

// parseSites reads the {data:[{id,name,internalReference}]} envelope returned by
// the network sites overview operation.
func parseSites(body []byte) ([]site, error) {
	var page struct {
		Data []struct {
			ID                string `json:"id"`
			Name              string `json:"name"`
			InternalReference string `json:"internalReference"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, err
	}
	sites := make([]site, 0, len(page.Data))
	for _, d := range page.Data {
		sites = append(sites, site{ID: d.ID, Name: d.Name, InternalReference: d.InternalReference})
	}
	return sites, nil
}

func siteLabels(sites []site) []string {
	labels := make([]string, 0, len(sites))
	for _, s := range sites {
		labels = append(labels, s.Name+" ("+s.InternalReference+")")
	}
	return labels
}

// selectSite resolves a siteId from a list of sites and a wanted value. An empty
// want auto-selects when exactly one site exists; otherwise want is matched
// case-insensitively against a site's id, internalReference (e.g. "default"), or
// name.
func selectSite(sites []site, want string) (string, error) {
	if want == "" {
		switch len(sites) {
		case 0:
			return "", NewUsageError("no sites found on this console")
		case 1:
			return sites[0].ID, nil
		default:
			return "", NewUsageError("multiple sites; pass --site <name|default|id> — have: " + strings.Join(siteLabels(sites), ", "))
		}
	}
	w := strings.ToLower(want)
	var matches []string
	for _, s := range sites {
		if strings.ToLower(s.ID) == w || strings.ToLower(s.InternalReference) == w || strings.ToLower(s.Name) == w {
			matches = append(matches, s.ID)
		}
	}
	switch len(matches) {
	case 0:
		return "", NewUsageError("no site matches --site " + strconv.Quote(want) + " — have: " + strings.Join(siteLabels(sites), ", "))
	case 1:
		return matches[0], nil
	default:
		return "", NewUsageError("ambiguous --site " + strconv.Quote(want) + " matches multiple sites")
	}
}

// opNeedsSite reports whether op has a siteId path parameter.
func opNeedsSite(op Operation) bool {
	for _, p := range op.PathParams {
		if p.Name == "siteId" {
			return true
		}
	}
	return false
}
