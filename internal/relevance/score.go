package relevance

// TitleScore computes how well a result title matches the search query.
// Returns 0.0–1.0 based on what fraction of query tokens appear in the title.
func TitleScore(title, query string) float64 {
	queryTokens := Tokenize(query)
	if len(queryTokens) == 0 {
		return 0.0
	}

	titleTokens := Tokenize(title)
	if len(titleTokens) == 0 {
		return 0.0
	}

	titleSet := make(map[string]bool, len(titleTokens))
	for _, t := range titleTokens {
		titleSet[t] = true
	}

	matched := 0
	for _, q := range queryTokens {
		if titleSet[q] {
			matched++
		}
	}

	return float64(matched) / float64(len(queryTokens))
}

// FieldMatchScore checks if all tokens of filterValue appear in the field.
// Returns 1.0 if all tokens match, 0.0 otherwise. Used for author/publisher matching.
// Returns 0.0 if either argument is empty.
func FieldMatchScore(field, filterValue string) float64 {
	filterTokens := Tokenize(filterValue)
	if len(filterTokens) == 0 {
		return 0.0
	}

	fieldTokens := Tokenize(field)
	if len(fieldTokens) == 0 {
		return 0.0
	}

	fieldSet := make(map[string]bool, len(fieldTokens))
	for _, t := range fieldTokens {
		fieldSet[t] = true
	}

	for _, q := range filterTokens {
		if !fieldSet[q] {
			return 0.0
		}
	}
	return 1.0
}

// ComputeRelevance calculates an overall relevance score (0.0–1.0) for a book result.
//
// Parameters:
//   - title, author, publisher: fields from the BookResult
//   - query: the user's search query
//   - authorFilter, publisherFilter: optional --author/--publisher flag values (empty = not set)
//
// Weights:
//   - Title match: 70% of score (always applied)
//   - Author match: 15% of score (only when --author is set, otherwise redistributes to title)
//   - Publisher match: 15% of score (only when --publisher is set, otherwise redistributes to title)
func ComputeRelevance(title, author, publisher, query, authorFilter, publisherFilter string) float64 {
	titleScore := TitleScore(title, query)

	hasAuthorFilter := authorFilter != ""
	hasPublisherFilter := publisherFilter != ""

	// Dynamic weight redistribution
	titleWeight := 1.0
	authorWeight := 0.0
	publisherWeight := 0.0

	if hasAuthorFilter && hasPublisherFilter {
		titleWeight = 0.70
		authorWeight = 0.15
		publisherWeight = 0.15
	} else if hasAuthorFilter {
		titleWeight = 0.85
		authorWeight = 0.15
	} else if hasPublisherFilter {
		titleWeight = 0.85
		publisherWeight = 0.15
	}

	score := titleScore * titleWeight

	if hasAuthorFilter {
		score += FieldMatchScore(author, authorFilter) * authorWeight
	}
	if hasPublisherFilter {
		score += FieldMatchScore(publisher, publisherFilter) * publisherWeight
	}

	return score
}
