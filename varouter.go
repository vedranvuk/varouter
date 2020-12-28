// Copyright 2020 Vedran Vuk. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// Package varouter implements a flexible path matching router with support for
// variables and wildcards that does not suffer (greatly) on performance with
// large number of registered items.
package varouter

import (
	"errors"
	"fmt"
)

var (
	// ErrVarouter is the base varouter package error.
	ErrVarouter = errors.New("varouter")

	// ErrRegister is base registration error.
	ErrRegister = fmt.Errorf("%w: register", ErrVarouter)
	// ErrDuplicate is returned when a duplicate template is specified.
	ErrDuplicate = fmt.Errorf("%w: duplicate template", ErrRegister)
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
// (greatly) on large number of registered items.
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
	template *string  // template being registered.
	current  *element // current element being matched against.
	cursor   int      // cursor is the template scan position.
	marker   int      // marker is the position from which an element name is extracted, up to cursor.
	length   int      // length is the length of template.
	override bool     // override denotes template is an override.
	existing bool     // existing template was retrieved.
}

// matchState maintains the path matching state.
type matchState struct {
	current     *element  // current element being matched against.
	path        *string   // path being matched.
	matches     *[]string // matches is a list templates matching path.
	vars        *Vars     // vars hold the extracted variable values.
	length      int       // length is the length of the path.
	hasoverride bool      // hasoverride denotes an override match has been added to matches.
}

// New returns a new *Varouter instance with default configuration.
func New() *Varouter { return NewVarouter(false, '!', '/', ':', '+', '?', '*') }

// NewVarouter returns a new *Varouter instance with the given override,
// separator, variable, prefix, wildcard-one and wildcard-many characters.
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
// A Prefix template which will match a path if it is prefixed by it can be
// defined by adding a Prefix character suffix to the template. For example:
// "/+", "/edit+", "/home/+"
//
// Prefix characters as part of the path element name are not allowed and can
// appear exclusively as a single suffix to the template being registered.
//
// Template path elements can be defined as Variables by prefixing the path
// element with a Variable character which matches the whole path element as a
// value of the named path element. For example:
// "/home/users/:user", "/:item/:action/", "/movies/:id/comments/".
//
// Templates can be defined as Overrides by prefixing the template with the
// override character. This forces Match to return only one template regardless
// if the path matches multiple templates and it will be an override template.
// If more than one override templates Match a path, the override template with
// the longest prefix wins. More specific matches of templates that are not
// overrides after a matched override template are not considered. Override
// characters as part of template name are allowed.
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
		return fmt.Errorf("%w: empty template name", ErrRegister)
	}
	for state.cursor = 0; state.cursor < state.length; state.cursor++ {
		if template[state.cursor] == vr.prefix && state.cursor < state.length-1 {
			return fmt.Errorf("%w: prefix character allowed only as suffix", ErrRegister)
		}
	}
	state.cursor = 1
	if (*state.template)[0] == vr.override {
		state.override = true
		state.marker++
		state.cursor++
	}
	if (*state.template)[state.marker] != vr.separator {
		return fmt.Errorf("%w: invalid template", ErrRegister)
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
		return fmt.Errorf("%w: '%s'", ErrDuplicate, template)
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
func (vr *Varouter) matchOrInsert(state *registerState) (err error) {
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
			return fmt.Errorf("%w: '%s'", ErrDuplicate, elem.template)
		}
		// Update state and advance to next registered level.
		state.current = elem
		state.existing = true
		return nil
	}
	elem = newElement()
	if prefix {
		// Mark parent for match optimization.
		state.current.hasprefixes = true
	}
	if elem.iswildcard = vr.hasWildcards(&name, &namelen); elem.iswildcard {
		// Mark parent for match optimization.
		state.current.haswildcards = true
	}
	// Register as variable.
	if state.current.hasvariable != "" {
		return fmt.Errorf("%w: element registration on a level with a variable", ErrRegister)
	}
	if namelen > 1 && name[1] == vr.variable {
		if err = vr.validateVariableName(&name, &namelen); err != nil {
			return
		}
		if len(state.current.subs) > 0 {
			return fmt.Errorf("%w: multiple variable registrations on a path level", ErrRegister)
		}
		if elem.iswildcard {
			return fmt.Errorf("%w: variable names cannot contain wildcards", ErrRegister)
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
func (vr *Varouter) hasWildcards(name *string, namelen *int) bool {
	for i := 0; i < *namelen; i++ {
		if (*name)[i] == vr.wildcardone || (*name)[i] == vr.wildcardmany {
			return true
		}
	}
	return false
}

// isValidVariableName returns an error if variable name is invalid.
func (vr *Varouter) validateVariableName(name *string, namelen *int) error {
	if *namelen <= 2 {
		return fmt.Errorf("%w: empty variable name", ErrRegister)
	}
	for i := 2; i < *namelen; i++ {
		if (*name)[i] == vr.variable {
			return fmt.Errorf("%w: invalid variable name", ErrRegister)
		}
	}
	return nil
}

// Match matches a path against registered templates and returns the names of
// matched templates, a map of parsed param names to param values and a bool
// indicating if a match occured and previous two result vars are valid.
//
// See Register for details on how the path is matched against templates.
//
// If no templates were matched the resulting templates will be nil.
// If no params were parsed from the path the resulting ParamMap wil be nil.
func (vr *Varouter) Match(path string) (matches []string, vars Vars, matched bool) {
	vars = make(Vars)
	matched = vr.match(&path, &matches, &vars)
	return
}

// MatchTo is a potentially faster version of Match that takes pointers to
// preallocated inputs and outputs. All paramaters must be valid pointers.
// Path is a pointer to a string to match registered templates against.
// Matches is a pointer to a slice that needs to have enough match capacity.
// Vars is a pointer to a map into which parsed variables will be stored into.
// Returns a boolean denoting if anything was matched.
func (vr *Varouter) MatchTo(path *string, matches *[]string, vars *Vars) bool {
	return vr.match(path, matches, vars)
}

// match is the implementation of Match and MatchTo.
func (vr *Varouter) match(path *string, matches *[]string, vars *Vars) bool {
	var state = matchState{
		current: vr.root,
		path:    path,
		length:  len(*path),
		matches: matches,
		vars:    vars,
	}
	if state.length < 1 {
		return false
	}
	vr.nextLevel(0, &state)
	return len(*matches) > 0
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

// matchLevel matches a path level against one or more corresponding registered
// template levels. Result denotes if matching should continue.
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

// maybeAddMatch maybe adds the current level to state.matches if:
// This level is not a prefix template.
// State.current item is the last element of a registered template.
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

// addMatch adds state.current.template to a list of matches.
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

// matchWildcard returns truth if text matches wildcard. Bytescan.
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
