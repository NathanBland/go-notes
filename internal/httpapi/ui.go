package httpapi

import (
	"bytes"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"

	"github.com/nathanbland/go-notes/internal/auth"
	"github.com/nathanbland/go-notes/internal/notes"
)

var noteMarkdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
)

var uiTemplates = template.Must(template.New("ui").Funcs(template.FuncMap{
	"displayName":      displayName,
	"noteTitle":        noteTitle,
	"joinTags":         joinTags,
	"tagCSV":           tagCSV,
	"formatTimestamp":  formatTimestamp,
	"checkedAttr":      checkedAttr,
	"selectedNoteLink": selectedNoteLink,
	"editNoteLink":     editNoteLink,
	"tagFilterLink":    tagFilterLink,
	"renderMarkdown":   renderMarkdown,
	"dict":             dict,
}).Parse(uiTemplatesSource))

type workspaceViewModel struct {
	User         auth.User
	Notes        []notes.Note
	SavedQueries []notes.SavedQuery
	SelectedNote *notes.Note
	Editing      bool
	Filters      noteListFormValues
	FilterErrors map[string]string
	FilterQuery  string
	SaveQuery    saveQueryFormValues
	SaveErrors   map[string]string
	RenameTag    renameTagFormValues
	RenameErrors map[string]string
	CreateForm   noteFormValues
	CreateErrors map[string]string
	EditForm     noteFormValues
	EditErrors   map[string]string
}

type noteListFormValues struct {
	SavedQueryID string
	Search       string
	SearchMode   string
	HasTitle     string
	Tags         string
	TagCountMin  string
	TagCountMax  string
	Mode         string
	Sort         string
	Order        string
}

type noteFormValues struct {
	ID       string
	Title    string
	Content  string
	Tags     string
	Archived bool
	Shared   bool
}

type saveQueryFormValues struct {
	Name string
}

type renameTagFormValues struct {
	OldTag string
	NewTag string
}

type noteDetailViewModel struct {
	Note        notes.Note
	Filters     noteListFormValues
	FilterQuery string
}

func (a *API) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		a.handleNotFound(w, r)
		return
	}
	user, ok, invalidateCookie := a.maybeUser(r)
	if !ok {
		if invalidateCookie {
			http.SetCookie(w, a.expiredSessionCookie())
		}
		a.renderHTML(w, http.StatusOK, "landing", nil)
		return
	}

	vm, status := a.workspaceForRequest(r, user, saveQueryFormValues{}, nil, renameTagFormValues{}, nil, noteFormValues{}, nil, noteFormValues{}, nil)
	a.renderFullPage(w, status, vm)
}

func (a *API) handleUILogout(w http.ResponseWriter, r *http.Request) {
	sessionCookie, _ := r.Cookie(a.sessionCookieName)
	if sessionCookie != nil {
		_ = a.authService.Logout(r.Context(), sessionCookie.Value)
	}
	http.SetCookie(w, a.expiredSessionCookie())
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *API) handlePublicSharedNote(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if !validShareSlug(slug) {
		a.renderHTML(w, http.StatusBadRequest, "shared_note_error", map[string]string{
			"Title":   "That share link is not valid.",
			"Message": "Check the link and try again.",
		})
		return
	}
	note, err := a.notesService.GetByShareSlug(r.Context(), slug)
	if err != nil {
		a.renderHTML(w, http.StatusNotFound, "shared_note_error", map[string]string{
			"Title":   "Shared note not found.",
			"Message": "This note may be private now, or the link may no longer exist.",
		})
		return
	}
	a.renderHTML(w, http.StatusOK, "shared_note_page", note)
}

func (a *API) handleUIShowNote(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	noteID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		a.renderHTML(w, http.StatusBadRequest, "note_error", map[string]string{"Message": "Note ID must be a valid UUID."})
		return
	}
	note, err := a.notesService.GetByIDForOwner(r.Context(), user.ID, noteID)
	if err != nil {
		a.renderHTML(w, a.noteHTMLStatus(err), "note_error", map[string]string{"Message": "Unable to load that note."})
		return
	}
	_, filterForm, _ := a.parseUIListFilters(r, user.ID)
	a.renderHTML(w, http.StatusOK, "note_detail", noteDetailViewModel{
		Note:        note,
		Filters:     filterForm,
		FilterQuery: buildFilterQuery(filterForm),
	})
}

func (a *API) handleUIEditNote(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	noteID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		a.renderHTML(w, http.StatusBadRequest, "note_error", map[string]string{"Message": "Note ID must be a valid UUID."})
		return
	}
	note, err := a.notesService.GetByIDForOwner(r.Context(), user.ID, noteID)
	if err != nil {
		a.renderHTML(w, a.noteHTMLStatus(err), "note_error", map[string]string{"Message": "Unable to load that note for editing."})
		return
	}
	_, filterForm, _ := a.parseUIListFilters(r, user.ID)
	a.renderHTML(w, http.StatusOK, "note_edit", struct {
		Form        noteFormValues
		Errors      map[string]string
		Filters     noteListFormValues
		FilterQuery string
	}{
		Form:        noteToForm(note),
		Errors:      nil,
		Filters:     filterForm,
		FilterQuery: buildFilterQuery(filterForm),
	})
}

func (a *API) handleUICreateNote(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	input, form, fields := parseCreateNoteForm(r, user.ID)
	if len(fields) > 0 {
		vm, _ := a.workspaceForRequest(r, user, saveQueryFormValues{}, nil, renameTagFormValues{}, nil, form, fields, noteFormValues{}, nil)
		status := http.StatusBadRequest
		a.renderWorkspaceOrPage(w, r, status, vm)
		return
	}

	created, err := a.notesService.Create(r.Context(), input)
	if err != nil {
		filters, filterForm, filterErrors := a.parseUIListFilters(r, user.ID)
		vm := a.workspaceForSelection(r, user, filters, filterForm, filterErrors, nil, false, saveQueryFormValues{}, nil, renameTagFormValues{}, nil, form, map[string]string{"content": "Unable to create the note right now."}, noteFormValues{}, nil)
		status := http.StatusInternalServerError
		a.renderWorkspaceOrPage(w, r, status, vm)
		return
	}
	a.redirectOrRenderWorkspace(w, r, user, &created, false)
}

func (a *API) handleUIUpdateNote(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	noteID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		a.renderHTML(w, http.StatusBadRequest, "note_error", map[string]string{"Message": "Note ID must be a valid UUID."})
		return
	}

	input, form, fields := parseUpdateNoteForm(r, noteID, user.ID)
	if len(fields) > 0 {
		filters, filterForm, filterErrors := a.parseUIListFilters(r, user.ID)
		vm := a.workspaceForSelection(r, user, filters, filterForm, filterErrors, nil, true, saveQueryFormValues{}, nil, renameTagFormValues{}, nil, noteFormValues{}, nil, form, fields)
		status := http.StatusBadRequest
		a.renderWorkspaceOrPage(w, r, status, vm)
		return
	}

	updated, err := a.notesService.Patch(r.Context(), input)
	if err != nil {
		status := a.noteHTMLStatus(err)
		filters, filterForm, filterErrors := a.parseUIListFilters(r, user.ID)
		vm := a.workspaceForSelection(r, user, filters, filterForm, filterErrors, nil, true, saveQueryFormValues{}, nil, renameTagFormValues{}, nil, noteFormValues{}, nil, form, map[string]string{"content": "Unable to update that note right now."})
		a.renderWorkspaceOrPage(w, r, status, vm)
		return
	}
	a.redirectOrRenderWorkspace(w, r, user, &updated, false)
}

func (a *API) handleUICreateSavedQuery(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	_ = r.ParseForm()
	saveForm := saveQueryFormValues{Name: strings.TrimSpace(r.FormValue("saved_query_name"))}
	filters, filterForm, filterErrors := a.parseUIListFilters(r, user.ID)
	saveErrors := map[string]string{}
	if saveForm.Name == "" {
		saveErrors["name"] = "Saved query name is required."
	}
	if len(filterErrors) > 0 || len(saveErrors) > 0 {
		vm := a.workspaceForSelection(r, user, filters, filterForm, filterErrors, nil, false, saveForm, saveErrors, renameTagFormValues{}, nil, noteFormValues{}, nil, noteFormValues{}, nil)
		a.renderWorkspaceOrPage(w, r, http.StatusBadRequest, vm)
		return
	}
	_, err := a.notesService.CreateSavedQuery(r.Context(), notes.CreateSavedQueryInput{
		OwnerUserID: user.ID,
		Name:        saveForm.Name,
		Query:       notes.EncodeListFilters(filters),
	})
	if err != nil {
		vm := a.workspaceForSelection(r, user, filters, filterForm, filterErrors, nil, false, saveForm, map[string]string{"name": "Unable to save that query right now."}, renameTagFormValues{}, nil, noteFormValues{}, nil, noteFormValues{}, nil)
		a.renderWorkspaceOrPage(w, r, http.StatusInternalServerError, vm)
		return
	}
	a.redirectOrRenderWorkspace(w, r, user, nil, false)
}

func (a *API) handleUIDeleteSavedQuery(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	savedQueryID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if err := a.notesService.DeleteSavedQuery(r.Context(), user.ID, savedQueryID); err != nil && err != notes.ErrNotFound {
		filters, filterForm, filterErrors := a.parseUIListFilters(r, user.ID)
		vm := a.workspaceForSelection(r, user, filters, filterForm, filterErrors, nil, false, saveQueryFormValues{}, map[string]string{"name": "Unable to delete that saved query right now."}, renameTagFormValues{}, nil, noteFormValues{}, nil, noteFormValues{}, nil)
		a.renderWorkspaceOrPage(w, r, http.StatusInternalServerError, vm)
		return
	}
	redirectTo := "/"
	if query := strings.TrimSpace(r.FormValue("return_query")); query != "" {
		redirectTo = "/?" + query
	}
	http.Redirect(w, r, redirectTo, http.StatusSeeOther)
}

func (a *API) handleUIRenameTag(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	form, fields := parseRenameTagForm(r)
	filters, filterForm, filterErrors := a.parseUIListFilters(r, user.ID)
	if len(fields) > 0 {
		vm := a.workspaceForSelection(r, user, filters, filterForm, filterErrors, nil, false, saveQueryFormValues{}, nil, form, fields, noteFormValues{}, nil, noteFormValues{}, nil)
		a.renderWorkspaceOrPage(w, r, http.StatusBadRequest, vm)
		return
	}
	if _, err := a.notesService.RenameTag(r.Context(), user.ID, form.OldTag, form.NewTag); err != nil {
		vm := a.workspaceForSelection(r, user, filters, filterForm, filterErrors, nil, false, saveQueryFormValues{}, nil, form, map[string]string{"new_tag": "Unable to rename that tag right now."}, noteFormValues{}, nil, noteFormValues{}, nil)
		a.renderWorkspaceOrPage(w, r, http.StatusInternalServerError, vm)
		return
	}
	a.redirectOrRenderWorkspace(w, r, user, nil, false)
}

func (a *API) maybeUser(r *http.Request) (auth.User, bool, bool) {
	sessionCookie, err := r.Cookie(a.sessionCookieName)
	if err != nil || sessionCookie.Value == "" {
		return auth.User{}, false, false
	}
	user, _, authErr := a.authService.Authenticate(r.Context(), sessionCookie.Value)
	if authErr != nil {
		return auth.User{}, false, true
	}
	return user, true, false
}

func (a *API) workspaceForRequest(r *http.Request, user auth.User, saveForm saveQueryFormValues, saveErrors map[string]string, renameForm renameTagFormValues, renameErrors map[string]string, createForm noteFormValues, createErrors map[string]string, editForm noteFormValues, editErrors map[string]string) (workspaceViewModel, int) {
	filters, filterForm, filterErrors := a.parseUIListFilters(r, user.ID)
	status := http.StatusOK
	if len(filterErrors) > 0 {
		status = http.StatusBadRequest
		filters = defaultUIFilters(user.ID)
	}
	selected, editing, selectedStatus := a.selectedNoteFromQuery(r, user.ID)
	if selectedStatus != http.StatusOK && status == http.StatusOK {
		status = selectedStatus
	}
	return a.workspaceForSelection(r, user, filters, filterForm, filterErrors, selected, editing, saveForm, saveErrors, renameForm, renameErrors, createForm, createErrors, editForm, editErrors), status
}

func (a *API) workspaceForSelection(r *http.Request, user auth.User, filters notes.ListFilters, filterForm noteListFormValues, filterErrors map[string]string, selected *notes.Note, editing bool, saveForm saveQueryFormValues, saveErrors map[string]string, renameForm renameTagFormValues, renameErrors map[string]string, createForm noteFormValues, createErrors map[string]string, editForm noteFormValues, editErrors map[string]string) workspaceViewModel {
	result, err := a.notesService.List(r.Context(), filters)
	items := []notes.Note{}
	if err == nil {
		items = result.Notes
	}
	savedQueries, savedQueriesErr := a.notesService.ListSavedQueries(r.Context(), user.ID)
	if savedQueriesErr != nil {
		savedQueries = []notes.SavedQuery{}
	}
	if selected == nil && len(items) > 0 {
		selected = &items[0]
	}
	if editing && selected != nil && editForm.ID == "" {
		editForm = noteToForm(*selected)
	}
	return workspaceViewModel{
		User:         user,
		Notes:        items,
		SavedQueries: savedQueries,
		SelectedNote: selected,
		Editing:      editing,
		Filters:      filterForm,
		FilterErrors: filterErrors,
		FilterQuery:  buildFilterQuery(filterForm),
		SaveQuery:    saveForm,
		SaveErrors:   saveErrors,
		RenameTag:    renameForm,
		RenameErrors: renameErrors,
		CreateForm:   createForm,
		CreateErrors: createErrors,
		EditForm:     editForm,
		EditErrors:   editErrors,
	}
}

func (a *API) selectedNoteFromQuery(r *http.Request, ownerID uuid.UUID) (*notes.Note, bool, int) {
	noteID := strings.TrimSpace(r.URL.Query().Get("note"))
	if noteID == "" {
		return nil, false, http.StatusOK
	}
	parsedID, err := uuid.Parse(noteID)
	if err != nil {
		return nil, false, http.StatusBadRequest
	}
	note, err := a.notesService.GetByIDForOwner(r.Context(), ownerID, parsedID)
	if err != nil {
		return nil, false, a.noteHTMLStatus(err)
	}
	return &note, r.URL.Query().Get("edit") == "1", http.StatusOK
}

func (a *API) redirectOrRenderWorkspace(w http.ResponseWriter, r *http.Request, user auth.User, selected *notes.Note, editing bool) {
	filters, filterForm, filterErrors := a.parseUIListFilters(r, user.ID)
	if len(filterErrors) > 0 {
		filters = defaultUIFilters(user.ID)
		filterForm = noteListFormValues{Mode: notes.TagMatchAny, Sort: "updated_at", Order: "desc"}
	}
	if !isHTMX(r) {
		target := "/"
		if selected != nil {
			target = selectedNoteLink(*selected, buildFilterQuery(filterForm))
		}
		http.Redirect(w, r, target, http.StatusSeeOther)
		return
	}
	vm := a.workspaceForSelection(r, user, filters, filterForm, filterErrors, selected, editing, saveQueryFormValues{}, nil, renameTagFormValues{}, nil, noteFormValues{}, nil, noteFormValues{}, nil)
	a.renderHTML(w, http.StatusOK, "workspace", vm)
}

func (a *API) renderWorkspaceOrPage(w http.ResponseWriter, r *http.Request, status int, vm workspaceViewModel) {
	if isHTMX(r) {
		a.renderHTML(w, status, "workspace", vm)
		return
	}
	a.renderFullPage(w, status, vm)
}

func (a *API) renderFullPage(w http.ResponseWriter, status int, vm workspaceViewModel) {
	a.renderHTML(w, status, "app_page", struct {
		Workspace workspaceViewModel
	}{
		Workspace: vm,
	})
}

func (a *API) renderHTML(w http.ResponseWriter, status int, name string, data any) {
	var buf bytes.Buffer
	if err := uiTemplates.ExecuteTemplate(&buf, name, data); err != nil {
		if a.logger != nil {
			a.logger.Error("failed to render HTML template", "template", name, "error", err)
		}
		http.Error(w, "template render failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func (a *API) noteHTMLStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if err == notes.ErrNotFound {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

func defaultUIFilters(ownerID uuid.UUID) notes.ListFilters {
	return notes.ListFilters{
		OwnerUserID:   ownerID,
		Page:          1,
		PageSize:      20,
		SearchMode:    notes.SearchModePlain,
		Status:        notes.StatusActive,
		TagMatchMode:  notes.TagMatchAny,
		SortField:     "updated_at",
		SortDirection: "desc",
	}
}

func (a *API) parseUIListFilters(r *http.Request, ownerID uuid.UUID) (notes.ListFilters, noteListFormValues, map[string]string) {
	_ = r.ParseForm()
	values := cloneValues(r.Form)
	copyIfMissing(values, "saved_query_id", "ui_saved_query_id")
	copyIfMissing(values, "tags", "ui_tags")
	copyIfMissing(values, "search", "ui_search")
	copyIfMissing(values, "search_mode", "ui_search_mode")
	copyIfMissing(values, "has_title", "ui_has_title")
	copyIfMissing(values, "tag_count_min", "ui_tag_count_min")
	copyIfMissing(values, "tag_count_max", "ui_tag_count_max")
	copyIfMissing(values, "tag_mode", "ui_tag_mode")
	copyIfMissing(values, "sort", "ui_sort")
	copyIfMissing(values, "order", "ui_order")
	filters, fields := a.parseListFiltersWithSavedQuery(r.Context(), values, ownerID)
	form := noteListFormValues{
		SavedQueryID: strings.TrimSpace(values.Get("saved_query_id")),
		Search:       strings.TrimSpace(values.Get("search")),
		SearchMode:   filters.SearchMode,
		HasTitle:     strings.TrimSpace(values.Get("has_title")),
		Tags:         strings.Join(parseTagFilters(values), ", "),
		TagCountMin:  strings.TrimSpace(values.Get("tag_count_min")),
		TagCountMax:  strings.TrimSpace(values.Get("tag_count_max")),
		Mode:         filters.TagMatchMode,
		Sort:         filters.SortField,
		Order:        filters.SortDirection,
	}
	if form.SearchMode == "" {
		form.SearchMode = notes.SearchModePlain
	}
	if form.Mode == "" {
		form.Mode = notes.TagMatchAny
	}
	if form.Sort == "" {
		form.Sort = "updated_at"
	}
	if form.Order == "" {
		form.Order = "desc"
	}
	return filters, form, fields
}

func cloneValues(values url.Values) url.Values {
	cloned := url.Values{}
	for key, items := range values {
		for _, item := range items {
			cloned.Add(key, item)
		}
	}
	return cloned
}

func copyIfMissing(values url.Values, target, fallback string) {
	if len(values[target]) > 0 || len(values[fallback]) == 0 {
		return
	}
	for _, value := range values[fallback] {
		values.Add(target, value)
	}
}

func parseCreateNoteForm(r *http.Request, ownerID uuid.UUID) (notes.CreateInput, noteFormValues, map[string]string) {
	_ = r.ParseForm()
	form := noteFormValues{
		Title:    strings.TrimSpace(r.FormValue("title")),
		Content:  strings.TrimSpace(r.FormValue("content")),
		Tags:     strings.TrimSpace(r.FormValue("tags")),
		Archived: checkboxValue(r, "archived"),
		Shared:   checkboxValue(r, "shared"),
	}
	fields := map[string]string{}
	if form.Content == "" {
		fields["content"] = "Content is required."
	}

	var title *string
	if form.Title != "" {
		title = &form.Title
	}
	input := notes.CreateInput{
		OwnerUserID: ownerID,
		Title:       title,
		Content:     form.Content,
		Tags:        parseTags(form.Tags),
		Archived:    form.Archived,
		Shared:      form.Shared,
	}
	return input, form, fields
}

func parseUpdateNoteForm(r *http.Request, noteID, ownerID uuid.UUID) (notes.PatchInput, noteFormValues, map[string]string) {
	_ = r.ParseForm()
	form := noteFormValues{
		ID:       noteID.String(),
		Title:    strings.TrimSpace(r.FormValue("title")),
		Content:  strings.TrimSpace(r.FormValue("content")),
		Tags:     strings.TrimSpace(r.FormValue("tags")),
		Archived: checkboxValue(r, "archived"),
		Shared:   checkboxValue(r, "shared"),
	}
	fields := map[string]string{}
	if form.Content == "" {
		fields["content"] = "Content is required."
	}

	var title *string
	if form.Title != "" {
		title = &form.Title
	}
	content := form.Content
	tags := parseTags(form.Tags)
	archived := form.Archived
	shared := form.Shared
	input := notes.PatchInput{
		ID:          noteID,
		OwnerUserID: ownerID,
		TitleSet:    true,
		Title:       title,
		ContentSet:  true,
		Content:     &content,
		TagsSet:     true,
		Tags:        &tags,
		ArchivedSet: true,
		Archived:    &archived,
		SharedSet:   true,
		Shared:      &shared,
	}
	return input, form, fields
}

func parseRenameTagForm(r *http.Request) (renameTagFormValues, map[string]string) {
	_ = r.ParseForm()
	form := renameTagFormValues{
		OldTag: strings.TrimSpace(r.FormValue("old_tag")),
		NewTag: strings.TrimSpace(r.FormValue("new_tag")),
	}
	fields := map[string]string{}
	if form.OldTag == "" {
		fields["old_tag"] = "Current tag is required."
	}
	if form.NewTag == "" {
		fields["new_tag"] = "New tag is required."
	}
	if form.OldTag != "" && form.OldTag == form.NewTag {
		fields["new_tag"] = "New tag must be different."
	}
	return form, fields
}

func parseTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func checkboxValue(r *http.Request, name string) bool {
	return strings.TrimSpace(r.FormValue(name)) != ""
}

func noteToForm(note notes.Note) noteFormValues {
	title := ""
	if note.Title != nil {
		title = strings.TrimSpace(*note.Title)
	}
	return noteFormValues{
		ID:       note.ID.String(),
		Title:    title,
		Content:  note.Content,
		Tags:     tagCSV(note.Tags),
		Archived: note.Archived,
		Shared:   note.Shared,
	}
}

func displayName(user auth.User) string {
	if user.DisplayName != nil && strings.TrimSpace(*user.DisplayName) != "" {
		return strings.TrimSpace(*user.DisplayName)
	}
	if user.Email != nil && strings.TrimSpace(*user.Email) != "" {
		return strings.TrimSpace(*user.Email)
	}
	return "there"
}

func noteTitle(note notes.Note) string {
	if note.Title != nil && strings.TrimSpace(*note.Title) != "" {
		return strings.TrimSpace(*note.Title)
	}
	return "Untitled note"
}

func joinTags(tags []string) string {
	if len(tags) == 0 {
		return "No tags yet"
	}
	return strings.Join(tags, " • ")
}

func tagCSV(tags []string) string {
	return strings.Join(tags, ", ")
}

func checkedAttr(value bool) string {
	if value {
		return "checked"
	}
	return ""
}

func formatTimestamp(value any) string {
	switch typed := value.(type) {
	case interface{ Format(string) string }:
		return typed.Format("Jan 2, 2006 3:04 PM MST")
	default:
		return ""
	}
}

func selectedNoteLink(note notes.Note, filterQuery string) string {
	values := url.Values{}
	values.Set("note", note.ID.String())
	return "/?" + combineQuery(values, filterQuery)
}

func editNoteLink(note notes.Note, filterQuery string) string {
	values := url.Values{}
	values.Set("note", note.ID.String())
	values.Set("edit", "1")
	return "/?" + combineQuery(values, filterQuery)
}

func tagFilterLink(tag, sort, order string) string {
	values := url.Values{}
	values.Set("tags", tag)
	values.Set("tag_mode", notes.TagMatchAny)
	if strings.TrimSpace(sort) != "" {
		values.Set("sort", sort)
	}
	if strings.TrimSpace(order) != "" {
		values.Set("order", order)
	}
	return "/?" + values.Encode()
}

func isHTMX(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("HX-Request")), "true")
}

func dict(values ...any) map[string]any {
	result := map[string]any{}
	for i := 0; i+1 < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			continue
		}
		result[key] = values[i+1]
	}
	return result
}

func buildFilterQuery(form noteListFormValues) string {
	values := url.Values{}
	if strings.TrimSpace(form.SavedQueryID) != "" {
		values.Set("saved_query_id", form.SavedQueryID)
	}
	if strings.TrimSpace(form.Search) != "" {
		values.Set("search", form.Search)
	}
	if strings.TrimSpace(form.SearchMode) != "" && form.SearchMode != notes.SearchModePlain {
		values.Set("search_mode", form.SearchMode)
	}
	if strings.TrimSpace(form.HasTitle) != "" {
		values.Set("has_title", form.HasTitle)
	}
	if strings.TrimSpace(form.Tags) != "" {
		values.Set("tags", form.Tags)
	}
	if strings.TrimSpace(form.TagCountMin) != "" {
		values.Set("tag_count_min", form.TagCountMin)
	}
	if strings.TrimSpace(form.TagCountMax) != "" {
		values.Set("tag_count_max", form.TagCountMax)
	}
	if strings.TrimSpace(form.Mode) != "" && form.Mode != notes.TagMatchAny {
		values.Set("tag_mode", form.Mode)
	}
	if strings.TrimSpace(form.Sort) != "" && form.Sort != "updated_at" {
		values.Set("sort", form.Sort)
	}
	if strings.TrimSpace(form.Order) != "" && form.Order != "desc" {
		values.Set("order", form.Order)
	}
	return values.Encode()
}

func combineQuery(values url.Values, encoded string) string {
	if encoded == "" {
		return values.Encode()
	}
	combined, err := url.ParseQuery(encoded)
	if err != nil {
		return values.Encode()
	}
	for key, items := range combined {
		for _, item := range items {
			values.Add(key, item)
		}
	}
	return values.Encode()
}

func renderMarkdown(source string) template.HTML {
	var buf bytes.Buffer
	// Goldmark blocks raw HTML unless html.WithUnsafe is enabled. That keeps the
	// teaching UI safer while still rendering normal Markdown formatting.
	if err := noteMarkdown.Convert([]byte(source), &buf); err != nil {
		return template.HTML(template.HTMLEscapeString(source))
	}
	return template.HTML(buf.String())
}

const uiTemplatesSource = `
{{define "landing"}}
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>go-notes</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <script src="https://unpkg.com/htmx.org@2.0.4"></script>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@400;500;600;700&family=Newsreader:opsz,wght@6..72,500;6..72,700&display=swap" rel="stylesheet">
  <script>
    tailwind.config = { theme: { extend: { fontFamily: { sans: ['IBM Plex Sans', 'sans-serif'], serif: ['Newsreader', 'serif'] } } } }
  </script>
</head>
<body class="min-h-screen bg-stone-950 text-stone-100">
  <main class="mx-auto flex min-h-screen max-w-7xl flex-col justify-center px-6 py-16">
    <div class="grid gap-8 lg:grid-cols-[1.15fr_0.85fr]">
      <section class="rounded-[2rem] border border-teal-400/20 bg-gradient-to-br from-stone-900 via-stone-900 to-teal-950/60 p-8 shadow-2xl shadow-teal-950/30 lg:p-10">
        <p class="mb-3 text-sm uppercase tracking-[0.3em] text-teal-300">go-notes</p>
        <h1 class="max-w-4xl font-serif text-5xl leading-tight text-stone-50 md:text-6xl">Your private workspace for notes that keep their context.</h1>
        <p class="mt-6 max-w-3xl text-lg leading-8 text-stone-300">Sign in to capture Markdown notes, organize them with tags, save useful searches, and come back to the same filtered workspace later. The service keeps your notes private by default, with explicit sharing when you choose to publish a note link.</p>
        <div class="mt-8 flex flex-wrap gap-4">
          <a href="/api/v1/auth/login?redirect_to=/" class="rounded-full bg-teal-300 px-6 py-3 text-sm font-semibold text-stone-950 transition hover:bg-teal-200">Continue with OIDC</a>
          <a href="/api/v1/healthz" class="rounded-full border border-stone-700 px-6 py-3 text-sm font-semibold text-stone-200 transition hover:border-stone-500 hover:text-stone-50">Check API health</a>
        </div>
        <div class="mt-10 grid gap-3 sm:grid-cols-2">
          <div class="rounded-2xl border border-stone-800 bg-stone-950/55 p-4">
            <p class="text-sm font-semibold text-teal-200">Write in Markdown</p>
            <p class="mt-2 text-sm leading-6 text-stone-400">Create and update notes with a focused editor, then read them with rendered Markdown formatting.</p>
          </div>
          <div class="rounded-2xl border border-stone-800 bg-stone-950/55 p-4">
            <p class="text-sm font-semibold text-teal-200">Find things again</p>
            <p class="mt-2 text-sm leading-6 text-stone-400">Search, filter by tags, sort your list, and save searches you use often.</p>
          </div>
          <div class="rounded-2xl border border-stone-800 bg-stone-950/55 p-4">
            <p class="text-sm font-semibold text-teal-200">Keep tags tidy</p>
            <p class="mt-2 text-sm leading-6 text-stone-400">Use tag browsing and rename workflows to keep related notes grouped together.</p>
          </div>
          <div class="rounded-2xl border border-stone-800 bg-stone-950/55 p-4">
            <p class="text-sm font-semibold text-teal-200">Share intentionally</p>
            <p class="mt-2 text-sm leading-6 text-stone-400">Notes stay owner-scoped unless you explicitly turn on sharing for a public link.</p>
          </div>
        </div>
      </section>
      <aside class="space-y-5">
        <section class="rounded-[2rem] border border-amber-300/20 bg-stone-900/80 p-8 shadow-xl shadow-black/20">
          <h2 class="font-serif text-3xl text-amber-100">What you can try here</h2>
          <ul class="mt-6 space-y-4 text-stone-300">
            <li class="rounded-2xl border border-stone-800 bg-stone-950/60 p-4">Sign in with your configured identity provider.</li>
            <li class="rounded-2xl border border-stone-800 bg-stone-950/60 p-4">Start a note immediately from the top of the workspace.</li>
            <li class="rounded-2xl border border-stone-800 bg-stone-950/60 p-4">Use saved queries and tag filters when your note list grows.</li>
          </ul>
        </section>
        <section class="rounded-[2rem] border border-stone-800 bg-stone-900/70 p-6">
          <h2 class="font-serif text-2xl text-stone-50">API and agent friendly</h2>
          <p class="mt-3 text-sm leading-6 text-stone-400">The same notes can also be reached through the JSON API and local MCP tools when those clients are configured for your deployment.</p>
        </section>
      </aside>
    </div>
  </main>
</body>
</html>
{{end}}

{{define "app_page"}}
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>go-notes workspace</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <script src="https://unpkg.com/htmx.org@2.0.4"></script>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@400;500;600;700&family=Newsreader:opsz,wght@6..72,500;6..72,700&display=swap" rel="stylesheet">
  <script>
    tailwind.config = { theme: { extend: { fontFamily: { sans: ['IBM Plex Sans', 'sans-serif'], serif: ['Newsreader', 'serif'] } } } }
  </script>
  <style>
    .note-markdown {
      color: rgb(231 229 228);
    }

    .note-markdown > * + * {
      margin-top: 1rem;
    }

    .note-markdown h1,
    .note-markdown h2,
    .note-markdown h3,
    .note-markdown h4 {
      font-family: "Newsreader", serif;
      color: rgb(250 250 249);
      letter-spacing: -0.02em;
      line-height: 1.1;
      margin-top: 2rem;
      margin-bottom: 0.9rem;
    }

    .note-markdown h1 {
      font-size: 2.4rem;
    }

    .note-markdown h2 {
      font-size: 2rem;
    }

    .note-markdown h3 {
      font-size: 1.55rem;
    }

    .note-markdown h4 {
      font-size: 1.2rem;
    }

    .note-markdown p,
    .note-markdown li {
      font-size: 1rem;
      line-height: 1.9;
      color: rgb(231 229 228);
    }

    .note-markdown a {
      color: rgb(94 234 212);
      text-decoration: underline;
      text-underline-offset: 0.22em;
    }

    .note-markdown strong {
      color: rgb(250 250 249);
      font-weight: 700;
    }

    .note-markdown em {
      color: rgb(253 230 138);
    }

    .note-markdown ul,
    .note-markdown ol {
      padding-left: 1.4rem;
    }

    .note-markdown ul {
      list-style: disc;
    }

    .note-markdown ol {
      list-style: decimal;
    }

    .note-markdown li + li {
      margin-top: 0.35rem;
    }

    .note-markdown blockquote {
      border-left: 4px solid rgb(45 212 191 / 0.55);
      background: rgb(12 10 9 / 0.55);
      color: rgb(214 211 209);
      border-radius: 0 1rem 1rem 0;
      padding: 1rem 1.25rem;
      margin: 1.5rem 0;
    }

    .note-markdown hr {
      border: 0;
      border-top: 1px solid rgb(68 64 60);
      margin: 2rem 0;
    }

    .note-markdown code {
      background: rgb(28 25 23);
      color: rgb(253 230 138);
      border: 1px solid rgb(68 64 60);
      border-radius: 0.6rem;
      padding: 0.15rem 0.4rem;
      font-size: 0.92em;
    }

    .note-markdown pre {
      background: rgb(12 10 9);
      border: 1px solid rgb(68 64 60);
      border-radius: 1rem;
      padding: 1rem 1.1rem;
      overflow-x: auto;
      box-shadow: inset 0 1px 0 rgb(255 255 255 / 0.02);
    }

    .note-markdown pre code {
      background: transparent;
      border: 0;
      color: rgb(231 229 228);
      padding: 0;
      border-radius: 0;
      font-size: 0.95rem;
    }

    .note-markdown table {
      width: 100%;
      border-collapse: collapse;
      overflow: hidden;
      border-radius: 1rem;
      border: 1px solid rgb(68 64 60);
      margin: 1.5rem 0;
    }

    .note-markdown thead {
      background: rgb(28 25 23);
    }

    .note-markdown th,
    .note-markdown td {
      border-bottom: 1px solid rgb(68 64 60);
      padding: 0.85rem 1rem;
      text-align: left;
      vertical-align: top;
    }

    .note-markdown th {
      color: rgb(250 250 249);
      font-weight: 600;
    }

    .note-markdown tbody tr:last-child td {
      border-bottom: 0;
    }
  </style>
</head>
<body class="min-h-screen bg-stone-950 text-stone-100">
  <div class="min-h-screen bg-[radial-gradient(circle_at_top_left,_rgba(45,212,191,0.18),_transparent_28%),radial-gradient(circle_at_bottom_right,_rgba(251,191,36,0.14),_transparent_30%)]">
    <header class="border-b border-stone-800/80 bg-stone-950/80 backdrop-blur">
      <div class="mx-auto flex max-w-7xl items-center justify-between px-6 py-4">
        <div>
          <p class="text-xs uppercase tracking-[0.35em] text-teal-300">go-notes</p>
          <h1 class="font-serif text-3xl text-stone-50">Notes workspace</h1>
        </div>
        <div class="flex items-center gap-3">
          <p class="text-sm text-stone-300">Signed in as <span class="font-semibold text-stone-50">{{displayName .Workspace.User}}</span></p>
          <form method="post" action="/app/logout">
            <button class="rounded-full border border-stone-700 px-4 py-2 text-sm font-semibold text-stone-200 transition hover:border-stone-500 hover:text-stone-50">Log out</button>
          </form>
        </div>
      </div>
    </header>
    <main class="mx-auto max-w-7xl px-6 py-8">
      {{template "workspace" .Workspace}}
    </main>
  </div>
</body>
</html>
{{end}}

{{define "workspace"}}
<section id="workspace" class="grid gap-6 xl:grid-cols-[22rem_1fr]">
  <aside class="space-y-6">
    <section class="rounded-[1.75rem] border border-stone-800 bg-stone-900/85 p-5 shadow-xl shadow-black/20">
      <div>
        <h2 class="font-serif text-2xl text-stone-50">New note</h2>
        <p class="mt-1 text-sm text-stone-400">Start here with a title, markdown body, and tags.</p>
      </div>
      <form class="mt-5 space-y-4" method="post" action="/app/notes" hx-post="/app/notes" hx-target="#workspace" hx-swap="outerHTML">
        <input type="hidden" name="ui_saved_query_id" value="{{.Filters.SavedQueryID}}">
        <input type="hidden" name="ui_search" value="{{.Filters.Search}}">
        <input type="hidden" name="ui_search_mode" value="{{.Filters.SearchMode}}">
        <input type="hidden" name="ui_has_title" value="{{.Filters.HasTitle}}">
        <input type="hidden" name="ui_tags" value="{{.Filters.Tags}}">
        <input type="hidden" name="ui_tag_count_min" value="{{.Filters.TagCountMin}}">
        <input type="hidden" name="ui_tag_count_max" value="{{.Filters.TagCountMax}}">
        <input type="hidden" name="ui_tag_mode" value="{{.Filters.Mode}}">
        <input type="hidden" name="ui_sort" value="{{.Filters.Sort}}">
        <input type="hidden" name="ui_order" value="{{.Filters.Order}}">
        <label class="block">
          <span class="mb-2 block text-sm font-medium text-stone-300">Title</span>
          <input class="w-full rounded-2xl border border-stone-700 bg-stone-950 px-4 py-3 text-stone-100 outline-none ring-0 transition focus:border-teal-300" type="text" name="title" value="{{.CreateForm.Title}}" placeholder="Sprint review ideas">
        </label>
        <label class="block">
          <span class="mb-2 block text-sm font-medium text-stone-300">Content</span>
          <textarea class="min-h-[8rem] w-full rounded-2xl border border-stone-700 bg-stone-950 px-4 py-3 text-stone-100 outline-none transition focus:border-teal-300" name="content" placeholder="Write the body of the note here...">{{.CreateForm.Content}}</textarea>
          {{with index .CreateErrors "content"}}<p class="mt-2 text-sm text-rose-300">{{.}}</p>{{end}}
        </label>
        <label class="block">
          <span class="mb-2 block text-sm font-medium text-stone-300">Tags</span>
          <input class="w-full rounded-2xl border border-stone-700 bg-stone-950 px-4 py-3 text-stone-100 outline-none transition focus:border-teal-300" type="text" name="tags" value="{{.CreateForm.Tags}}" placeholder="planning, retro, ideas">
        </label>
        <div class="flex flex-wrap gap-4 text-sm text-stone-300">
          <label class="inline-flex items-center gap-2">
            <input class="accent-teal-300" type="checkbox" name="shared" value="true" {{checkedAttr .CreateForm.Shared}}>
            Shared
          </label>
          <label class="inline-flex items-center gap-2">
            <input class="accent-amber-300" type="checkbox" name="archived" value="true" {{checkedAttr .CreateForm.Archived}}>
            Archived
          </label>
        </div>
        <button class="w-full rounded-full bg-teal-300 px-5 py-3 text-sm font-semibold text-stone-950 transition hover:bg-teal-200">Create note</button>
      </form>
    </section>

    <section class="rounded-[1.75rem] border border-stone-800 bg-stone-900/85 p-5 shadow-xl shadow-black/20">
      <div>
        <h2 class="font-serif text-2xl text-stone-50">Saved queries</h2>
        <p class="mt-1 text-sm text-stone-400">Save the current filters so you can reuse them in the UI, API, or MCP.</p>
      </div>
      <form class="mt-5 space-y-4" method="post" action="/app/saved-queries" hx-post="/app/saved-queries" hx-target="#workspace" hx-swap="outerHTML">
        <input type="hidden" name="ui_saved_query_id" value="{{.Filters.SavedQueryID}}">
        <input type="hidden" name="ui_search" value="{{.Filters.Search}}">
        <input type="hidden" name="ui_search_mode" value="{{.Filters.SearchMode}}">
        <input type="hidden" name="ui_has_title" value="{{.Filters.HasTitle}}">
        <input type="hidden" name="ui_tags" value="{{.Filters.Tags}}">
        <input type="hidden" name="ui_tag_count_min" value="{{.Filters.TagCountMin}}">
        <input type="hidden" name="ui_tag_count_max" value="{{.Filters.TagCountMax}}">
        <input type="hidden" name="ui_tag_mode" value="{{.Filters.Mode}}">
        <input type="hidden" name="ui_sort" value="{{.Filters.Sort}}">
        <input type="hidden" name="ui_order" value="{{.Filters.Order}}">
        <label class="block">
          <span class="mb-2 block text-sm font-medium text-stone-300">Name this filter set</span>
          <input class="w-full rounded-2xl border border-stone-700 bg-stone-950 px-4 py-3 text-stone-100 outline-none transition focus:border-fuchsia-300" type="text" name="saved_query_name" value="{{.SaveQuery.Name}}" placeholder="Recently shared work notes">
          {{with index .SaveErrors "name"}}<p class="mt-2 text-sm text-rose-300">{{.}}</p>{{end}}
        </label>
        <button class="w-full rounded-full bg-fuchsia-300 px-5 py-3 text-sm font-semibold text-stone-950 transition hover:bg-fuchsia-200">Save current filters</button>
      </form>
      <div class="mt-5 space-y-3">
        {{if .SavedQueries}}
          {{range .SavedQueries}}
            <div class="rounded-2xl border border-stone-800 bg-stone-950/70 px-4 py-3">
              <div class="flex items-start justify-between gap-3">
                <a href="/?saved_query_id={{.ID}}" class="font-semibold text-stone-100 transition hover:text-fuchsia-200">{{.Name}}</a>
                <form method="post" action="/app/saved-queries/{{.ID}}/delete">
                  <input type="hidden" name="return_query" value="{{$.FilterQuery}}">
                  <button class="text-xs uppercase tracking-[0.2em] text-stone-500 transition hover:text-rose-300">Delete</button>
                </form>
              </div>
              <p class="mt-2 text-xs text-stone-500">{{if .Query}}{{.Query}}{{else}}Default active-note query{{end}}</p>
            </div>
          {{end}}
        {{else}}
          <div class="rounded-2xl border border-dashed border-stone-700 px-4 py-5 text-sm text-stone-400">Save your current filters to reuse them later from the API, UI, or MCP.</div>
        {{end}}
      </div>
    </section>

    <section class="rounded-[1.75rem] border border-stone-800 bg-stone-900/85 p-5 shadow-xl shadow-black/20">
      <div>
        <h2 class="font-serif text-2xl text-stone-50">Filters</h2>
        <p class="mt-1 text-sm text-stone-400">Search, sort, and narrow the note list without leaving the workspace.</p>
      </div>
      <form class="mt-5 space-y-4" method="get" action="/">
        <input type="hidden" name="saved_query_id" value="{{.Filters.SavedQueryID}}">
        <label class="block">
          <span class="mb-2 block text-sm font-medium text-stone-300">Search</span>
          <input class="w-full rounded-2xl border border-stone-700 bg-stone-950 px-4 py-3 text-stone-100 outline-none transition focus:border-amber-300" type="text" name="search" value="{{.Filters.Search}}" placeholder="golang notes, phrase match, minus exclude">
          {{with index .FilterErrors "search"}}<p class="mt-2 text-sm text-rose-300">{{.}}</p>{{end}}
        </label>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="block">
            <span class="mb-2 block text-sm font-medium text-stone-300">Search mode</span>
            <select class="w-full rounded-2xl border border-stone-700 bg-stone-950 px-4 py-3 text-stone-100 outline-none transition focus:border-amber-300" name="search_mode">
              <option value="plain" {{if eq .Filters.SearchMode "plain"}}selected{{end}}>Plain contains</option>
              <option value="fts" {{if eq .Filters.SearchMode "fts"}}selected{{end}}>Full-text ranked</option>
            </select>
            {{with index .FilterErrors "search_mode"}}<p class="mt-2 text-sm text-rose-300">{{.}}</p>{{end}}
          </label>
          <label class="block">
            <span class="mb-2 block text-sm font-medium text-stone-300">Has title</span>
            <select class="w-full rounded-2xl border border-stone-700 bg-stone-950 px-4 py-3 text-stone-100 outline-none transition focus:border-amber-300" name="has_title">
              <option value="" {{if eq .Filters.HasTitle ""}}selected{{end}}>Either</option>
              <option value="true" {{if eq .Filters.HasTitle "true"}}selected{{end}}>Titled only</option>
              <option value="false" {{if eq .Filters.HasTitle "false"}}selected{{end}}>Untitled only</option>
            </select>
            {{with index .FilterErrors "has_title"}}<p class="mt-2 text-sm text-rose-300">{{.}}</p>{{end}}
          </label>
        </div>
        <label class="block">
          <span class="mb-2 block text-sm font-medium text-stone-300">Tags</span>
          <input class="w-full rounded-2xl border border-stone-700 bg-stone-950 px-4 py-3 text-stone-100 outline-none transition focus:border-amber-300" type="text" name="tags" value="{{.Filters.Tags}}" placeholder="work, planning">
          {{with index .FilterErrors "tag_mode"}}<p class="mt-2 text-sm text-rose-300">{{.}}</p>{{end}}
        </label>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="block">
            <span class="mb-2 block text-sm font-medium text-stone-300">Tag mode</span>
            <select class="w-full rounded-2xl border border-stone-700 bg-stone-950 px-4 py-3 text-stone-100 outline-none transition focus:border-amber-300" name="tag_mode">
              <option value="any" {{if eq .Filters.Mode "any"}}selected{{end}}>Any tag</option>
              <option value="all" {{if eq .Filters.Mode "all"}}selected{{end}}>All tags</option>
            </select>
          </label>
          <label class="block">
            <span class="mb-2 block text-sm font-medium text-stone-300">Sort</span>
            <select class="w-full rounded-2xl border border-stone-700 bg-stone-950 px-4 py-3 text-stone-100 outline-none transition focus:border-amber-300" name="sort">
              <option value="updated_at" {{if eq .Filters.Sort "updated_at"}}selected{{end}}>Recently updated</option>
              <option value="created_at" {{if eq .Filters.Sort "created_at"}}selected{{end}}>Recently created</option>
              <option value="title" {{if eq .Filters.Sort "title"}}selected{{end}}>Title</option>
              <option value="tag_count" {{if eq .Filters.Sort "tag_count"}}selected{{end}}>Tag count</option>
              <option value="primary_tag" {{if eq .Filters.Sort "primary_tag"}}selected{{end}}>Primary tag</option>
              <option value="relevance" {{if eq .Filters.Sort "relevance"}}selected{{end}}>Relevance</option>
            </select>
            {{with index .FilterErrors "sort"}}<p class="mt-2 text-sm text-rose-300">{{.}}</p>{{end}}
          </label>
        </div>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="block">
            <span class="mb-2 block text-sm font-medium text-stone-300">Min tag count</span>
            <input class="w-full rounded-2xl border border-stone-700 bg-stone-950 px-4 py-3 text-stone-100 outline-none transition focus:border-amber-300" type="number" min="0" name="tag_count_min" value="{{.Filters.TagCountMin}}" placeholder="0">
            {{with index .FilterErrors "tag_count_min"}}<p class="mt-2 text-sm text-rose-300">{{.}}</p>{{end}}
          </label>
          <label class="block">
            <span class="mb-2 block text-sm font-medium text-stone-300">Max tag count</span>
            <input class="w-full rounded-2xl border border-stone-700 bg-stone-950 px-4 py-3 text-stone-100 outline-none transition focus:border-amber-300" type="number" min="0" name="tag_count_max" value="{{.Filters.TagCountMax}}" placeholder="8">
            {{with index .FilterErrors "tag_count_max"}}<p class="mt-2 text-sm text-rose-300">{{.}}</p>{{end}}
          </label>
        </div>
        <label class="block">
          <span class="mb-2 block text-sm font-medium text-stone-300">Order</span>
          <select class="w-full rounded-2xl border border-stone-700 bg-stone-950 px-4 py-3 text-stone-100 outline-none transition focus:border-amber-300" name="order">
            <option value="desc" {{if eq .Filters.Order "desc"}}selected{{end}}>Descending</option>
            <option value="asc" {{if eq .Filters.Order "asc"}}selected{{end}}>Ascending</option>
          </select>
          {{with index .FilterErrors "order"}}<p class="mt-2 text-sm text-rose-300">{{.}}</p>{{end}}
        </label>
        <div class="flex gap-3">
          <button class="flex-1 rounded-full bg-amber-300 px-5 py-3 text-sm font-semibold text-stone-950 transition hover:bg-amber-200">Apply filters</button>
          <a href="/" class="rounded-full border border-stone-700 px-5 py-3 text-sm font-semibold text-stone-200 transition hover:border-stone-500 hover:text-stone-50">Reset</a>
        </div>
      </form>
      <div class="mt-6 border-t border-stone-800 pt-6">
        <h3 class="font-serif text-xl text-stone-100">Rename tag</h3>
        <p class="mt-1 text-sm text-stone-400">Rewrite one owner-scoped tag across matching notes using the same SQL-backed rules as the API and MCP tools.</p>
        <form class="mt-4 space-y-4" method="post" action="/app/tags/rename" hx-post="/app/tags/rename" hx-target="#workspace" hx-swap="outerHTML">
          <input type="hidden" name="ui_saved_query_id" value="{{.Filters.SavedQueryID}}">
          <input type="hidden" name="ui_search" value="{{.Filters.Search}}">
          <input type="hidden" name="ui_search_mode" value="{{.Filters.SearchMode}}">
          <input type="hidden" name="ui_has_title" value="{{.Filters.HasTitle}}">
          <input type="hidden" name="ui_tags" value="{{.Filters.Tags}}">
          <input type="hidden" name="ui_tag_count_min" value="{{.Filters.TagCountMin}}">
          <input type="hidden" name="ui_tag_count_max" value="{{.Filters.TagCountMax}}">
          <input type="hidden" name="ui_tag_mode" value="{{.Filters.Mode}}">
          <input type="hidden" name="ui_sort" value="{{.Filters.Sort}}">
          <input type="hidden" name="ui_order" value="{{.Filters.Order}}">
          <div class="grid gap-4 sm:grid-cols-2">
            <label class="block">
              <span class="mb-2 block text-sm font-medium text-stone-300">Current tag</span>
              <input class="w-full rounded-2xl border border-stone-700 bg-stone-950 px-4 py-3 text-stone-100 outline-none transition focus:border-amber-300" type="text" name="old_tag" value="{{.RenameTag.OldTag}}" placeholder="planning">
              {{with index .RenameErrors "old_tag"}}<p class="mt-2 text-sm text-rose-300">{{.}}</p>{{end}}
            </label>
            <label class="block">
              <span class="mb-2 block text-sm font-medium text-stone-300">New tag</span>
              <input class="w-full rounded-2xl border border-stone-700 bg-stone-950 px-4 py-3 text-stone-100 outline-none transition focus:border-amber-300" type="text" name="new_tag" value="{{.RenameTag.NewTag}}" placeholder="roadmap">
              {{with index .RenameErrors "new_tag"}}<p class="mt-2 text-sm text-rose-300">{{.}}</p>{{end}}
            </label>
          </div>
          <button class="rounded-full border border-amber-300/40 px-5 py-3 text-sm font-semibold text-amber-100 transition hover:border-amber-200 hover:text-amber-50">Rename tag</button>
        </form>
      </div>
    </section>

    <section class="rounded-[1.75rem] border border-stone-800 bg-stone-900/85 p-5 shadow-xl shadow-black/20">
      <div class="flex items-center justify-between">
        <h2 class="font-serif text-2xl text-stone-50">Recent notes</h2>
        <span class="text-sm text-stone-400">{{len .Notes}} total</span>
      </div>
      <div class="mt-4 space-y-3">
        {{if .Notes}}
          {{range .Notes}}
            <a href="{{selectedNoteLink . $.FilterQuery}}" hx-get="/app/notes/{{.ID}}?{{$.FilterQuery}}" hx-target="#note-panel" hx-swap="innerHTML" class="block rounded-2xl border border-stone-800 bg-stone-950/70 px-4 py-3 transition hover:border-teal-300/40 hover:bg-stone-950">
              <div class="flex items-start justify-between gap-3">
                <div>
                  <p class="font-semibold text-stone-100">{{noteTitle .}}</p>
                  <div class="mt-2 flex flex-wrap gap-2">
                    {{if .Tags}}
                      {{range .Tags}}
                        <span class="rounded-full border border-stone-700 px-2.5 py-1 text-xs text-stone-300">{{.}}</span>
                      {{end}}
                    {{else}}
                      <span class="text-sm text-stone-500">No tags yet</span>
                    {{end}}
                  </div>
                </div>
                <span class="text-xs uppercase tracking-[0.2em] text-stone-500">{{if .Shared}}shared{{else}}private{{end}}</span>
              </div>
            </a>
          {{end}}
        {{else}}
          <div class="rounded-2xl border border-dashed border-stone-700 px-4 py-8 text-center text-stone-400">No notes yet. Create the first one from the form above.</div>
        {{end}}
      </div>
    </section>
  </aside>

  <section id="note-panel" class="rounded-[2rem] border border-stone-800 bg-stone-900/85 p-6 shadow-xl shadow-black/20">
    {{if .SelectedNote}}
      {{if .Editing}}
        {{template "note_edit" (dict "Form" .EditForm "Errors" .EditErrors "FilterQuery" .FilterQuery "Filters" .Filters)}}
      {{else}}
        {{template "note_detail" (dict "Note" .SelectedNote "Filters" .Filters "FilterQuery" .FilterQuery)}}
      {{end}}
    {{else}}
      <div class="flex h-full min-h-[24rem] flex-col items-center justify-center rounded-[1.5rem] border border-dashed border-stone-700 bg-stone-950/60 text-center">
        <p class="font-serif text-3xl text-stone-50">Pick a note to read it here.</p>
        <p class="mt-3 max-w-xl text-stone-400">This panel is intentionally HTMX-driven so the project can demonstrate small, focused HTML updates without giving up the server-rendered workflow.</p>
      </div>
    {{end}}
  </section>
</section>
{{end}}

{{define "note_detail"}}
<article class="space-y-6">
  <header class="border-b border-stone-800 pb-5">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <p class="text-xs uppercase tracking-[0.3em] text-teal-300">Selected note</p>
        <h2 class="mt-2 font-serif text-4xl text-stone-50">{{noteTitle .Note}}</h2>
      </div>
      <a href="{{editNoteLink .Note .FilterQuery}}" hx-get="/app/notes/{{.Note.ID}}/edit?{{.FilterQuery}}" hx-target="#note-panel" hx-swap="innerHTML" class="rounded-full border border-stone-700 px-4 py-2 text-sm font-semibold text-stone-200 transition hover:border-teal-300 hover:text-stone-50">Edit</a>
    </div>
    <div class="mt-4 flex flex-wrap gap-3 text-sm text-stone-400">
      <span>Updated {{formatTimestamp .Note.UpdatedAt}}</span>
      <span>•</span>
      <span class="flex flex-wrap gap-2">
        {{if .Note.Tags}}
          {{range .Note.Tags}}
            <a href="{{tagFilterLink . $.Filters.Sort $.Filters.Order}}" class="rounded-full border border-teal-500/30 bg-teal-500/10 px-2.5 py-1 text-xs text-teal-100 transition hover:border-teal-300 hover:text-teal-50">{{.}}</a>
          {{end}}
        {{else}}
          <span>No tags yet</span>
        {{end}}
      </span>
      <span>•</span>
      <span>{{if .Note.Shared}}Shared{{else}}Private{{end}}</span>
      <span>•</span>
      <span>{{if .Note.Archived}}Archived{{else}}Active{{end}}</span>
    </div>
    {{if .Note.Shared}}
      {{with .Note.ShareSlug}}
        <div class="mt-4 rounded-2xl border border-teal-500/20 bg-teal-500/10 px-4 py-3 text-sm text-teal-50">
          Public link:
          <a class="font-semibold underline underline-offset-4 transition hover:text-teal-100" href="/shared/{{.}}">/shared/{{.}}</a>
        </div>
      {{end}}
    {{end}}
  </header>
  <div class="note-markdown max-w-none">{{renderMarkdown .Note.Content}}</div>
</article>
{{end}}

{{define "note_edit"}}
<section>
  <div class="flex items-center justify-between gap-4 border-b border-stone-800 pb-5">
    <div>
      <p class="text-xs uppercase tracking-[0.3em] text-amber-300">Editing</p>
      <h2 class="mt-2 font-serif text-4xl text-stone-50">Revise note</h2>
    </div>
    <a href="/?{{.FilterQuery}}&note={{.Form.ID}}" hx-get="/app/notes/{{.Form.ID}}?{{.FilterQuery}}" hx-target="#note-panel" hx-swap="innerHTML" class="rounded-full border border-stone-700 px-4 py-2 text-sm font-semibold text-stone-200 transition hover:border-stone-500 hover:text-stone-50">Cancel</a>
  </div>
  <form class="mt-6 space-y-4" method="post" action="/app/notes/{{.Form.ID}}" hx-post="/app/notes/{{.Form.ID}}" hx-target="#workspace" hx-swap="outerHTML">
    <input type="hidden" name="ui_saved_query_id" value="{{.Filters.SavedQueryID}}">
    <input type="hidden" name="ui_tags" value="{{.Filters.Tags}}">
    <input type="hidden" name="ui_search" value="{{.Filters.Search}}">
    <input type="hidden" name="ui_search_mode" value="{{.Filters.SearchMode}}">
    <input type="hidden" name="ui_has_title" value="{{.Filters.HasTitle}}">
    <input type="hidden" name="ui_tag_count_min" value="{{.Filters.TagCountMin}}">
    <input type="hidden" name="ui_tag_count_max" value="{{.Filters.TagCountMax}}">
    <input type="hidden" name="ui_tag_mode" value="{{.Filters.Mode}}">
    <input type="hidden" name="ui_sort" value="{{.Filters.Sort}}">
    <input type="hidden" name="ui_order" value="{{.Filters.Order}}">
    <label class="block">
      <span class="mb-2 block text-sm font-medium text-stone-300">Title</span>
      <input class="w-full rounded-2xl border border-stone-700 bg-stone-950 px-4 py-3 text-stone-100 outline-none transition focus:border-amber-300" type="text" name="title" value="{{.Form.Title}}">
    </label>
    <label class="block">
      <span class="mb-2 block text-sm font-medium text-stone-300">Content</span>
      <textarea class="min-h-[12rem] w-full rounded-2xl border border-stone-700 bg-stone-950 px-4 py-3 text-stone-100 outline-none transition focus:border-amber-300" name="content">{{.Form.Content}}</textarea>
      {{with index .Errors "content"}}<p class="mt-2 text-sm text-rose-300">{{.}}</p>{{end}}
    </label>
    <label class="block">
      <span class="mb-2 block text-sm font-medium text-stone-300">Tags</span>
      <input class="w-full rounded-2xl border border-stone-700 bg-stone-950 px-4 py-3 text-stone-100 outline-none transition focus:border-amber-300" type="text" name="tags" value="{{.Form.Tags}}">
    </label>
    <div class="flex flex-wrap gap-4 text-sm text-stone-300">
      <label class="inline-flex items-center gap-2">
        <input class="accent-teal-300" type="checkbox" name="shared" value="true" {{checkedAttr .Form.Shared}}>
        Shared
      </label>
      <label class="inline-flex items-center gap-2">
        <input class="accent-amber-300" type="checkbox" name="archived" value="true" {{checkedAttr .Form.Archived}}>
        Archived
      </label>
    </div>
    <button class="rounded-full bg-amber-300 px-5 py-3 text-sm font-semibold text-stone-950 transition hover:bg-amber-200">Save changes</button>
  </form>
</section>
{{end}}

{{define "note_error"}}
<div class="rounded-[1.5rem] border border-rose-400/30 bg-rose-950/40 p-6 text-rose-100">
  <h2 class="font-serif text-3xl">Unable to load note</h2>
  <p class="mt-3 text-sm text-rose-200">{{.Message}}</p>
</div>
{{end}}

{{define "shared_note_page"}}
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{noteTitle .}} · go-notes</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@400;500;600;700&family=Newsreader:opsz,wght@6..72,500;6..72,700&display=swap" rel="stylesheet">
  <script>
    tailwind.config = { theme: { extend: { fontFamily: { sans: ['IBM Plex Sans', 'sans-serif'], serif: ['Newsreader', 'serif'] } } } }
  </script>
  <style>
    .note-markdown > * + * { margin-top: 1rem; }
    .note-markdown h1, .note-markdown h2, .note-markdown h3, .note-markdown h4 {
      font-family: "Newsreader", serif;
      color: rgb(250 250 249);
      letter-spacing: -0.02em;
      line-height: 1.1;
      margin-top: 2rem;
      margin-bottom: 0.9rem;
    }
    .note-markdown h1 { font-size: 2.4rem; }
    .note-markdown h2 { font-size: 2rem; }
    .note-markdown h3 { font-size: 1.55rem; }
    .note-markdown h4 { font-size: 1.2rem; }
    .note-markdown p, .note-markdown li {
      font-size: 1rem;
      line-height: 1.9;
      color: rgb(231 229 228);
    }
    .note-markdown a {
      color: rgb(94 234 212);
      text-decoration: underline;
      text-underline-offset: 0.22em;
    }
    .note-markdown strong { color: rgb(250 250 249); font-weight: 700; }
    .note-markdown em { color: rgb(253 230 138); }
    .note-markdown ul, .note-markdown ol { padding-left: 1.4rem; }
    .note-markdown ul { list-style: disc; }
    .note-markdown ol { list-style: decimal; }
    .note-markdown li + li { margin-top: 0.35rem; }
    .note-markdown blockquote {
      border-left: 4px solid rgb(45 212 191 / 0.55);
      background: rgb(12 10 9 / 0.55);
      color: rgb(214 211 209);
      border-radius: 0 1rem 1rem 0;
      padding: 1rem 1.25rem;
      margin: 1.5rem 0;
    }
    .note-markdown code {
      background: rgb(28 25 23);
      color: rgb(253 230 138);
      border: 1px solid rgb(68 64 60);
      border-radius: 0.6rem;
      padding: 0.15rem 0.4rem;
      font-size: 0.92em;
    }
    .note-markdown pre {
      background: rgb(12 10 9);
      border: 1px solid rgb(68 64 60);
      border-radius: 1rem;
      padding: 1rem 1.1rem;
      overflow-x: auto;
    }
    .note-markdown pre code {
      background: transparent;
      border: 0;
      color: rgb(231 229 228);
      padding: 0;
    }
  </style>
</head>
<body class="min-h-screen bg-stone-950 text-stone-100">
  <main class="mx-auto max-w-4xl px-6 py-12">
    <article class="rounded-[2rem] border border-teal-400/20 bg-stone-900/85 p-6 shadow-2xl shadow-teal-950/30 md:p-10">
      <header class="border-b border-stone-800 pb-6">
        <p class="text-xs uppercase tracking-[0.35em] text-teal-300">Shared note</p>
        <h1 class="mt-3 font-serif text-5xl leading-tight text-stone-50">{{noteTitle .}}</h1>
        <div class="mt-5 flex flex-wrap gap-3 text-sm text-stone-400">
          <span>Updated {{formatTimestamp .UpdatedAt}}</span>
          {{if .Tags}}
            <span>•</span>
            <span class="flex flex-wrap gap-2">
              {{range .Tags}}
                <span class="rounded-full border border-teal-500/30 bg-teal-500/10 px-2.5 py-1 text-xs text-teal-100">{{.}}</span>
              {{end}}
            </span>
          {{end}}
        </div>
      </header>
      <div class="note-markdown mt-8 max-w-none">{{renderMarkdown .Content}}</div>
    </article>
    <p class="mt-6 text-center text-sm text-stone-500">This note was intentionally shared by its owner.</p>
  </main>
</body>
</html>
{{end}}

{{define "shared_note_error"}}
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}} · go-notes</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@400;500;600;700&family=Newsreader:opsz,wght@6..72,500;6..72,700&display=swap" rel="stylesheet">
  <script>
    tailwind.config = { theme: { extend: { fontFamily: { sans: ['IBM Plex Sans', 'sans-serif'], serif: ['Newsreader', 'serif'] } } } }
  </script>
</head>
<body class="min-h-screen bg-stone-950 text-stone-100">
  <main class="mx-auto flex min-h-screen max-w-3xl flex-col justify-center px-6 py-16 text-center">
    <section class="rounded-[2rem] border border-rose-400/25 bg-rose-950/30 p-8 shadow-2xl shadow-black/20">
      <p class="text-xs uppercase tracking-[0.35em] text-rose-200">go-notes</p>
      <h1 class="mt-3 font-serif text-5xl text-stone-50">{{.Title}}</h1>
      <p class="mt-5 text-lg leading-8 text-rose-100">{{.Message}}</p>
      <a href="/" class="mt-8 inline-flex rounded-full border border-stone-700 px-5 py-3 text-sm font-semibold text-stone-200 transition hover:border-stone-500 hover:text-stone-50">Go to sign in</a>
    </section>
  </main>
</body>
</html>
{{end}}
`
