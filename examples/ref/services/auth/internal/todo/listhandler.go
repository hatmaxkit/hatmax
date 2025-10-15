package todo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/adrianpk/hatmax-ref/pkg/lib/core"
	"github.com/adrianpk/hatmax-ref/services/auth/internal/config"
)

const ListMaxBodyBytes = 1 << 20

// NewListHandler creates a new ListHandler for the aggregate root.
func NewListHandler(repo ListRepo, xparams config.XParams) *ListHandler {
	return &ListHandler{
		repo:    repo,
		xparams: xparams,
	}
}

type ListHandler struct {
	repo    ListRepo
	xparams config.XParams
}

func (h *ListHandler) RegisterRoutes(r chi.Router) {
	r.Route("/lists", func(r chi.Router) {
		r.Post("/", h.CreateList)
		r.Get("/", h.GetAllLists)
		r.Get("/{id}", h.GetList)
		r.Put("/{id}", h.UpdateList)
		r.Delete("/{id}", h.DeleteList)

		// Item operations (part of the aggregate)
		r.Post("/{id}/items", h.AddItemToList)
		r.Put("/{id}/items/{childId}", h.UpdateItemInList)
		r.Delete("/{id}/items/{childId}", h.RemoveItemFromList)

		// Tag operations (part of the aggregate)
		r.Post("/{id}/tags", h.AddTagToList)
		r.Put("/{id}/tags/{childId}", h.UpdateTagInList)
		r.Delete("/{id}/tags/{childId}", h.RemoveTagFromList)

	})
}

func (h *ListHandler) CreateList(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	list, ok := h.decodeListPayload(w, r, log)
	if !ok {
		return
	}

	list.EnsureID()
	list.BeforeCreate()

	validationErrors := ValidateCreateList(ctx, list)
	if len(validationErrors) > 0 {
		log.Debug("validation failed", "errors", validationErrors)
		core.RespondError(w, http.StatusBadRequest, "Validation failed")
		return
	}

	if err := h.repo.Create(ctx, &list); err != nil {
		log.Error("cannot create list", "error", err)
		core.RespondError(w, http.StatusInternalServerError, "Could not create list")
		return
	}

	// Standard links
	links := core.RESTfulLinksFor(&list)

	// Child collection links
	links = append(links, core.Link{
		Rel:  "items",
		Href: fmt.Sprintf("/lists/%s/items", list.ID),
	})

	// Child collection links
	links = append(links, core.Link{
		Rel:  "tags",
		Href: fmt.Sprintf("/lists/%s/tags", list.ID),
	})

	w.WriteHeader(http.StatusCreated)
	core.RespondSuccess(w, list, links...)
}

func (h *ListHandler) GetList(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	list, err := h.repo.Get(ctx, id)
	if err != nil {
		log.Error("error loading list", "error", err, "id", id.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not retrieve list")
		return
	}

	if list == nil {
		core.RespondError(w, http.StatusNotFound, "List not found")
		return
	}

	// Standard links
	links := core.RESTfulLinksFor(list)

	// Child collection links
	links = append(links, core.Link{
		Rel:  "items",
		Href: fmt.Sprintf("/lists/%s/items", list.ID),
	})

	// Child collection links
	links = append(links, core.Link{
		Rel:  "tags",
		Href: fmt.Sprintf("/lists/%s/tags", list.ID),
	})

	// Child links
	for _, item := range list.Items {
		childLinks := core.ChildLinksFor(list, &item)
		// Child entity link
		links = append(links, core.Link{
			Rel:  "item",
			Href: childLinks[0].Href,
		})
	}

	// Child links
	for _, tag := range list.Tags {
		childLinks := core.ChildLinksFor(list, &tag)
		// Child entity link
		links = append(links, core.Link{
			Rel:  "tag",
			Href: childLinks[0].Href,
		})
	}

	core.RespondSuccess(w, list, links...)
}

func (h *ListHandler) GetAllLists(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	lists, err := h.repo.List(ctx)
	if err != nil {
		log.Error("error retrieving lists", "error", err)
		core.RespondError(w, http.StatusInternalServerError, "Could not list all lists")
		return
	}

	// Collection response
	core.RespondCollection(w, lists, "list")
}

func (h *ListHandler) UpdateList(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	list, ok := h.decodeListPayload(w, r, log)
	if !ok {
		return
	}

	list.SetID(id)
	list.BeforeUpdate()

	validationErrors := ValidateUpdateList(ctx, id, list)
	if len(validationErrors) > 0 {
		log.Debug("validation failed", "errors", validationErrors, "id", id.String())
		core.RespondError(w, http.StatusBadRequest, "Validation failed")
		return
	}

	if err := h.repo.Save(ctx, &list); err != nil {
		log.Error("cannot save list", "error", err, "id", id.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not update list")
		return
	}

	// Standard links
	links := core.RESTfulLinksFor(&list)

	// Child collection links
	links = append(links, core.Link{
		Rel:  "items",
		Href: fmt.Sprintf("/lists/%s/items", list.ID),
	})

	// Child collection links
	links = append(links, core.Link{
		Rel:  "tags",
		Href: fmt.Sprintf("/lists/%s/tags", list.ID),
	})

	core.RespondSuccess(w, list, links...)
}

func (h *ListHandler) DeleteList(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	validationErrors := ValidateDeleteList(ctx, id)
	if len(validationErrors) > 0 {
		log.Debug("validation failed", "errors", validationErrors, "id", id.String())
		core.RespondError(w, http.StatusBadRequest, "Validation failed")
		return
	}

	if err := h.repo.Delete(ctx, id); err != nil {
		log.Error("error deleting list", "error", err, "id", id.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not delete list")
		return
	}

	// Post-deletion links
	links := core.CollectionLinksFor("list")
	w.WriteHeader(http.StatusNoContent)
	core.RespondSuccess(w, nil, links...)
}

// Child entity operations (Items)
func (h *ListHandler) AddItemToList(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	listID, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	item, ok := h.decodeItemPayload(w, r, log)
	if !ok {
		return
	}

	// Load the aggregate
	list, err := h.repo.Get(ctx, listID)
	if err != nil {
		log.Error("cannot load list for adding item", "error", err, "listId", listID.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not retrieve list")
		return
	}

	if list == nil {
		core.RespondError(w, http.StatusNotFound, "List not found")
		return
	}

	// Add item to aggregate
	item.EnsureID()
	item.BeforeCreate()
	list.Items = append(list.Items, item)

	// Save the entire aggregate
	if err := h.repo.Save(ctx, list); err != nil {
		log.Error("error saving list with new item", "error", err, "listId", listID.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not add item to list")
		return
	}

	// Child response
	w.WriteHeader(http.StatusCreated)
	core.RespondChild(w, list, &item)
}

func (h *ListHandler) UpdateItemInList(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	listID, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	itemID, ok := h.parseItemIDParam(w, r, log)
	if !ok {
		return
	}

	item, ok := h.decodeItemPayload(w, r, log)
	if !ok {
		return
	}

	// Load the aggregate
	list, err := h.repo.Get(ctx, listID)
	if err != nil {
		log.Error("cannot load list for updating item", "error", err, "listId", listID.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not retrieve list")
		return
	}

	if list == nil {
		core.RespondError(w, http.StatusNotFound, "List not found")
		return
	}

	// Find and update item in aggregate
	found := false
	for i, existingItem := range list.Items {
		if existingItem.ID == itemID {
			item.SetID(itemID)
			item.BeforeUpdate()
			list.Items[i] = item
			found = true
			break
		}
	}

	if !found {
		core.RespondError(w, http.StatusNotFound, "Item not found in list")
		return
	}

	// Save the entire aggregate
	if err := h.repo.Save(ctx, list); err != nil {
		log.Error("error saving list with updated item", "error", err, "listId", listID.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not update item in list")
		return
	}

	// Child response
	core.RespondChild(w, list, &item)
}

func (h *ListHandler) RemoveItemFromList(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	listID, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	itemID, ok := h.parseItemIDParam(w, r, log)
	if !ok {
		return
	}

	// Load the aggregate
	list, err := h.repo.Get(ctx, listID)
	if err != nil {
		log.Error("cannot load list for removing item", "error", err, "listId", listID.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not retrieve list")
		return
	}

	if list == nil {
		core.RespondError(w, http.StatusNotFound, "List not found")
		return
	}

	// Remove item from aggregate
	found := false
	for i, existingItem := range list.Items {
		if existingItem.ID == itemID {
			list.Items = append(list.Items[:i], list.Items[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		core.RespondError(w, http.StatusNotFound, "Item not found in list")
		return
	}

	// Save the entire aggregate
	if err := h.repo.Save(ctx, list); err != nil {
		log.Error("error saving list after removing item", "error", err, "listId", listID.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not remove item from list")
		return
	}

	// Post-deletion links
	links := []core.Link{
		{Rel: "list", Href: fmt.Sprintf("/lists/%s", listID)},
		{Rel: "collection", Href: fmt.Sprintf("/lists/%s/items", listID)},
		{Rel: "create", Href: fmt.Sprintf("/lists/%s/items", listID)},
	}

	w.WriteHeader(http.StatusNoContent)
	core.RespondSuccess(w, nil, links...)
}

func (h *ListHandler) parseItemIDParam(w http.ResponseWriter, r *http.Request, log core.Logger) (uuid.UUID, bool) {
	rawID := strings.TrimSpace(chi.URLParam(r, "childId"))
	if rawID == "" {
		core.RespondError(w, http.StatusBadRequest, "Missing childId path parameter")
		return uuid.Nil, false
	}

	id, err := uuid.Parse(rawID)
	if err != nil {
		log.Debug("invalid childId parameter", "childId", rawID, "error", err)
		core.RespondError(w, http.StatusBadRequest, "Invalid childId format")
		return uuid.Nil, false
	}

	return id, true
}

func (h *ListHandler) decodeItemPayload(w http.ResponseWriter, r *http.Request, log core.Logger) (Item, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, ListMaxBodyBytes)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var item Item
	if err := dec.Decode(&item); err != nil {
		log.Error("cannot decode item request body", "error", err)
		core.RespondError(w, http.StatusBadRequest, "Item request body could not be decoded")
		return Item{}, false
	}

	return item, true
}

// Child entity operations (Tags)
func (h *ListHandler) AddTagToList(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	listID, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	tag, ok := h.decodeTagPayload(w, r, log)
	if !ok {
		return
	}

	// Load the aggregate
	list, err := h.repo.Get(ctx, listID)
	if err != nil {
		log.Error("cannot load list for adding tag", "error", err, "listId", listID.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not retrieve list")
		return
	}

	if list == nil {
		core.RespondError(w, http.StatusNotFound, "List not found")
		return
	}

	// Add tag to aggregate
	tag.EnsureID()
	tag.BeforeCreate()
	list.Tags = append(list.Tags, tag)

	// Save the entire aggregate
	if err := h.repo.Save(ctx, list); err != nil {
		log.Error("error saving list with new tag", "error", err, "listId", listID.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not add tag to list")
		return
	}

	// Child response
	w.WriteHeader(http.StatusCreated)
	core.RespondChild(w, list, &tag)
}

func (h *ListHandler) UpdateTagInList(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	listID, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	tagID, ok := h.parseTagIDParam(w, r, log)
	if !ok {
		return
	}

	tag, ok := h.decodeTagPayload(w, r, log)
	if !ok {
		return
	}

	// Load the aggregate
	list, err := h.repo.Get(ctx, listID)
	if err != nil {
		log.Error("cannot load list for updating tag", "error", err, "listId", listID.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not retrieve list")
		return
	}

	if list == nil {
		core.RespondError(w, http.StatusNotFound, "List not found")
		return
	}

	// Find and update tag in aggregate
	found := false
	for i, existingTag := range list.Tags {
		if existingTag.ID == tagID {
			tag.SetID(tagID)
			tag.BeforeUpdate()
			list.Tags[i] = tag
			found = true
			break
		}
	}

	if !found {
		core.RespondError(w, http.StatusNotFound, "Tag not found in list")
		return
	}

	// Save the entire aggregate
	if err := h.repo.Save(ctx, list); err != nil {
		log.Error("error saving list with updated tag", "error", err, "listId", listID.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not update tag in list")
		return
	}

	// Child response
	core.RespondChild(w, list, &tag)
}

func (h *ListHandler) RemoveTagFromList(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	listID, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	tagID, ok := h.parseTagIDParam(w, r, log)
	if !ok {
		return
	}

	// Load the aggregate
	list, err := h.repo.Get(ctx, listID)
	if err != nil {
		log.Error("cannot load list for removing tag", "error", err, "listId", listID.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not retrieve list")
		return
	}

	if list == nil {
		core.RespondError(w, http.StatusNotFound, "List not found")
		return
	}

	// Remove tag from aggregate
	found := false
	for i, existingTag := range list.Tags {
		if existingTag.ID == tagID {
			list.Tags = append(list.Tags[:i], list.Tags[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		core.RespondError(w, http.StatusNotFound, "Tag not found in list")
		return
	}

	// Save the entire aggregate
	if err := h.repo.Save(ctx, list); err != nil {
		log.Error("error saving list after removing tag", "error", err, "listId", listID.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not remove tag from list")
		return
	}

	// Post-deletion links
	links := []core.Link{
		{Rel: "list", Href: fmt.Sprintf("/lists/%s", listID)},
		{Rel: "collection", Href: fmt.Sprintf("/lists/%s/tags", listID)},
		{Rel: "create", Href: fmt.Sprintf("/lists/%s/tags", listID)},
	}

	w.WriteHeader(http.StatusNoContent)
	core.RespondSuccess(w, nil, links...)
}

func (h *ListHandler) parseTagIDParam(w http.ResponseWriter, r *http.Request, log core.Logger) (uuid.UUID, bool) {
	rawID := strings.TrimSpace(chi.URLParam(r, "childId"))
	if rawID == "" {
		core.RespondError(w, http.StatusBadRequest, "Missing childId path parameter")
		return uuid.Nil, false
	}

	id, err := uuid.Parse(rawID)
	if err != nil {
		log.Debug("invalid childId parameter", "childId", rawID, "error", err)
		core.RespondError(w, http.StatusBadRequest, "Invalid childId format")
		return uuid.Nil, false
	}

	return id, true
}

func (h *ListHandler) decodeTagPayload(w http.ResponseWriter, r *http.Request, log core.Logger) (Tag, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, ListMaxBodyBytes)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var tag Tag
	if err := dec.Decode(&tag); err != nil {
		log.Error("cannot decode tag request body", "error", err)
		core.RespondError(w, http.StatusBadRequest, "Tag request body could not be decoded")
		return Tag{}, false
	}

	return tag, true
}

// Helper methods
func (h *ListHandler) parseIDParam(w http.ResponseWriter, r *http.Request, log core.Logger) (uuid.UUID, bool) {
	rawID := strings.TrimSpace(chi.URLParam(r, "id"))
	if rawID == "" {
		core.RespondError(w, http.StatusBadRequest, "Missing id path parameter")
		return uuid.Nil, false
	}

	id, err := uuid.Parse(rawID)
	if err != nil {
		log.Debug("invalid id parameter", "id", rawID, "error", err)
		core.RespondError(w, http.StatusBadRequest, "Invalid id format")
		return uuid.Nil, false
	}

	return id, true
}

func (h *ListHandler) decodeListPayload(w http.ResponseWriter, r *http.Request, log core.Logger) (List, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, ListMaxBodyBytes)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var list List
	if err := dec.Decode(&list); err != nil {
		log.Error("cannot decode request body", "error", err)
		core.RespondError(w, http.StatusBadRequest, "Request body could not be decoded")
		return List{}, false
	}

	if err := ensureListSingleJSONValue(dec); err != nil {
		log.Error("request body contains extra data", "error", err)
		core.RespondError(w, http.StatusBadRequest, "Request body contains unexpected data")
		return List{}, false
	}

	return list, true
}

func ensureListSingleJSONValue(dec *json.Decoder) error {
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("additional JSON values detected")
		}
		return err
	}
	return nil
}

func (h *ListHandler) Log() core.Logger {
	return h.xparams.Log
}

func (h *ListHandler) logForRequest(r *http.Request) core.Logger {
	return h.xparams.Log.With(
		"request_id", middleware.GetReqID(r.Context()),
		"method", r.Method,
		"path", r.URL.Path,
	)
}

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidateCreateList validates a List entity before creation.
// TODO: Implement validation logic
func ValidateCreateList(ctx context.Context, list List) []ValidationError {
	// TODO: Add validation logic here
	return []ValidationError{}
}

// ValidateUpdateList validates a List entity before update.
// TODO: Implement validation logic
func ValidateUpdateList(ctx context.Context, id uuid.UUID, list List) []ValidationError {
	// TODO: Add validation logic here
	return []ValidationError{}
}

// ValidateDeleteList validates a List entity before deletion.
// TODO: Implement validation logic
func ValidateDeleteList(ctx context.Context, id uuid.UUID) []ValidationError {
	// TODO: Add validation logic here
	return []ValidationError{}
}
