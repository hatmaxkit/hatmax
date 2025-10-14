package core

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// ResourceIdentifiable provides both ID and resource type for URL generation
type ResourceIdentifiable interface {
	ID() uuid.UUID
	ResourceType() string // Resource type for URLs: "lists", "items", "users", "profiles"
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

// RESTfulLinksFor generates standard CRUD links for a resource object
func RESTfulLinksFor(obj ResourceIdentifiable, basePath ...string) []Link {
	resourceName := obj.ResourceType()
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
// parent: the aggregate root (e.g., List)
// child: the child entity (e.g., Item)
func ChildLinksFor(parent, child ResourceIdentifiable) []Link {
	parentResource := parent.ResourceType()
	childResource := child.ResourceType()
	
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

func (b *LinkBuilder) AddRESTfulLinks(obj ResourceIdentifiable) *LinkBuilder {
	b.links = append(b.links, RESTfulLinksFor(obj)...)
	return b
}

func (b *LinkBuilder) AddChildLinks(parent, child ResourceIdentifiable) *LinkBuilder {
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
func RespondWithLinks(w http.ResponseWriter, obj ResourceIdentifiable) {
	links := RESTfulLinksFor(obj)
	RespondSuccess(w, obj, links...)
}

func RespondCollection(w http.ResponseWriter, data interface{}, resourceName string) {
	links := CollectionLinksFor(resourceName)
	RespondSuccess(w, data, links...)
}

func RespondChild(w http.ResponseWriter, parent, child ResourceIdentifiable) {
	links := ChildLinksFor(parent, child)
	RespondSuccess(w, child, links...)
}