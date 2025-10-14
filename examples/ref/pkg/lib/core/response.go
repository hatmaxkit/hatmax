package core

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gertd/go-pluralize"
)

var pluralizer = pluralize.NewClient()

// Standard RESTful link relations
const (
	RelSelf       = "self"
	RelCollection = "collection"
	RelCreate     = "create"
	RelUpdate     = "update"
	RelDelete     = "delete"
	RelEdit       = "edit"
	RelParent     = "parent"
	RelNext       = "next"
	RelPrev       = "prev"
)

// Pluralize converts singular resource type to plural for URLs.
func Pluralize(singular string) string {
	return pluralizer.Plural(singular)
}

// Singularize converts plural resource type to singular.
func Singularize(plural string) string {
	return pluralizer.Singular(plural)
}

// IsPlural returns true if word is plural.
func IsPlural(word string) bool {
	return pluralizer.IsPlural(word)
}

// Link represents a HATEOAS link
type Link struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

// SuccessResponse defines the envelope for successful responses.
type SuccessResponse struct {
	Data  interface{} `json:"data"`
	Meta  interface{} `json:"meta,omitempty"`
	Links []Link      `json:"links,omitempty"`
}

// ErrorPayload defines the internal structure of the error object
type ErrorPayload struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details []ValidationError `json:"details,omitempty"`
}

// ErrorResponse defines the envelope for error responses
type ErrorResponse struct {
	Error ErrorPayload `json:"error"`
}

// RespondSuccess sends a successful JSON response with HATEOAS links
func RespondSuccess(w http.ResponseWriter, data interface{}, links ...Link) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SuccessResponse{Data: data, Links: links})
}

// RespondError sends a simple error response
func RespondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorPayload{
			Code:    http.StatusText(code),
			Message: message,
		},
	})
}

// Respond sends a successful JSON response (backward compatibility)
// TODO: Remove this one
func Respond(w http.ResponseWriter, code int, data interface{}, meta interface{}) {
	if code == http.StatusNoContent {
		w.WriteHeader(code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(SuccessResponse{Data: data, Meta: meta})
}

// Error sends a JSON error response (backward compatibility)
// TODO: Remove this one
func Error(w http.ResponseWriter, code int, errorCode string, message string, details ...ValidationError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorPayload{
			Code:    errorCode,
			Message: message,
			Details: details,
		},
	})
}

// RESTfulLinksFor generates standard CRUD links for a resource object
func RESTfulLinksFor(obj Identifiable, basePath ...string) []Link {
	singular := obj.ResourceType()
	plural := Pluralize(singular)
	id := obj.GetID().String()

	base := ""
	if len(basePath) > 0 {
		base = basePath[0]
	}

	resourcePath := fmt.Sprintf("%s/%s", base, plural)
	itemPath := fmt.Sprintf("%s/%s", resourcePath, id)

	return []Link{
		{Rel: RelSelf, Href: itemPath},
		{Rel: RelUpdate, Href: itemPath},
		{Rel: RelDelete, Href: itemPath},
		{Rel: RelCollection, Href: resourcePath},
	}
}

// CollectionLinksFor generates collection links for a resource type
func CollectionLinksFor(resourceType string, basePath ...string) []Link {
	plural := Pluralize(resourceType)
	base := ""
	if len(basePath) > 0 {
		base = basePath[0]
	}

	resourcePath := fmt.Sprintf("%s/%s", base, plural)

	return []Link{
		{Rel: RelSelf, Href: resourcePath},
		{Rel: RelCreate, Href: resourcePath},
	}
}

// ChildLinksFor generates links for child entities within aggregates
func ChildLinksFor(parent, child Identifiable) []Link {
	parentType := parent.ResourceType()
	childType := child.ResourceType()

	parentPlural := Pluralize(parentType)
	childPlural := Pluralize(childType)

	parentID := parent.GetID().String()
	childID := child.GetID().String()

	parentPath := fmt.Sprintf("/%s/%s", parentPlural, parentID)
	childCollectionPath := fmt.Sprintf("%s/%s", parentPath, childPlural)
	childItemPath := fmt.Sprintf("%s/%s", childCollectionPath, childID)

	return []Link{
		{Rel: RelSelf, Href: childItemPath},
		{Rel: RelUpdate, Href: childItemPath},
		{Rel: RelDelete, Href: childItemPath},
		{Rel: RelParent, Href: parentPath},
		{Rel: RelCollection, Href: childCollectionPath},
	}
}

// LinkBuilder provides a fluent interface for building custom links
type LinkBuilder struct {
	links []Link
}

func NewLinkBuilder() *LinkBuilder {
	return &LinkBuilder{links: []Link{}}
}

func (b *LinkBuilder) AddRESTfulLinks(obj Identifiable) *LinkBuilder {
	b.links = append(b.links, RESTfulLinksFor(obj)...)
	return b
}

func (b *LinkBuilder) AddChildLinks(parent, child Identifiable) *LinkBuilder {
	b.links = append(b.links, ChildLinksFor(parent, child)...)
	return b
}

func (b *LinkBuilder) Custom(rel, href string) *LinkBuilder {
	b.links = append(b.links, Link{Rel: rel, Href: href})
	return b
}

func (b *LinkBuilder) Add(links ...Link) *LinkBuilder {
	b.links = append(b.links, links...)
	return b
}

func (b *LinkBuilder) Build() []Link {
	return b.links
}

// High-level convenience methods
func RespondWithLinks(w http.ResponseWriter, obj Identifiable) {
	links := RESTfulLinksFor(obj)
	RespondSuccess(w, obj, links...)
}

func RespondCollection(w http.ResponseWriter, data interface{}, resourceType string) {
	links := CollectionLinksFor(resourceType)
	RespondSuccess(w, data, links...)
}

func RespondChild(w http.ResponseWriter, parent, child Identifiable) {
	links := ChildLinksFor(parent, child)
	RespondSuccess(w, child, links...)
}
