# varouter

## Description

Package varouter implements a flexible path matching router with support for
variables and wildcards that does not suffer on performance with large
number of registered items. It uses maps for path element matching, one per depth level.

Varouter is not a mux, it's a matcher that can be easily adapted to a custom mux that supports any type of Handler.

The additional servemux package demonstrates wrapping varouter into a mux compatible with `http.ServeMux.`

## Example

```Go
vr.Register("/+")
vr.Register("/dir/:var/+")

templates, params, matched := vr.Match("/dir/val/abc")
fmt.Printf("Templates: '%v', Params: '%v', Matched: '%t'\n", templates, params, matched)
// Output: Templates: '[/+ /dir/:var/+]', Params: 'map[var:val]', Matched: 'true'
```

## Features

* Relaxed over Restrictive. Tries to be maximally flexible in the smallest package and API possible.
* Parse tokens are configurable in hope of broadening package use cases.
* Matches are matched exactly but wildcards can be specified in which case multiple matches are possible.
* Overrides can be defined to force single matches.

## Status

Complete. May be further optimized but is quick as it is.

## License

MIT. See included LICENSE file.