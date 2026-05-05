package game

import (
	"math/rand"
	"sort"
)

const imageSelectionSeedSalt int64 = 0x1f2e3d4c5b6a798

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

// ShuffledImageIDs returns unique image ids in deterministic per-game shuffled order.
func ShuffledImageIDs(settings Settings, ids []string) []string {
	unique := uniqueImageIDs(ids)
	rng := rand.New(rand.NewSource(settings.Seed ^ imageSelectionSeedSalt))
	perm := rng.Perm(len(unique))
	shuffled := make([]string, len(unique))
	for i, idx := range perm {
		shuffled[i] = unique[idx]
	}
	return shuffled
}
