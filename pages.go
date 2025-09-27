package gomponentsconverter

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"maragu.dev/gomponents/components"
)

//go:embed convert.js
var convertJS string

//go:embed main.css
var mainCSS string

const example = `<ul class="block w-11/12 my-4 mx-auto" x-data="{selected:null}">
    <li class="flex align-center flex-col">
        <h4 @click="selected !== 0 ? selected = 0 : selected = null"
            class="cursor-pointer px-5 py-3 bg-indigo-300 text-white text-center inline-block hover:opacity-75 hover:shadow hover:-mb-3 rounded-t">Accordion item 1</h4>
        <p x-show="selected == 0" class="border py-4 px-2">
            This is made with Alpine JS and Tailwind CSS
        </p>
    </li>
    <li class="flex align-center flex-col">
        <h4 @click="selected !== 1 ? selected = 1 : selected = null"
            class="cursor-pointer px-5 py-3 bg-indigo-400 text-white text-center inline-block hover:opacity-75 hover:shadow hover:-mb-3">Accordion item 2</h4>
        <p x-show="selected == 1" class="border py-4 px-2">
              There's no external CSS or JS
        </p>
    </li>
    <li class="flex align-center flex-col">
        <h4 @click="selected !== 2 ? selected = 2 : selected = null"
            :class="{'cursor-pointer px-5 py-3 bg-indigo-500 text-white text-center inline-block hover:opacity-75 hover:shadow hover:-mb-3': true, 'rounded-b': selected != 2}">Accordion item 3</h4>
        <p x-show="selected == 2" :class="{'border py-4 px-2': true, 'rounded-b': selected == 2}">
            Pretty cool huh?
        </p>
    </li>
</ul>`

func IndexHandler(ctx context.Context) http.HandlerFunc {
	convertedExample, err := ConvertString(example)
	if err != nil {
		slog.Error("Converting example failed", "error", err)
	}

	cloakDirectives := `[x-cloak], [v-cloak] { display: none !important; }`

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		year := time.Now().Format("2006")
		html := components.HTML5(components.HTML5Props{
			Title:       "Gomponents Converter",
			Description: "Convert raw HTML to Gomponents",
			Language:    "en",
			Head: []Node{
				StyleEl(Raw(cloakDirectives)),
				//Script(Src(MustJoin(prefix, "alpinetrap@3.x.x.js")), Defer()),
				Script(Src("https://cdn.jsdelivr.net/npm/@alpinejs/persist@3.x.x/dist/cdn.min.js"), Defer()),
				//Script(Src(MustJoin(prefix, "alpinecollapse@3.x.x.js")), Defer()),
				//Script(Src(MustJoin(prefix, "alpinemorph@3.x.x.js")), Defer()),
				Script(Src("https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js"), Defer()),
				StyleEl(Raw(mainCSS)),
				Script(Rawf("const initialInput = %q;\nconst initialOutput = %q;", example, convertedExample)),
				Script(Raw(convertJS)),
			},
			Body: []Node{
				Div(
					ID("wrapper"),
					Attr("x-data", "converter"),
					Div(
						ID("header"),
						H1(Text("HTML to Gomponents")),
						Div(
							Text("Convert raw HTML to "),
							A(Href("https://www.gomponents.com"), Text("Gomponents")),
							Text(", a component-based templating library for Go."),
						),
					),
					Div(
						ID("inout"),
						Label(
							For("input"),
							Class("sr-only"),
							Text("HTML input to convert"),
						),
						Textarea(
							ID("input"),
							Placeholder(`<div>html here</div>`),
							Attr("x-model", "input"),
							Attr("@input.debounce", "convert()"),
						),
						Div(
							ID("outwrapper"),
							Div(
								ID("settings"),
								Div(
									Label(
										For("prefixhelper"),
										Text("HTML prefix"),
									),
									Input(
										Type("text"),
										ID("prefixhelper"),
										Placeholder("h"),
										Attr("x-model", "settings.prefix_html"),
										Attr("@input.debounce", "convert()"),
									),
								),
								Div(
									Label(
										For("prefixcore"),
										Text("Core prefix"),
									),
									Input(
										Type("text"),
										ID("prefixcore"),
										Placeholder("g"),
										Attr("x-model", "settings.prefix_core"),
										Attr("@input.debounce", "convert()"),
									),
								),
							),
							Label(
								For("imports"),
								Class("sr-only"),
								Text("Import statements to import Gomponents with the specified prefixes"),
							),
							Textarea(
								ID("imports"),
								ReadOnly(),
								Attr("x-cloak"),
								Attr("x-on:click", "copyImports()"),
								Attr("x-model", "imports"),
							),
							Div(
								ID("error"),
								Attr("x-cloak"),
								Attr("x-show", "conversion_error"),
								Attr("x-text", "conversion_error"),
							),
							Label(
								For("output"),
								Class("sr-only"),
								Text("Generated Gomponents Go code"),
							),
							Textarea(
								ID("output"),
								ReadOnly(),
								Attr("x-model", "output"),
								Attr("x-on:click", "copyOutput()"),
								Attr("x-transition"),
							),
						),
					),
					Div(
						ID("copiedNotice"),
						Attr("x-cloak"),
						Attr("x-transition.scale.origin.bottom"),
						Attr("x-show", "show_copied"),
						Div(
							Attr("x-text", "copied_text"),
						),
					),
					Div(
						ID("footer"),
						Div(
							Textf("Tool © %s Ryan Morehart, hosting sponsored by ", year),
							A(Href("https://xylok.io"), Text("Xylok")),
							Text("."),
						),
					),
				),
			},
		})

		ctx := r.Context()
		err = html.Render(w)
		if err != nil {
			slog.ErrorContext(ctx, "Page failed to render", "error", err)
		}
	})
}

func ConvertHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > 100000 {
			slog.ErrorContext(r.Context(), "Body is too long, dropping", "size", r.ContentLength)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		input := struct {
			Data       string `json:"data"`
			PrefixCore string `json:"prefix_core"`
			PrefixHTML string `json:"prefix_html"`
		}{}
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&input)
		if err != nil {
			slog.ErrorContext(r.Context(), "Failed to read body", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		settings := Config{
			PreserveWhitespace: false,
			PrefixHTML:         input.PrefixHTML,
			PrefixCore:         input.PrefixCore,
		}
		out, err := ConvertConfig(settings, bytes.NewBufferString(input.Data))

		output := struct {
			Converted string `json:"converted"`
			Error     string `json:"error"`
		}{
			Converted: out,
		}

		if err != nil {
			output.Error = err.Error()
		}

		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		err = encoder.Encode(output)
		if err != nil {
			slog.ErrorContext(r.Context(), "Failed to marshal response", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
}
