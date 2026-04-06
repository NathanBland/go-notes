package mcpapi

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/nathanbland/go-notes/internal/notes"
)

func normalizeListArgs(ownerID uuid.UUID, args listNotesArgs) (notes.ListFilters, error) {
	return normalizeListValues(ownerID, valuesFromListArgs(args))
}

func normalizeListValues(ownerID uuid.UUID, values url.Values) (notes.ListFilters, error) {
	filters := notes.ListFilters{
		OwnerUserID:   ownerID,
		Page:          1,
		PageSize:      20,
		SearchMode:    notes.SearchModePlain,
		Status:        notes.StatusActive,
		TagMatchMode:  notes.TagMatchAny,
		SortField:     "updated_at",
		SortDirection: "desc",
	}

	if page := strings.TrimSpace(values.Get("page")); page != "" {
		parsed, err := strconv.Atoi(page)
		if err != nil {
			return notes.ListFilters{}, fmt.Errorf("page must be at least 1")
		}
		filters.Page = int32(parsed)
	}
	if pageSize := strings.TrimSpace(values.Get("page_size")); pageSize != "" {
		parsed, err := strconv.Atoi(pageSize)
		if err != nil {
			return notes.ListFilters{}, fmt.Errorf("page_size must be between 1 and 100")
		}
		filters.PageSize = int32(parsed)
	}
	if filters.Page < 1 {
		return notes.ListFilters{}, fmt.Errorf("page must be at least 1")
	}
	if filters.PageSize < 1 || filters.PageSize > 100 {
		return notes.ListFilters{}, fmt.Errorf("page_size must be between 1 and 100")
	}

	if search := strings.TrimSpace(values.Get("search")); search != "" {
		filters.Search = &search
	}
	if mode := strings.TrimSpace(values.Get("search_mode")); mode != "" {
		switch mode {
		case notes.SearchModePlain, notes.SearchModeFTS:
			filters.SearchMode = mode
		default:
			return notes.ListFilters{}, fmt.Errorf("search_mode must be plain or fts")
		}
	}
	if status := strings.TrimSpace(values.Get("status")); status != "" {
		switch status {
		case notes.StatusActive, notes.StatusArchived, notes.StatusAll:
			filters.Status = status
		default:
			return notes.ListFilters{}, fmt.Errorf("status must be active, archived, or all")
		}
	}
	if sharedRaw := strings.TrimSpace(values.Get("shared")); sharedRaw != "" {
		shared, err := strconv.ParseBool(sharedRaw)
		if err != nil {
			return notes.ListFilters{}, fmt.Errorf("shared must be true or false")
		}
		filters.Shared = &shared
	}
	if hasTitleRaw := strings.TrimSpace(values.Get("has_title")); hasTitleRaw != "" {
		hasTitle, err := strconv.ParseBool(hasTitleRaw)
		if err != nil {
			return notes.ListFilters{}, fmt.Errorf("has_title must be true or false")
		}
		filters.HasTitle = &hasTitle
	}
	tags := make([]string, 0, len(values["tags"])+len(values["tag"]))
	for _, tag := range values["tag"] {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			tags = append(tags, trimmed)
		}
	}
	for _, group := range values["tags"] {
		for _, tag := range strings.Split(group, ",") {
			trimmed := strings.TrimSpace(tag)
			if trimmed != "" {
				tags = append(tags, trimmed)
			}
		}
	}
	if len(tags) > 0 {
		filters.Tags = uniqueStrings(tags)
	}
	if raw := strings.TrimSpace(values.Get("tag_count_min")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return notes.ListFilters{}, fmt.Errorf("tag_count_min must be greater than or equal to 0")
		}
		value := int32(parsed)
		if value < 0 {
			return notes.ListFilters{}, fmt.Errorf("tag_count_min must be greater than or equal to 0")
		}
		filters.TagCountMin = &value
	}
	if raw := strings.TrimSpace(values.Get("tag_count_max")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return notes.ListFilters{}, fmt.Errorf("tag_count_max must be greater than or equal to 0")
		}
		value := int32(parsed)
		if value < 0 {
			return notes.ListFilters{}, fmt.Errorf("tag_count_max must be greater than or equal to 0")
		}
		filters.TagCountMax = &value
	}
	if filters.TagCountMin != nil && filters.TagCountMax != nil && *filters.TagCountMin > *filters.TagCountMax {
		return notes.ListFilters{}, fmt.Errorf("tag_count_max must be greater than or equal to tag_count_min")
	}
	if mode := strings.TrimSpace(values.Get("tag_mode")); mode != "" {
		switch mode {
		case notes.TagMatchAny, notes.TagMatchAll:
			filters.TagMatchMode = mode
		default:
			return notes.ListFilters{}, fmt.Errorf("tag_mode must be any or all")
		}
	}
	if sort := strings.TrimSpace(values.Get("sort")); sort != "" {
		switch sort {
		case "created_at", "updated_at", "title", "tag_count", "primary_tag", "relevance":
			filters.SortField = sort
		default:
			return notes.ListFilters{}, fmt.Errorf("sort must be created_at, updated_at, title, tag_count, primary_tag, or relevance")
		}
	}
	if order := strings.TrimSpace(values.Get("order")); order != "" {
		switch order {
		case "asc", "desc":
			filters.SortDirection = order
		default:
			return notes.ListFilters{}, fmt.Errorf("order must be asc or desc")
		}
	}

	if err := assignTimeArg(values.Get("created_after"), &filters.CreatedAfter, "created_after"); err != nil {
		return notes.ListFilters{}, err
	}
	if err := assignTimeArg(values.Get("created_before"), &filters.CreatedBefore, "created_before"); err != nil {
		return notes.ListFilters{}, err
	}
	if err := assignTimeArg(values.Get("updated_after"), &filters.UpdatedAfter, "updated_after"); err != nil {
		return notes.ListFilters{}, err
	}
	if err := assignTimeArg(values.Get("updated_before"), &filters.UpdatedBefore, "updated_before"); err != nil {
		return notes.ListFilters{}, err
	}
	if filters.SortField == "relevance" {
		if filters.Search == nil || strings.TrimSpace(*filters.Search) == "" {
			return notes.ListFilters{}, fmt.Errorf("search is required when sort is relevance")
		}
		if filters.SearchMode != notes.SearchModeFTS {
			return notes.ListFilters{}, fmt.Errorf("search_mode must be fts when sort is relevance")
		}
	}
	if filters.SearchMode == notes.SearchModeFTS && (filters.Search == nil || strings.TrimSpace(*filters.Search) == "") {
		return notes.ListFilters{}, fmt.Errorf("search is required when search_mode is fts")
	}

	return filters, nil
}

func valuesFromListArgs(args listNotesArgs) url.Values {
	values := url.Values{}
	if args.Page != nil {
		values.Set("page", strconv.Itoa(*args.Page))
	}
	if args.PageSize != nil {
		values.Set("page_size", strconv.Itoa(*args.PageSize))
	}
	if args.Search != nil {
		values.Set("search", *args.Search)
	}
	if args.SearchMode != nil {
		values.Set("search_mode", *args.SearchMode)
	}
	if args.Status != nil {
		values.Set("status", *args.Status)
	}
	if args.Shared != nil {
		values.Set("shared", strconv.FormatBool(*args.Shared))
	}
	if args.HasTitle != nil {
		values.Set("has_title", strconv.FormatBool(*args.HasTitle))
	}
	if args.Tag != nil {
		values.Set("tag", *args.Tag)
	}
	for _, tag := range args.Tags {
		values.Add("tag", tag)
	}
	if args.TagCountMin != nil {
		values.Set("tag_count_min", strconv.Itoa(*args.TagCountMin))
	}
	if args.TagCountMax != nil {
		values.Set("tag_count_max", strconv.Itoa(*args.TagCountMax))
	}
	if args.TagMode != nil {
		values.Set("tag_mode", *args.TagMode)
	}
	if args.Sort != nil {
		values.Set("sort", *args.Sort)
	}
	if args.Order != nil {
		values.Set("order", *args.Order)
	}
	if args.CreatedAfter != nil {
		values.Set("created_after", *args.CreatedAfter)
	}
	if args.CreatedBefore != nil {
		values.Set("created_before", *args.CreatedBefore)
	}
	if args.UpdatedAfter != nil {
		values.Set("updated_after", *args.UpdatedAfter)
	}
	if args.UpdatedBefore != nil {
		values.Set("updated_before", *args.UpdatedBefore)
	}
	return values
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func assignTimeArg(value string, target **time.Time, field string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return fmt.Errorf("%s must be an RFC3339 timestamp", field)
	}
	utc := parsed.UTC()
	*target = &utc
	return nil
}
