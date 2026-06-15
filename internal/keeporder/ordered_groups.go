// Package keeporder contains definitions for internal use.
package keeporder

// OrderedGroups keeps the order of requests by the given key for each group.
type OrderedGroups interface {
	Add(key string, fn func())
	Remove(key string)
	Stop()
}
