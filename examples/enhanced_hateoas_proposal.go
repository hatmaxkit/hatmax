package core

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/google/uuid"
)

// ResourceIdentifiable extends Identifiable to provide resource type info
type ResourceIdentifiable interface {
	Identifiable
	ResourceName() string // e.g., "lists", "items", "users"
}

// AutoResourceIdentifiable infers resource name from struct type
type AutoResourceIdentifiable interface {
	Identifiable
	// No ResourceName() method - will be inferred from struct name
}

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

// inferResourceName infers resource name from struct type
func inferResourceName(obj interface{}) string {
	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	
	name := t.Name()
	// Convert "List" -> "lists", "TodoItem" -> "todoitems"
	return strings.ToLower(name) + "s"
}

// getResourceName gets resource name from object
func getResourceName(obj interface{}) string {
	if r, ok := obj.(ResourceIdentifiable); ok {
		return r.ResourceName()
	}
	if _, ok := obj.(AutoResourceIdentifiable); ok {
		return inferResourceName(obj)
	}
	// Fallback to reflection
	return inferResourceName(obj)
}

// RESTfulLinksFor generates standard CRUD links for a resource object
func RESTfulLinksFor(obj Identifiable, basePath ...string) []Link {
	resourceName := getResourceName(obj)
	id := obj.ID().String()
	
	base := ""
	if len(basePath) > 0 {
		base = basePath[0]
	}
	
	resourcePath := fmt.Sprintf("%s/%s", base, resourceName)
	itemPath := fmt.Sprintf("%s/%s", resourcePath, id)
	
	return []Link{
		{Rel: RelSelf, Href: itemPath},
		{Rel: RelUpdate, Href: itemPath},
		{Rel: RelDelete, Href: itemPath},
		{Rel: RelCollection, Href: resourcePath},
	}
}

// CollectionLinksFor generates collection links for a resource type
func CollectionLinksFor(resourceName string, basePath ...string) []Link {
	base := ""
	if len(basePath) > 0 {
		base = basePath[0]
	}
	
	resourcePath := fmt.Sprintf("%s/%s", base, resourceName)
	
	return []Link{
		{Rel: RelSelf, Href: resourcePath},
		{Rel: RelCreate, Href: resourcePath},
	}
}

// ChildLinksFor generates links for child entities within aggregates
func ChildLinksFor(parent, child Identifiable) []Link {
	parentResource := getResourceName(parent)
	childResource := getResourceName(child)
	
	parentID := parent.ID().String()
	childID := child.ID().String()
	
	parentPath := fmt.Sprintf("/%s/%s", parentResource, parentID)
	childCollectionPath := fmt.Sprintf("%s/%s", parentPath, childResource)
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

func (b *LinkBuilder) Self(href string) *LinkBuilder {
	b.links = append(b.links, Link{Rel: RelSelf, Href: href})
	return b
}

func (b *LinkBuilder) Collection(href string) *LinkBuilder {
	b.links = append(b.links, Link{Rel: RelCollection, Href: href})
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

func (b *LinkBuilder) AddForResource(obj Identifiable) *LinkBuilder {
	b.links = append(b.links, RESTfulLinksFor(obj)...)
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

func RespondCollection(w http.ResponseWriter, data interface{}, resourceName string) {
	links := CollectionLinksFor(resourceName)
	RespondSuccess(w, data, links...)
}

func RespondChild(w http.ResponseWriter, parent, child Identifiable) {
	links := ChildLinksFor(parent, child)
	RespondSuccess(w, child, links...)
}