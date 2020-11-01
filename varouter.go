// Copyright 2020 Vedran Vuk. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// Package varouter implements a flexible path matching router with support for
// variables and wildcards that does not suffer on performance with large
// number of registered items.
package varouter

import (
	"errors"
)

// elementMap is a map of path element names to their definitions.
type elementMap map[string]*element

// element defines a path element.
type element struct {
	// subs is a map of sub path elements.
	subs elementMap
	// wildcard specifies if this element contains a wildcard element.
	wildcard bool
	// template is the template string that registered this element.
	template string
	// container, if not empty, specifies that this element is a container
	// of a single placeholder element and the value is it's name.
	container string
}

// newElement returns a new element instance.
func newElement() *element { return &element{subs: make(elementMap)} }

// Varouter is a flexible path matching router with support for path element
// variables and wildcards for matching multiple templates that does not suffer
// on large number of registered items.
//
// Register parses a template path, splits it on path separators and builds a
// tree of registered paths using maps.
//
// Match matches specified path against registered templates and returns a list
// of matched templates and any placeholders parsed from the path.
//
// Adapters for handlers of various packages can easily be built.
//
// For details on use see Register and Match.
type Varouter struct {
	count int      // count is the number of registered templates.
	root  *element // root is the root element.

	Separator   byte // Separator is the path separator character to use. (Default '/').
	Placeholder byte // Placeholder is the variable placeholder character to use. Default (':').
	Wildcard    byte // Wildcard is the wildcard character to use. Default: ('*').
}

// New returns a new *Varouter instance with default configuration.
func New() *Varouter { return NewVarouter('/', ':', '*') }

// NewVarouter returns a new *Varouter instance with the given separator,
// placeholder and wildcard.
func NewVarouter(separator, placeholder, wildcard byte) *Varouter {
	return &Varouter{
		root:        newElement(),
		Separator:   separator,
		Placeholder: placeholder,
		Wildcard:    wildcard,
	}
}

// Register registers a template which will be matched against a path specified
// by Match method. If an error occurs during registration it is returned and
// no template was registered.
//
// Template must be a rooted path, starting with the defined Separator.
// Match path is matched exactly, including any possibly multiple Separators
// anywhere in the registered template and dotdot names.
// For example, all of the following registration templates are legal:
// "/home", "/home/", "/home//", "/home////users//", "../home", "/what/./the".
//
// A Wildcard template which will match a path if it is prefixed by it can be
// defined by adding a Wildcard character suffix to the template where the
// suffix appears as if instead of a name, e.g. "/home/users/*".
//
// Wildcard characters as part of the path element name are legal and registered
// as is and are left to be interpreted by the user. For example:
// "/usr/lib*", "/usr/lib*/bash", "/tests/*_test.go", "/home/users/*/.config".
//
// Template path elements can be defined as Placeholders by prefixing the path
// element with a Placeholder which matches the whole path element as a value
// of the named path element and are returned as a map. For example:
// "/home/users/:user", "/:item/:action/", "/movies/:id/comments/".
//
// Only one Placeholder per registered template tree path element level is
// allowed. For example:
// "/edit/:user" and "/export/:user" is allowed but
// "/edit/:user" and "/edit/:admin" is not.
func (vr *Varouter) Register(template string) (err error) {
	var wildcard bool
	var current *element = vr.root
	var cursor, marker, length int = 0, 0, len(template)
	if length == 0 {
		return errors.New("varouter: empty path")
	}
	if template[0] != vr.Separator {
		return errors.New("varouter: path must start with a separator")
	}
	if length > 1 && template[length-1] == vr.Wildcard && template[length-2] == vr.Separator {
		length--
		wildcard = true
	}
	for cursor, marker = 1, 0; cursor < length; cursor++ {
		if template[cursor] != vr.Separator {
			continue
		}
		if current, err = vr.getOrAddSub(current, template[marker:cursor], false); err != nil {
			return
		}
		marker = cursor
	}
	if current, err = vr.getOrAddSub(current, template[marker:cursor], wildcard); err != nil {
		return
	}
	current.template = template
	return nil
}

// getOrAddSub is a helper to Register that gets a sub element by name or adds
// one if it does not existing respecting the element type in the process.
func (vr *Varouter) getOrAddSub(elem *element, name string, wildcard bool) (e *element, err error) {
	var container bool = len(name) > 1 && name[1] == vr.Placeholder
	if container {
		name = name[2:]
	}
	var ok bool
	if e, ok = elem.subs[name]; ok {
		return
	}
	if elem.container != "" {
		return nil, errors.New("varouter: only one placeholder allowed per level")
	}
	if len(elem.subs) > 0 && container {
		return nil, errors.New("varouter: cannot register a placeholder on a path level with defined elements")
	}
	e = newElement()
	if wildcard {
		elem.wildcard = true
		elem.subs[name+string(vr.Wildcard)] = e
	} else {
		elem.subs[name] = e
	}
	if container {
		elem.container = name
	}
	vr.count++
	return
}

// PlaceholderMap is a map of Placeholder names to their values parsed from a
// Match path.
type PlaceholderMap map[string]string

// Match matches a path against registered templates and returns the names of
// matched templates, a map of parsed param names to param values and a bool
// indicating if a match occured and previous two result vars are valid.
//
// Returned template names will consist of possibly one or more Wildcard
// templates that matched the path and possibly a template that matched the
// path exactly, regardless if template has any placeholders.
//
// If no templates were matched the resulting templates will be nil.
// If no params were parsed from the path the resulting ParamMap wil be nil.
func (vr *Varouter) Match(path string) (templates []string, params PlaceholderMap, matched bool) {
	var cursor, marker int
	var length int = len(path)
	var current *element = vr.root
	for cursor, marker = 1, 0; cursor < length; cursor++ {
		if path[cursor] != vr.Separator {
			continue
		}
		if current = vr.get(current, path[marker:cursor], &templates, &params); current == nil {
			return nil, nil, false
		}
		marker = cursor
	}
	matched = len(templates) > 0
	if current = vr.get(current, path[marker:cursor], &templates, &params); current == nil && !matched {
		return nil, nil, false
	}
	if cursor == length && current != nil && current.template != "" {
		templates = append(templates, current.template)
		matched = true
	}
	return
}

// get gets a sub element of elem by name in a manner depending on element type
// and returns it or nil if element is not found.
func (vr *Varouter) get(elem *element, name string, templates *[]string, params *PlaceholderMap) (e *element) {
	if elem.container != "" {
		if *params == nil {
			*params = make(PlaceholderMap)
		}
		e = elem.subs[elem.container]
		(*params)[elem.container] = name[1:]
		return
	}
	var ok bool
	if elem.wildcard {
		e = elem.subs[string([]byte{vr.Separator, vr.Wildcard})]
		*templates = append(*templates, e.template)
		ok = true
	}
	var save *element
	if save, ok = elem.subs[name]; ok {
		return save
	}
	return
}

// printelement recursively puts names of defined templates in e to a.
func printElement(e *element, a *[]string) {
	for _, elem := range e.subs {
		if elem.template != "" {
			*a = append(*a, elem.template)
		}
		printElement(elem, a)
	}
	return
}

// DefinedTemplates returns a slice of defined templates.
func (vr *Varouter) DefinedTemplates() (a []string) {
	a = make([]string, 0, vr.count)
	printElement(vr.root, &a)
	return a
}

// NumTemplates returns number of defined templates.
func (vr *Varouter) NumTemplates() int { return vr.count }
