package todo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"example.com/ref/internal/config"
	"example.com/ref/pkg/lib/core"
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

		// Child entity operations (part of the aggregate)
		r.Post("/{id}/items", h.AddItemToList)
		r.Put("/{id}/items/{itemId}", h.UpdateItemInList)
		r.Delete("/{id}/items/{itemId}", h.RemoveItemFromList)
		
		r.Post("/{id}/tags", h.AddTagToList)
		r.Delete("/{id}/tags/{tagId}", h.RemoveTagFromList)
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
		log.Error("failed to create list", "error", err)
		core.RespondError(w, http.StatusInternalServerError, "Could not create list")
		return
	}

	// HATEOAS links for the created resource
	links := []core.Link{
		{Rel: "self", Href: fmt.Sprintf("/lists/%s", list.ID)},
		{Rel: "update", Href: fmt.Sprintf("/lists/%s", list.ID)},
		{Rel: "delete", Href: fmt.Sprintf("/lists/%s", list.ID)},
		{Rel: "items", Href: fmt.Sprintf("/lists/%s/items", list.ID)},
		{Rel: "tags", Href: fmt.Sprintf("/lists/%s/tags", list.ID)},
	}

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
		log.Error("failed to get list", "error", err, "id", id.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not retrieve list")
		return
	}

	if list == nil {
		core.RespondError(w, http.StatusNotFound, "List not found")
		return
	}

	// HATEOAS links for the retrieved resource
	links := []core.Link{
		{Rel: "self", Href: fmt.Sprintf("/lists/%s", list.ID)},
		{Rel: "update", Href: fmt.Sprintf("/lists/%s", list.ID)},
		{Rel: "delete", Href: fmt.Sprintf("/lists/%s", list.ID)},
		{Rel: "items", Href: fmt.Sprintf("/lists/%s/items", list.ID)},
		{Rel: "tags", Href: fmt.Sprintf("/lists/%s/tags", list.ID)},
		{Rel: "collection", Href: "/lists"},
	}

	// Add item-specific links if items exist
	for _, item := range list.Items {
		links = append(links, core.Link{
			Rel:  "item",
			Href: fmt.Sprintf("/lists/%s/items/%s", list.ID, item.ID),
		})
	}

	core.RespondSuccess(w, list, links...)
}

func (h *ListHandler) GetAllLists(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	lists, err := h.repo.List(ctx)
	if err != nil {
		log.Error("failed to list all lists", "error", err)
		core.RespondError(w, http.StatusInternalServerError, "Could not list all lists")
		return
	}

	// HATEOAS links for the collection
	links := []core.Link{
		{Rel: "self", Href: "/lists"},
		{Rel: "create", Href: "/lists"},
	}

	// Add individual resource links
	for _, list := range lists {
		links = append(links, core.Link{
			Rel:  "item",
			Href: fmt.Sprintf("/lists/%s", list.ID),
		})
	}

	core.RespondSuccess(w, lists, links...)
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
		log.Error("failed to update list", "error", err, "id", id.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not update list")
		return
	}

	// HATEOAS links for the updated resource
	links := []core.Link{
		{Rel: "self", Href: fmt.Sprintf("/lists/%s", list.ID)},
		{Rel: "delete", Href: fmt.Sprintf("/lists/%s", list.ID)},
		{Rel: "items", Href: fmt.Sprintf("/lists/%s/items", list.ID)},
		{Rel: "tags", Href: fmt.Sprintf("/lists/%s/tags", list.ID)},
		{Rel: "collection", Href: "/lists"},
	}

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
		log.Error("failed to delete list", "error", err, "id", id.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not delete list")
		return
	}

	// HATEOAS links after deletion
	links := []core.Link{
		{Rel: "collection", Href: "/lists"},
		{Rel: "create", Href: "/lists"},
	}

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
		log.Error("failed to get list for adding item", "error", err, "listId", listID.String())
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
		log.Error("failed to save list with new item", "error", err, "listId", listID.String())
		core.RespondError(w, http.StatusInternalServerError, "Could not add item to list")
		return
	}

	// HATEOAS links for the added item within the aggregate
	links := []core.Link{
		{Rel: "self", Href: fmt.Sprintf("/lists/%s/items/%s", listID, item.ID)},
		{Rel: "update", Href: fmt.Sprintf("/lists/%s/items/%s", listID, item.ID)},
		{Rel: "delete", Href: fmt.Sprintf("/lists/%s/items/%s", listID, item.ID)},
		{Rel: "list", Href: fmt.Sprintf("/lists/%s", listID)},
		{Rel: "collection", Href: fmt.Sprintf("/lists/%s/items", listID)},
	}

	w.WriteHeader(http.StatusCreated)
	core.RespondSuccess(w, item, links...)
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
		log.Error("failed to decode request body", "error", err)
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

func (h *ListHandler) decodeItemPayload(w http.ResponseWriter, r *http.Request, log core.Logger) (Item, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, ListMaxBodyBytes)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var item Item
	if err := dec.Decode(&item); err != nil {
		log.Error("failed to decode item request body", "error", err)
		core.RespondError(w, http.StatusBadRequest, "Item request body could not be decoded")
		return Item{}, false
	}

	return item, true
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