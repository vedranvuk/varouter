# varouter

## Description

Package varouter implements a flexible path matching router with support for
variables and wildcards that does not suffer on performance with large
number of registered items.

Varouter is not a mux, it's a matcher that can be easily adapted to a custom mux that supports any type of Handler.

The additional servemux package demonstrates wrapping varouter into a mux compatible with `http.ServeMux.`

## Example

```Go
vr := New()
vr.Register("/*")
vr.Register("/home/users/:username/*")

templates, params, matched := vr.Match("/home/users/vedran/.config")

fmt.Printf("Templates: '%v', Params: '%v', Matched: '%t'\n", templates, params, matched)
// Output: Templates: '[/* /home/users/:username/* /home/users/:username/*]', Params: 'map[username:vedran]', Matched: 'true'
```

## Status

Work in progress.

* API is fixed, only additions possible. 
* Will add possibility to register multiple wildcards from a single template.
* Should change default wildcard character to something other than '*' to not interfere with standard wildcards as the're allowed as parts of registered names.
* Requires further testing.

## License

MIT. See included LICENSE file.