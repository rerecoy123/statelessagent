package guard

import (
	"os"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// BlocklistConfig represents the _PRIVATE/.blocklist TOML file.
type BlocklistConfig struct {
	Hard BlocklistSection `toml:"hard"`
	Soft BlocklistSection `toml:"soft"`
}

// BlocklistSection is a list of terms for a given tier.
type BlocklistSection struct {
	Terms []string `toml:"terms"`
}

// CompiledTerm is a blocklist term compiled to a case-insensitive regex.
type CompiledTerm struct {
	Term  string
	Tier  Tier
	Regex *regexp.Regexp
}

// LoadBlocklist loads and compiles blocklist terms from a TOML file.
// Returns nil, nil if the file does not exist.
func LoadBlocklist(path string) ([]CompiledTerm, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg BlocklistConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	var terms []CompiledTerm
	for _, t := range cfg.Hard.Terms {
		ct, err := compileTerm(t, TierHard)
		if err != nil {
			continue // skip invalid terms
		}
		terms = append(terms, ct)
	}
	for _, t := range cfg.Soft.Terms {
		ct, err := compileTerm(t, TierSoft)
		if err != nil {
			continue
		}
		terms = append(terms, ct)
	}
	return terms, nil
}

// compileTerm creates a case-insensitive word-boundary regex for a term.
func compileTerm(term string, tier Tier) (CompiledTerm, error) {
	// Escape regex metacharacters in the term
	escaped := regexp.QuoteMeta(term)
	// Case-insensitive, word-boundary match
	pattern := `(?i)\b` + escaped + `\b`
	re, err := regexp.Compile(pattern)
	if err != nil {
		return CompiledTerm{}, err
	}
	return CompiledTerm{
		Term:  term,
		Tier:  tier,
		Regex: re,
	}, nil
}

// BlocklistMatch is a single blocklist match in a file.
type BlocklistMatch struct {
	Term     string
	Tier     Tier
	Category Category
}

// scanBlocklist checks a line against all compiled blocklist terms.
func scanBlocklist(line string, terms []CompiledTerm) []BlocklistMatch {
	var matches []BlocklistMatch
	for _, t := range terms {
		if t.Regex.MatchString(line) {
			cat := CatHardBlock
			if t.Tier == TierSoft {
				cat = CatSoftBlock
			}
			matches = append(matches, BlocklistMatch{
				Term:     t.Term,
				Tier:     t.Tier,
				Category: cat,
			})
		}
	}
	return matches
}

// blocklistPath returns the expected path for the blocklist file.
func blocklistPath(vaultPath string) string {
	return vaultPath + string(os.PathSeparator) + "_PRIVATE" + string(os.PathSeparator) + ".blocklist"
}

// matchesBlocklistTerm checks if content contains a specific blocklist term.
func matchesBlocklistTerm(content string, term string) bool {
	escaped := regexp.QuoteMeta(term)
	pattern := `(?i)\b` + escaped + `\b`
	re, err := regexp.Compile(pattern)
	if err != nil {
		return strings.Contains(strings.ToLower(content), strings.ToLower(term))
	}
	return re.MatchString(content)
}
