package todo

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/adrianpk/hatmax/pkg/lib/hm"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/adrianpk/hatmax-ref/services/todo/internal/config"
)

const ItemMaxBodyBytes = 1 << 20

// NewItemHandler creates a new ItemHandler.
func NewItemHandler(svc ItemService, xparams config.XParams) *ItemHandler {
	return &ItemHandler{
		svc:     svc,
		xparams: xparams,
	}
}

type ItemHandler struct {
	svc     ItemService
	xparams config.XParams
}



func (h *ItemHandler) RegisterRoutes(r chi.Router) {
	r.Route("/items", func(r chi.Router) {
		r.Post("/", h.CreateItem)
		r.Get("/", h.ListItems)
		r.Get("/{id}", h.GetItem)
		r.Put("/{id}", h.UpdateItem)
	})
}

func (h *ItemHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	model, ok := h.decodeItemPayload(w, r, log)
	if !ok {
		return
	}

	model.EnsureID()
	model.BeforeCreate()

	validationErrors := ValidateCreateItem(ctx, model)
	if len(validationErrors) > 0 {
		log.Debug("validation failed", "errors", validationErrors)
		hm.Error(w, http.StatusBadRequest, "validation_failed", "Validation failed", validationErrors...)
		return
	}

	if err := h.svc.Create(ctx, &model); err != nil {
		log.Error("failed to create item", "error", err)
		hm.Error(w, http.StatusInternalServerError, "internal_error", "Could not create item")
		return
	}

	hm.Respond(w, http.StatusCreated, model, nil)
}

func (h *ItemHandler) GetItem(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	model, err := h.svc.Get(ctx, id)
	if err != nil {
		log.Error("failed to get item", "error", err, "id", id.String())
		hm.Error(w, http.StatusInternalServerError, "internal_error", "Could not retrieve item")
		return
	}

	if model == nil {
		hm.Error(w, http.StatusNotFound, "not_found", "Item not found")
		return
	}

	hm.Respond(w, http.StatusOK, model, nil)
}

func (h *ItemHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	model, ok := h.decodeItemPayload(w, r, log)
	if !ok {
		return
	}

	model.SetID(id)
	model.BeforeUpdate()

	validationErrors := ValidateUpdateItem(ctx, id, model)
	if len(validationErrors) > 0 {
		log.Debug("validation failed", "errors", validationErrors, "id", id.String())
		hm.Error(w, http.StatusBadRequest, "validation_failed", "Validation failed", validationErrors...)
		return
	}

	if err := h.svc.Update(ctx, &model); err != nil {
		log.Error("failed to update item", "error", err, "id", id.String())
		hm.Error(w, http.StatusInternalServerError, "internal_error", "Could not update item")
		return
	}

	hm.Respond(w, http.StatusOK, model, nil)
}

func (h *ItemHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	models, err := h.svc.List(ctx)
	if err != nil {
		log.Error("failed to list items", "error", err)
		hm.Error(w, http.StatusInternalServerError, "internal_error", "Could not list items")
		return
	}

	// TODO: Add pagination metadata once repository supports it.
	hm.Respond(w, http.StatusOK, models, nil)
}

func (h *ItemHandler) parseIDParam(w http.ResponseWriter, r *http.Request, log hm.Logger) (uuid.UUID, bool) {
	rawID := strings.TrimSpace(chi.URLParam(r, "id"))
	if rawID == "" {
		hm.Error(w, http.StatusBadRequest, "bad_request", "Missing id path parameter")
		return uuid.Nil, false
	}

	id, err := uuid.Parse(rawID)
	if err != nil {
		log.Debug("invalid id parameter", "id", rawID, "error", err)
		hm.Error(w, http.StatusBadRequest, "bad_request", "Invalid id format")
		return uuid.Nil, false
	}

	return id, true
}

func (h *ItemHandler) decodeItemPayload(w http.ResponseWriter, r *http.Request, log hm.Logger) (Item, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, ItemMaxBodyBytes)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var model Item
	if err := dec.Decode(&model); err != nil {
		log.Error("failed to decode request body", "error", err)
		hm.Error(w, http.StatusBadRequest, "invalid_payload", "Request body could not be decoded")
		return Item{}, false
	}

	if err := ensureItemSingleJSONValue(dec); err != nil {
		log.Error("request body contains extra data", "error", err)
		hm.Error(w, http.StatusBadRequest, "invalid_payload", "Request body contains unexpected data")
		return Item{}, false
	}

	return model, true
}

func ensureItemSingleJSONValue(dec *json.Decoder) error {
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("additional JSON values detected")
		}
		return err
	}
	return nil
}

func (h *ItemHandler) Log() hm.Logger {
	return h.xparams.Log
}

func (h *ItemHandler) logForRequest(r *http.Request) hm.Logger {
	return h.xparams.Log.With(
		"request_id", middleware.GetReqID(r.Context()),
		"method", r.Method,
		"path", r.URL.Path,
	)
}