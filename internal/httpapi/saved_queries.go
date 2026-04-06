package httpapi

import (
	"context"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"github.com/nathanbland/go-notes/internal/notes"
)

func (a *API) parseListFiltersWithSavedQuery(ctx context.Context, values url.Values, ownerUserID uuid.UUID) (notes.ListFilters, map[string]string) {
	fields := map[string]string{}
	merged := cloneValues(values)

	savedQueryID := strings.TrimSpace(values.Get("saved_query_id"))
	if savedQueryID != "" {
		parsedID, err := uuid.Parse(savedQueryID)
		if err != nil {
			fields["saved_query_id"] = "must be a valid UUID"
			return notes.ListFilters{}, fields
		}
		savedQuery, err := a.notesService.GetSavedQuery(ctx, ownerUserID, parsedID)
		if err != nil {
			fields["saved_query_id"] = "was not found"
			return notes.ListFilters{}, fields
		}
		if savedQuery.Query != "" {
			baseValues, err := url.ParseQuery(savedQuery.Query)
			if err != nil {
				fields["saved_query_id"] = "contains invalid stored filters"
				return notes.ListFilters{}, fields
			}
			merged = overlayValues(baseValues, values, "saved_query_id")
		}
	}

	filters, parseFields := parseListFilters(merged, ownerUserID)
	for key, value := range parseFields {
		fields[key] = value
	}
	return filters, fields
}

func overlayValues(base, explicit url.Values, ignoredKeys ...string) url.Values {
	result := cloneValues(base)
	ignored := map[string]struct{}{}
	for _, key := range ignoredKeys {
		ignored[key] = struct{}{}
	}
	for key, values := range explicit {
		if _, skip := ignored[key]; skip {
			continue
		}
		result.Del(key)
		for _, value := range values {
			result.Add(key, value)
		}
	}
	return result
}
