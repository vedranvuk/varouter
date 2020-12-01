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

// Vars is a map of variable names to their values parsed from a path.
type Vars map[string]string

// elements is a map of path element names to their definitions.
type elements map[string]*element

// element defines a path element.
type element struct {
	// subs are the sub elements of this element.
	subs elements
	// template, if not empty, specifies this element is the last element of
	// a registered template and the value is the template.
	template string
	// hasvariable, if not empty, specifies that this element is a container
	// of a single variable element and the value is its name.
	hasvariable string
	// isprefix specifies if this element is a prefix element.
	// Value isignored if this element template is empty.
	isprefix bool
	// isoverride specifies if this element is an override element.
	// Value is ignored if this element template is empty.
	isoverride bool
	// iswildcard specifies if this element name has wildcards.
	iswildcard bool
	// hasprefixes specifies that one or more subs of this element have
	// prefix names.
	hasprefixes bool
	// haswildcards specifies that one or more subs of this element have
	// wildcards in the name.
	haswildcards bool
}

// newElement returns a new element instance.
func newElement() *element { return &element{subs: make(elements)} }

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

	override     byte // Override is the override character to use. Default: '!'.
	separator    byte // Separator is the path separator character to use. Default: '/'.
	variable     byte // Variable is the variable placeholder character to use. Default: ':'.
	prefix       byte // Prefix is the character that prefix character to use. Default: '+'.
	wildcardone  byte // Wildcardone is the character that matches any one character. Default: '?'.
	wildcardmany byte // WIldcardmany is the character that matches one or more characters. Default: '*';
}

// registerState maintains the template registration state.
type registerState struct {
	template *string
	current  *element
	cursor   int
	marker   int
	length   int
	override bool
	existing bool // existing template was retrieved.
}

// matchState maintains the match state.
type matchState struct {
	current     *element
	path        *string
	matches     *[]string
	vars        *Vars
	length      int
	hasoverride bool
}

// New returns a new *Varouter instance with default configuration.
func New() *Varouter { return NewVarouter(false, '!', '/', ':', '+', '?', '*') }

// NewVarouter returns a new *Varouter instance with the given override,
// separator, placeholder and wildcard character.
func NewVarouter(usewildcards bool, override, separator, variable, prefix, wildcardone, wildcardmany byte) *Varouter {
	return &Varouter{
		root:         newElement(),
		override:     override,
		separator:    separator,
		variable:     variable,
		prefix:       prefix,
		wildcardone:  wildcardone,
		wildcardmany: wildcardmany,
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
// suffix appears as if instead of a name, e.g. "/home/users/+".
//
// Wildcard characters as part of the path element name are legal and registered
// as is and are left to be interpreted by the user. For example:
// "/usr/lib+", "/usr/lib+/bash", "/tests/+_test.go", "/home/users/+/.config".
//
// Template path elements can be defined as Placeholders by prefixing the path
// element with a Placeholder which matches the whole path element as a value
// of the named path element and are returned as a map. For example:
// "/home/users/:user", "/:item/:action/", "/movies/:id/comments/".
//
// Templates can be defined as overrides by prefixing the template with the
// override character. This forces Match to return only one template regardless
// if the path matches multiple templates and it will be an override template.
// If more than one override templates Match a path, the override template with
// the longest prefix wins. More specific matches of templates that are not
// overrides after a matched override template are not considered.
//
// Only one Placeholder per registered template tree path element level is
// allowed. For example:
// "/edit/:user" and "/export/:user" is allowed but
// "/edit/:user" and "/edit/:admin" is not.
func (vr *Varouter) Register(template string) (err error) {
	var state registerState
	state.template = &template
	state.current = vr.root
	if state.length = len(template); state.length < 1 {
		return errors.New("varouter: empty template name")
	}
	for state.cursor = 0; state.cursor < state.length; state.cursor++ {
		if template[state.cursor] == vr.prefix && state.cursor < state.length-1 {
			return errors.New("varouter: prefix character allowed only as suffix")
		}
	}
	state.cursor = 1
	if (*state.template)[0] == vr.override {
		state.override = true
		state.marker++
		state.cursor++
	}
	if (*state.template)[state.marker] != vr.separator {
		return errors.New("varouter: invalid template")
	}
	for ; state.cursor < state.length; state.cursor++ {
		if (*state.template)[state.cursor] != vr.separator {
			continue
		}
		if err = vr.matchOrInsert(&state); err != nil {
			return err
		}
		state.marker = state.cursor
	}
	if err = vr.matchOrInsert(&state); err != nil {
		return err
	}
	if state.existing {
		return errors.New("varouter: template already registered")
	}
	// Mark the last element as override.
	if state.override {
		state.current.isoverride = true
	}
	state.current.template = template
	return nil
}

// matchOrInsert matches a single path element or inserts a new one if it does
// not exist and updates element properties in the process.
func (vr *Varouter) matchOrInsert(state *registerState) error {
	var name = (*state.template)[state.marker:state.cursor]
	var namelen = len(name)
	var prefix = name[namelen-1] == vr.prefix
	if prefix {
		name = name[:namelen-1]
		namelen--
	}
	var elem *element
	var exists bool
	// Try exact match first.
	if elem, exists = state.current.subs[name]; exists {
		// If last element being matched and this registered element
		// is not a template, error out.
		if state.cursor == state.length && state.current.template != "" {
			return errors.New("varouter: template '" + elem.template + "' already registered")
		}
		// Update state and advance to next registered level.
		state.current = elem
		state.existing = true
		return nil
	}
	elem = newElement()
	if prefix {
		state.current.hasprefixes = true
	}
	// Mark as wildcard for match optimization.
	if elem.iswildcard = vr.hasWildcards(&name); elem.iswildcard {
		state.current.haswildcards = true
	}
	// Register as variable.
	if state.current.hasvariable != "" {
		return errors.New("varouter: element registration on a level with a variable")
	}
	if namelen > 1 && name[1] == vr.variable {
		if namelen <= 2 {
			return errors.New("varouter: empty variable name")
		}
		if len(state.current.subs) > 0 {
			return errors.New("varouter: multiple variable registrations on a path level")
		}
		// Technically, wildcards in variable names would work.
		if elem.iswildcard {
			return errors.New("varouter: variable names cannot contain wildcards")
		}
		state.current.hasvariable = name
	}
	// If this is the last element and its last char is a prefix
	// character, set eventual preparsed flags.
	if state.cursor >= state.length {
		elem.isprefix = prefix
		elem.isoverride = state.override
	}
	// Add item.
	state.current.subs[name] = elem
	state.current = elem
	state.existing = false
	vr.count++
	return nil
}

// hasWildcards returns if specified name contains wildcard characters.
func (vr *Varouter) hasWildcards(name *string) bool {
	var i int
	var l = len(*name)
	for i = 0; i < l; i++ {
		if (*name)[i] == vr.wildcardone || (*name)[i] == vr.wildcardmany {
			return true
		}
	}
	return false
}

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
func (vr *Varouter) Match(path string) (matches []string, vars Vars, matched bool) {
	var state = matchState{
		current: vr.root,
		path:    &path,
		length:  len(path),
		matches: &matches,
		vars:    &vars,
	}
	if state.length < 1 {
		return nil, nil, false
	}
	vars = make(Vars)
	vr.nextLevel(0, &state)
	return matches, vars, len(matches) > 0
}

// nextLevel advances matching to the next path level.
func (vr *Varouter) nextLevel(marker int, state *matchState) {
	var cursor int
	for cursor = marker + 1; cursor < state.length; cursor++ {
		if (*state.path)[cursor] != vr.separator {
			continue
		}
		if vr.matchLevel(cursor, marker, state) {
			return
		}
		marker = cursor
	}
	vr.matchLevel(cursor, marker, state)
}

// matchLevel help.
func (vr *Varouter) matchLevel(cursor, marker int, state *matchState) (stop bool) {
	// End of template reached.
	if marker > state.length {
		return true
	}
	// Extract current level name.
	var name string
	if cursor >= state.length {
		name = (*state.path)[marker:]
	} else {
		name = (*state.path)[marker:cursor]
	}
	var namelen = len(name)
	// fmt.Printf("? MatchLevel: Path: '%s', Name: '%s', Cursor: '%d', Marker: '%d', Length: '%d'\n", *state.path, name, cursor, marker, state.length)
	if name == "" {
		return
	}
	// If element is a variable holder, retrieve the sub element by
	// variable name, add the current level name as variable value
	// and advance to next level.
	if state.current.hasvariable != "" {
		(*state.vars)[state.current.hasvariable[2:]] = (*state.path)[marker+1 : cursor]
		state.current = state.current.subs[state.current.hasvariable]
		if state.current.isprefix {
			stop = vr.addMatch(&cursor, state)
		} else {
			stop = vr.maybeAddMatch(&cursor, state)
		}
		if stop {
			return
		}
		cursor += len(name)
		vr.nextLevel(cursor, state)
		return
	}
	// Iterate subs if required.
	var subname string
	var subnamelen int
	var subelem *element
	var saveelem = state.current
	if state.current.haswildcards || state.current.hasprefixes {
		for subname, subelem = range state.current.subs {
			subnamelen = len(subname)
			// Match against any wildcards.
			if subelem.iswildcard && vr.matchWildcard(&name, &subname) {
				state.current = subelem
				vr.maybeAddMatch(&cursor, state)
				vr.nextLevel(cursor, state)
				state.current = saveelem
			}
			// Match agains any prefixes.
			if subelem.isprefix && namelen >= subnamelen && name[0:subnamelen] == subname {
				state.current = subelem
				vr.addMatch(&cursor, state)
				vr.nextLevel(cursor, state)
				state.current = saveelem
			}
		}
	}
	// Finally, try an exact match.
	var exists bool
	if subelem, exists = state.current.subs[name]; exists {
		state.current = subelem
		if vr.maybeAddMatch(&cursor, state) {
			return true
		}
		vr.nextLevel(cursor, state)
		return true
	}
	return true
}

// maybeAddMatch maybe adds the current level to matrches if:
// This level is a prefix template.
// This is the last level being matched and depth matches tested path.
// Matches are replaced if current level is a match and an override.
func (vr *Varouter) maybeAddMatch(cursor *int, state *matchState) (added bool) {
	if state.current.isprefix {
		return false
	}
	// Current level must be the last element of a registered template.
	if state.current.template == "" {
		return false
	}
	// Exit if current level below max level.
	if *cursor < state.length {
		return false
	}
	return vr.addMatch(cursor, state)
}

// addMatch help.
func (vr *Varouter) addMatch(cursor *int, state *matchState) (added bool) {
	// If current match is an override, clear other matches.
	if state.current.isoverride {
		state.hasoverride = true
		if len(*state.matches) > 0 {
			(*state.matches)[0] = state.current.template
			(*state.matches) = (*state.matches)[:1]
			return false
		}
		*state.matches = append(*state.matches, state.current.template)
		return false
	}
	// If there are override matches and current is not override, skip.
	if state.hasoverride && !state.current.isoverride {
		return false
	}
	*state.matches = append(*state.matches, state.current.template)
	if !state.current.isprefix {
		return true
	}
	return false
}

// matchWildcard returns truth if text matches wildcard.
func (vr *Varouter) matchWildcard(text, wildcard *string) bool {
	var lt, lw int = len(*text), len(*wildcard)
	if lt == 0 || lw == 0 {
		return false
	}
	var it, iw int
	for it < lt && iw < lw {
		if (*wildcard)[iw] == vr.wildcardmany {
			break
		}
		if (*wildcard)[iw] != vr.wildcardone && (*text)[it] != (*wildcard)[iw] {
			return false
		}
		it++
		iw++
	}
	var st, sw int = -1, 0
	for it < lt && iw < lw {
		if (*wildcard)[iw] == vr.wildcardmany {
			iw++
			if iw >= lw {
				return true
			}
			sw = iw
			st = it
		} else {
			if (*wildcard)[iw] == vr.wildcardone || (*text)[it] != (*wildcard)[iw] {
				it++
				iw++
			} else {
				it = st
				st++
				iw = sw
			}
		}
	}
	for iw < lw && (*wildcard)[iw] == vr.wildcardmany {
		iw++
	}
	return iw == lw
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
