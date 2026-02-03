package hooks

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sgx-labs/statelessagent/internal/embedding"
	"github.com/sgx-labs/statelessagent/internal/memory"
	"github.com/sgx-labs/statelessagent/internal/store"
)

const (
	minPromptChars   = 20
	maxResults       = 2   // 2 high-quality results > 3 noisy ones
	maxSnippetChars  = 300
	maxDistance       = 16.5 // L2 distance (not squared); good queries < 16.3, off-topic > 16.8
	minComposite     = 0.65 // raised from 0.6; fewer false positives
	minSemanticFloor = 0.25 // absolute floor: if semantic score < this, skip regardless of boost
	maxTokenBudget   = 800  // tightened from 1000; less context waste
)

// Recency-aware weights: when query has recency intent, shift weight heavily to recency.
const (
	recencyRelWeight  = 0.1
	recencyRecWeight  = 0.7
	recencyConfWeight = 0.2
	recencyMinComposite = 0.45 // lower threshold since semantic score may be weak
	recencyMaxResults   = 3    // show more results for "what did I work on" queries
)

var priorityTypes = map[string]bool{
	"handoff":  true,
	"decision": true,
	"research": true,
	"hub":      true,
}

// SECURITY: Paths that must never be auto-surfaced via hooks.
// _PRIVATE/ contains client-sensitive content. Defense-in-depth:
// indexer also skips these, but we filter here in case of stale index data.
const privateDirPrefix = "_PRIVATE/"

// Prompt injection patterns — content matching these is stripped from snippets
// before injection. Prevents vault notes from hijacking agent behavior.
var injectionPatterns = []string{
	"ignore previous",
	"ignore all previous",
	"ignore above",
	"disregard previous",
	"disregard all previous",
	"you are now",
	"new instructions",
	"system prompt",
	"<system>",
	"</system>",
	"IMPORTANT:",
	"CRITICAL:",
	"override",
}

type scored struct {
	path        string
	title       string
	contentType string
	confidence  float64
	snippet     string
	composite   float64
	semantic    float64
	distance    float64
}

// runContextSurfacing embeds the user's prompt, searches the vault,
// and injects relevant context.
func runContextSurfacing(db *store.DB, input *HookInput) *HookOutput {
	prompt := input.Prompt
	if len(prompt) < minPromptChars {
		return nil
	}

	// Skip slash commands
	if strings.HasPrefix(strings.TrimSpace(prompt), "/") {
		return nil
	}

	isRecency := memory.HasRecencyIntent(prompt)

	// Embed the prompt
	client := embedding.NewClient()
	queryVec, err := client.GetQueryEmbedding(prompt)
	if err != nil {
		return nil
	}

	var candidates []scored

	if isRecency {
		candidates = recencyHybridSearch(db, queryVec)
	} else {
		candidates = standardSearch(db, queryVec)
	}

	if len(candidates) == 0 {
		return nil
	}

	effectiveMax := maxResults
	if isRecency {
		effectiveMax = recencyMaxResults
	}
	if len(candidates) > effectiveMax {
		candidates = candidates[:effectiveMax]
	}

	// Build context string, capped at token budget
	var parts []string
	totalTokens := 0
	for _, s := range candidates {
		entry := fmt.Sprintf("**%s** (%s, score: %.3f)\n%s\n%s",
			s.title, s.contentType, s.composite, s.path, s.snippet)
		entryTokens := memory.EstimateTokens(entry)
		if totalTokens+entryTokens > maxTokenBudget {
			break
		}
		parts = append(parts, entry)
		totalTokens += entryTokens
	}

	if len(parts) == 0 {
		return nil
	}

	// Collect injected paths for usage tracking
	var injectedPaths []string
	for _, s := range candidates[:len(parts)] {
		injectedPaths = append(injectedPaths, s.path)
	}

	contextText := strings.Join(parts, "\n---\n")

	// Log the injection for budget tracking
	if input.SessionID != "" {
		memory.LogInjection(db, input.SessionID, "context_surfacing", injectedPaths, contextText)
	}

	return &HookOutput{
		HookSpecificOutput: &HookSpecific{
			HookEventName: "UserPromptSubmit",
			AdditionalContext: fmt.Sprintf(
				"\n<vault-context>\nRelevant vault notes for this prompt:\n\n%s\n</vault-context>\n",
				contextText,
			),
		},
	}
}

// standardSearch performs the original vector-search-based retrieval.
func standardSearch(db *store.DB, queryVec []float32) []scored {
	raw, err := db.VectorSearchRaw(queryVec, maxResults*6)
	if err != nil || len(raw) == 0 {
		return nil
	}

	if raw[0].Distance > maxDistance {
		return nil
	}

	deduped := dedup(raw)
	if len(deduped) == 0 {
		return nil
	}

	minDist, maxDist := distRange(deduped)
	dRange := maxDist - minDist
	if dRange <= 0 {
		dRange = 1.0
	}

	var candidates []scored
	for _, r := range deduped {
		if r.Distance > maxDistance {
			continue
		}
		// SECURITY: never auto-surface _PRIVATE/ content
		if isPrivatePath(r.Path) {
			continue
		}

		semScore := 1.0 - ((r.Distance - minDist) / dRange)
		if semScore < minSemanticFloor {
			continue
		}

		comp := memory.CompositeScore(semScore, r.Modified, r.Confidence, r.ContentType,
			0.3, 0.3, 0.4)
		if comp < minComposite {
			continue
		}

		candidates = append(candidates, makeScored(r, comp, semScore))
	}

	sort.Slice(candidates, func(i, j int) bool {
		iPri := priorityTypes[candidates[i].contentType]
		jPri := priorityTypes[candidates[j].contentType]
		if iPri != jPri {
			return iPri
		}
		return candidates[i].composite > candidates[j].composite
	})

	return candidates
}

// recencyHybridSearch merges vector results with time-sorted results.
// Uses recency-heavy weights and includes recently modified notes even
// if they aren't strong semantic matches.
func recencyHybridSearch(db *store.DB, queryVec []float32) []scored {
	// Get vector search results (relaxed distance threshold)
	raw, err := db.VectorSearchRaw(queryVec, recencyMaxResults*6)
	if err != nil {
		raw = nil
	}

	// Get most recently modified notes
	recentNotes, err := db.RecentNotes(recencyMaxResults * 3)
	if err != nil {
		recentNotes = nil
	}

	// Merge: build candidate set from both sources
	candidateMap := make(map[string]*scored)

	// Process vector results (if any matched)
	if len(raw) > 0 {
		deduped := dedup(raw)
		minDist, maxDist := distRange(deduped)
		dRange := maxDist - minDist
		if dRange <= 0 {
			dRange = 1.0
		}

		for _, r := range deduped {
			// Relaxed distance gate for recency queries
			if r.Distance > maxDistance+2.0 {
				continue
			}
			// SECURITY: never auto-surface _PRIVATE/ content
			if isPrivatePath(r.Path) {
				continue
			}
			semScore := 1.0 - ((r.Distance - minDist) / dRange)
			if semScore < 0 {
				semScore = 0
			}

			comp := memory.CompositeScore(semScore, r.Modified, r.Confidence, r.ContentType,
				recencyRelWeight, recencyRecWeight, recencyConfWeight)

			if comp >= recencyMinComposite {
				s := makeScored(r, comp, semScore)
				candidateMap[r.Path] = &s
			}
		}
	}

	// Process recent notes (time-sorted, no vector match required)
	for _, n := range recentNotes {
		if _, exists := candidateMap[n.Path]; exists {
			continue // already from vector results, keep that score
		}
		// SECURITY: never auto-surface _PRIVATE/ content
		if isPrivatePath(n.Path) {
			continue
		}

		// Score purely on recency + confidence (no semantic component)
		comp := memory.CompositeScore(0, n.Modified, n.Confidence, n.ContentType,
			recencyRelWeight, recencyRecWeight, recencyConfWeight)

		if comp >= recencyMinComposite {
			snippet := n.Text
			if len(snippet) > maxSnippetChars {
				snippet = snippet[:maxSnippetChars]
			}
			candidateMap[n.Path] = &scored{
				path:        n.Path,
				title:       n.Title,
				contentType: n.ContentType,
				confidence:  n.Confidence,
				snippet:     snippet,
				composite:   comp,
				semantic:    0,
				distance:    0,
			}
		}
	}

	// Collect and sort by composite (recency-heavy)
	var candidates []scored
	for _, s := range candidateMap {
		candidates = append(candidates, *s)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].composite > candidates[j].composite
	})

	return candidates
}

func dedup(raw []store.RawSearchResult) []store.RawSearchResult {
	seen := make(map[string]bool)
	var out []store.RawSearchResult
	for _, r := range raw {
		if seen[r.Path] {
			continue
		}
		seen[r.Path] = true
		out = append(out, r)
	}
	return out
}

func distRange(results []store.RawSearchResult) (float64, float64) {
	minD := results[0].Distance
	maxD := results[0].Distance
	for _, r := range results[1:] {
		if r.Distance < minD {
			minD = r.Distance
		}
		if r.Distance > maxD {
			maxD = r.Distance
		}
	}
	return minD, maxD
}

func makeScored(r store.RawSearchResult, comp, sem float64) scored {
	snippet := r.Text
	if len(snippet) > maxSnippetChars {
		snippet = snippet[:maxSnippetChars]
	}
	snippet = sanitizeSnippet(snippet)
	return scored{
		path:        r.Path,
		title:       r.Title,
		contentType: r.ContentType,
		confidence:  r.Confidence,
		snippet:     snippet,
		composite:   comp,
		semantic:    sem,
		distance:    r.Distance,
	}
}

// isPrivatePath returns true if the path is under the _PRIVATE/ directory.
func isPrivatePath(path string) bool {
	return strings.HasPrefix(path, privateDirPrefix) ||
		strings.HasPrefix(path, "_PRIVATE\\")
}

// sanitizeSnippet removes prompt injection patterns from snippet text.
func sanitizeSnippet(text string) string {
	lower := strings.ToLower(text)
	for _, pattern := range injectionPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			text = "[content filtered for security]"
			break
		}
	}
	return text
}
