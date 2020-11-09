# varouter

## Description

Package varouter implements a flexible path matching router with support for
variables and wildcards that does not suffer on performance with large
number of registered items. It uses maps for path element matching, one per depth level.

Varouter is not a mux, it's a matcher that can be easily adapted to a custom mux that supports any type of Handler.

The additional servemux package demonstrates wrapping varouter into a mux compatible with `http.ServeMux.`

## Example

```Go
vr := New()
vr.Register("/+")
vr.Register("/home/:username/+")

templates, params, matched := vr.Match("/home/vedran/.config")

fmt.Printf("Templates: '%v', Params: '%v', Matched: '%t'\n", templates, params, matched)
// Output: Templates: '[/+ /home/:username/+]', Params: 'map[username:vedran]', Matched: 'true'
```

## Features

* Relaxed over Restrictive. Tries to be maximally flexible in the smallest package and API possible.
* Parse tokens are configurable in hope of broadening package use cases.
* Matches are matched exactly but wildcards can be specified in which case multiple matches are possible.
* Overrides can be defined to force single matches.

## Status

Work in progress.

* API _could_ still change, but really, not by a lot.
* Will add possibility to register multiple wildcards from a single template.
* Requires further testing and few bugs to remove.

## License

MIT. See included LICENSE file.