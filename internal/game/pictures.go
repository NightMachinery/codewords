package game

import "sort"

func uniqueImageIDs(ids []string) []string {
	seen := map[string]bool{}
	unique := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		unique = append(unique, id)
	}
	sort.Strings(unique)
	return unique
}
