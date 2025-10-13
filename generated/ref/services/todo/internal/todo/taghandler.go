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

const TagMaxBodyBytes = 1 << 20

// NewTagHandler creates a new TagHandler.
func NewTagHandler(svc TagService, xparams config.XParams) *TagHandler {
	return &TagHandler{
		svc:     svc,
		xparams: xparams,
	}
}

type TagHandler struct {
	svc     TagService
	xparams config.XParams
}



func (h *TagHandler) RegisterRoutes(r chi.Router) {
	r.Route("/tags", func(r chi.Router) {
		r.Post("/", h.CreateTag)
		r.Get("/", h.ListTags)
		r.Get("/{id}", h.GetTag)
		r.Put("/{id}", h.UpdateTag)
	})
}

func (h *TagHandler) CreateTag(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	model, ok := h.decodeTagPayload(w, r, log)
	if !ok {
		return
	}

	model.EnsureID()
	model.BeforeCreate()

	validationErrors := ValidateCreateTag(ctx, model)
	if len(validationErrors) > 0 {
		log.Debug("validation failed", "errors", validationErrors)
		hm.Error(w, http.StatusBadRequest, "validation_failed", "Validation failed", validationErrors...)
		return
	}

	if err := h.svc.Create(ctx, &model); err != nil {
		log.Error("failed to create tag", "error", err)
		hm.Error(w, http.StatusInternalServerError, "internal_error", "Could not create tag")
		return
	}

	hm.Respond(w, http.StatusCreated, model, nil)
}

func (h *TagHandler) GetTag(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	model, err := h.svc.Get(ctx, id)
	if err != nil {
		log.Error("failed to get tag", "error", err, "id", id.String())
		hm.Error(w, http.StatusInternalServerError, "internal_error", "Could not retrieve tag")
		return
	}

	if model == nil {
		hm.Error(w, http.StatusNotFound, "not_found", "Tag not found")
		return
	}

	hm.Respond(w, http.StatusOK, model, nil)
}

func (h *TagHandler) UpdateTag(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	model, ok := h.decodeTagPayload(w, r, log)
	if !ok {
		return
	}

	model.SetID(id)
	model.BeforeUpdate()

	validationErrors := ValidateUpdateTag(ctx, id, model)
	if len(validationErrors) > 0 {
		log.Debug("validation failed", "errors", validationErrors, "id", id.String())
		hm.Error(w, http.StatusBadRequest, "validation_failed", "Validation failed", validationErrors...)
		return
	}

	if err := h.svc.Update(ctx, &model); err != nil {
		log.Error("failed to update tag", "error", err, "id", id.String())
		hm.Error(w, http.StatusInternalServerError, "internal_error", "Could not update tag")
		return
	}

	hm.Respond(w, http.StatusOK, model, nil)
}

func (h *TagHandler) ListTags(w http.ResponseWriter, r *http.Request) {
	log := h.logForRequest(r)
	ctx := r.Context()

	models, err := h.svc.List(ctx)
	if err != nil {
		log.Error("failed to list tags", "error", err)
		hm.Error(w, http.StatusInternalServerError, "internal_error", "Could not list tags")
		return
	}

	// TODO: Add pagination metadata once repository supports it.
	hm.Respond(w, http.StatusOK, models, nil)
}

func (h *TagHandler) parseIDParam(w http.ResponseWriter, r *http.Request, log hm.Logger) (uuid.UUID, bool) {
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

func (h *TagHandler) decodeTagPayload(w http.ResponseWriter, r *http.Request, log hm.Logger) (Tag, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, TagMaxBodyBytes)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var model Tag
	if err := dec.Decode(&model); err != nil {
		log.Error("failed to decode request body", "error", err)
		hm.Error(w, http.StatusBadRequest, "invalid_payload", "Request body could not be decoded")
		return Tag{}, false
	}

	if err := ensureTagSingleJSONValue(dec); err != nil {
		log.Error("request body contains extra data", "error", err)
		hm.Error(w, http.StatusBadRequest, "invalid_payload", "Request body contains unexpected data")
		return Tag{}, false
	}

	return model, true
}

func ensureTagSingleJSONValue(dec *json.Decoder) error {
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("additional JSON values detected")
		}
		return err
	}
	return nil
}

func (h *TagHandler) Log() hm.Logger {
	return h.xparams.Log
}

func (h *TagHandler) logForRequest(r *http.Request) hm.Logger {
	return h.xparams.Log.With(
		"request_id", middleware.GetReqID(r.Context()),
		"method", r.Method,
		"path", r.URL.Path,
	)
}