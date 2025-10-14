package core

import (
	"fmt"
	"net/http"
)

// Standard RESTful link relations
const (
	RelSelf       = "self"
	RelCollection = "collection"  
	RelCreate     = "create"
	RelUpdate     = "update"
	RelDelete     = "delete"
	RelEdit       = "edit"
	RelNext       = "next"
	RelPrev       = "prev"
)

// RESTfulLinks generates standard CRUD links for a resource
func RESTfulLinks(resource string, id string, basePath ...string) []Link {
	base := ""
	if len(basePath) > 0 {
		base = basePath[0]
	}
	
	resourcePath := fmt.Sprintf("%s/%s", base, resource)
	itemPath := fmt.Sprintf("%s/%s", resourcePath, id)
	
	return []Link{
		{Rel: RelSelf, Href: itemPath},
		{Rel: RelUpdate, Href: itemPath},
		{Rel: RelDelete, Href: itemPath},
		{Rel: RelCollection, Href: resourcePath},
	}
}

// CollectionLinks generates standard collection links
func CollectionLinks(resource string, basePath ...string) []Link {
	base := ""
	if len(basePath) > 0 {
		base = basePath[0]
	}
	
	resourcePath := fmt.Sprintf("%s/%s", base, resource)
	
	return []Link{
		{Rel: RelSelf, Href: resourcePath},
		{Rel: RelCreate, Href: resourcePath},
	}
}

// ChildResourceLinks generates links for child entities within aggregates
func ChildResourceLinks(parentResource, parentID, childResource, childID string) []Link {
	parentPath := fmt.Sprintf("/%s/%s", parentResource, parentID)
	childCollectionPath := fmt.Sprintf("%s/%s", parentPath, childResource)
	childItemPath := fmt.Sprintf("%s/%s", childCollectionPath, childID)
	
	return []Link{
		{Rel: RelSelf, Href: childItemPath},
		{Rel: RelUpdate, Href: childItemPath},
		{Rel: RelDelete, Href: childItemPath},
		{Rel: "parent", Href: parentPath},
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

func (b *LinkBuilder) Build() []Link {
	return b.links
}

// Convenience method to respond with standard RESTful links
func RespondWithRESTfulLinks(w http.ResponseWriter, data interface{}, resource, id string) {
	links := RESTfulLinks(resource, id)
	RespondSuccess(w, data, links...)
}

// Convenience method for collection responses
func RespondCollection(w http.ResponseWriter, data interface{}, resource string) {
	links := CollectionLinks(resource)
	RespondSuccess(w, data, links...)
}