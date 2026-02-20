package guard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// ReviewedTerms holds the set of agent-cleared false positives.
type ReviewedTerms struct {
	Terms []ReviewedTerm `json:"terms"`
}

// ReviewedTerm is a single cleared term+file combination.
type ReviewedTerm struct {
	Term       string   `json:"term"`
	Category   string   `json:"category"`
	Files      []string `json:"files"`
	Reason     string   `json:"reason"`
	ReviewedBy string   `json:"reviewed_by"`
	ReviewedAt string   `json:"reviewed_at"`
}

// reviewedTermsPath returns the path to the reviewed-terms file.
func reviewedTermsPath(vaultPath string) string {
	return filepath.Join(vaultPath, ".same", "reviewed-terms.json")
}

// LoadReviewedTerms loads reviewed terms from disk.
// Returns empty set if file doesn't exist.
func LoadReviewedTerms(vaultPath string) (*ReviewedTerms, error) {
	path := reviewedTermsPath(vaultPath)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &ReviewedTerms{}, nil
	}
	if err != nil {
		return nil, err
	}

	var rt ReviewedTerms
	if err := json.Unmarshal(data, &rt); err != nil {
		return nil, err
	}
	return &rt, nil
}

// Save writes reviewed terms to disk.
func (rt *ReviewedTerms) Save(vaultPath string) error {
	path := reviewedTermsPath(vaultPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(rt, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Add adds a reviewed term. If the term+category already exists, it merges the files.
func (rt *ReviewedTerms) Add(term, category, reason, reviewedBy string, files []string) {
	for i, t := range rt.Terms {
		if t.Term == term && t.Category == category {
			// Merge files
			fileSet := make(map[string]bool)
			for _, f := range t.Files {
				fileSet[f] = true
			}
			for _, f := range files {
				fileSet[f] = true
			}
			merged := make([]string, 0, len(fileSet))
			for f := range fileSet {
				merged = append(merged, f)
			}
			rt.Terms[i].Files = merged
			rt.Terms[i].Reason = reason
			rt.Terms[i].ReviewedBy = reviewedBy
			rt.Terms[i].ReviewedAt = time.Now().UTC().Format(time.RFC3339)
			return
		}
	}

	rt.Terms = append(rt.Terms, ReviewedTerm{
		Term:       term,
		Category:   category,
		Files:      files,
		Reason:     reason,
		ReviewedBy: reviewedBy,
		ReviewedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

// Remove removes a reviewed term by term string and category.
func (rt *ReviewedTerms) Remove(term, category string) bool {
	for i, t := range rt.Terms {
		if t.Term == term && t.Category == category {
			rt.Terms = append(rt.Terms[:i], rt.Terms[i+1:]...)
			return true
		}
	}
	return false
}

// IsReviewed checks if a term+file+category combination has been reviewed.
func (rt *ReviewedTerms) IsReviewed(term, filePath, category string) bool {
	for _, t := range rt.Terms {
		if t.Term != term || t.Category != category {
			continue
		}
		for _, f := range t.Files {
			// Support glob-like matching: "internal/parser/*.go"
			if matched, _ := filepath.Match(f, filePath); matched {
				return true
			}
			// Exact match
			if f == filePath {
				return true
			}
		}
	}
	return false
}
