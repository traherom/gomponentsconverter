# About
This tool converts from plain HTML to Gomponents (https://github.com/maragudk/gomponents), a great Go library for generating static HTML in a more type-safe way than the default `html/templates`. It's usable as both a server (with an HTML or an API, if you so choose) and as a CLI.

A hosted version can be found at [https://gomponents.morehart.dev](https://gomponents.morehart.dev).

# Server
  ```
  go run github.com/traherom/gomponentsconverter/cmd -- serve --port=3333
  ```

# CLI usage
  ```
  echo '<div>ioj</div>' | go run github.com/traherom/gomponentsconverter/cmd -- convert
  ```
