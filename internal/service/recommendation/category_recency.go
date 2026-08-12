package recommendation

var recencyIgnoredCategories = map[string]bool{
	"malaysian":      true,
	"singaporean":    true,
	"indonesian":     true,
	"chinese":        true,
	"indian":         true,
	"thai":           true,
	"vietnamese":     true,
	"japanese":       true,
	"korean":         true,
	"middle_eastern": true,
	"american":       true,
	"mexican":        true,
	"italian":        true,
	"french":         true,
	"greek":          true,
	"spanish":        true,
	"western":        true,
}

func shouldUseCategoryForRecency(category string) bool {
	return category != "" && !recencyIgnoredCategories[category]
}
