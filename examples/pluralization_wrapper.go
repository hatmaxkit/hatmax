package core

import (
	"github.com/gertd/go-pluralize"
)

var pluralizer = pluralize.NewClient()

// Pluralize converts singular resource type to plural for URLs
// "user" -> "users", "item" -> "items", "category" -> "categories"
func Pluralize(singular string) string {
	return pluralizer.Plural(singular)
}

// Singularize converts plural resource type to singular 
// "users" -> "user", "items" -> "item", "categories" -> "category"
func Singularize(plural string) string {
	return pluralizer.Singular(plural)
}

// IsPlural checks if a word is already plural
func IsPlural(word string) bool {
	return pluralizer.IsPlural(word)
}