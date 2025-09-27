// Copyright © 2025 Ryan Morehart
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
package gomponentsconverter

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io"
	"strings"

	"golang.org/x/net/html"
)

var ErrNothingFound = errors.New("nothing found to convert")

type Config struct {
	// Should whitespace in the original HTML be preserved in the output?
	PreserveWhitespace bool
	// Prefix calls to helpers functions from gomponents/html
	// Period should be included, ie "g."
	PrefixHTML string
	// Prefix calls to the core Gomponents functions (El, Attr, Text)
	// Period should be included, ie "gomponents."
	PrefixCore string
}

// Count number of children of HTML node
func childLen(node *html.Node) int {
	x := 0
	for range node.ChildNodes() {
		x++
	}
	return x
}

// Escape double quotes in a string
func quoteString(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}

func nodeToGomponent(config *Config, node *html.Node, indent string) (string, error) {
	full := strings.Builder{}

	switch node.Type {
	case html.ElementNode:
		c, err := elementToGomponent(config, node, indent)
		if err != nil {
			return "", err
		}

		full.WriteString(c)
	case html.TextNode:
		text := node.Data
		if !config.PreserveWhitespace {
			// Skip text which is just whitespace
			text = strings.TrimSpace(node.Data)
			if text == "" {
				return "", nil
			}
		}

		full.WriteString(indent)
		full.WriteString(config.PrefixCore)
		full.WriteString("Text(")
		full.WriteString(fmt.Sprintf("%q", text))
		full.WriteString(")")
	case html.CommentNode:
		lines := strings.Split(node.Data, "\n")
		for i, line := range lines {
			full.WriteString(indent)
			full.WriteString("// ")
			full.WriteString(strings.TrimSpace(line))
			if i < len(lines)-1 {
				full.WriteString("\n")
			}
		}
	default:
		// Ignore other types
	}

	return full.String(), nil
}

func elementToGomponent(config *Config, el *html.Node, indent string) (string, error) {
	if el.Type != html.ElementNode {
		return "", errors.New("top-level node for conversion must be an element")
	}

	childIndent := "\t" + indent
	hasChildren := len(el.Attr)+childLen(el) > 0

	full := strings.Builder{}
	full.WriteString(indent)

	name := el.Data
	if fName, ok := knownEls[name]; ok {
		full.WriteString(config.PrefixHTML)
		full.WriteString(fName)
		full.WriteString("(")
		if hasChildren {
			full.WriteString("\n")
		}
	} else {
		full.WriteString(config.PrefixCore)
		full.WriteString("El(\"")
		full.WriteString(quoteString(name))
		full.WriteString("\"")
		if hasChildren {
			full.WriteString(",\n")
		}
	}

	for _, attr := range el.Attr {
		full.WriteString(childIndent)
		if fName, ok := knownAttrs[attr.Key]; ok {
			full.WriteString(config.PrefixHTML)
			full.WriteString(fName)
			full.WriteString("(")
			if attr.Val != "" {
				full.WriteString(fmt.Sprintf("%q", attr.Val))
			}
		} else if strings.HasPrefix(attr.Key, "aria-") {
			ariaName := strings.TrimPrefix(attr.Key, "aria-")
			full.WriteString(config.PrefixHTML)
			full.WriteString("Aria(\"")
			full.WriteString(quoteString(ariaName))
			full.WriteString("\"")
			if attr.Val != "" {
				full.WriteString(fmt.Sprintf(", %q", attr.Val))
			}
		} else if strings.HasPrefix(attr.Key, "data-") {
			dataName := strings.TrimPrefix(attr.Key, "data-")
			full.WriteString(config.PrefixHTML)
			full.WriteString("Data(\"")
			full.WriteString(quoteString(dataName))
			full.WriteString("\"")
			if attr.Val != "" {
				full.WriteString(fmt.Sprintf(", %q", attr.Val))
			}
		} else {
			full.WriteString(config.PrefixCore)
			full.WriteString("Attr(\"")
			full.WriteString(quoteString(attr.Key))
			full.WriteString("\"")
			if attr.Val != "" {
				full.WriteString(fmt.Sprintf(", %q", attr.Val))
			}
		}

		full.WriteString("),\n")
	}

	for child := range el.ChildNodes() {
		childText, err := nodeToGomponent(config, child, childIndent)
		if err != nil {
			return full.String(), err
		}

		full.WriteString(childText)
		switch child.Type {
		case html.ElementNode, html.TextNode:
			if childText != "" {
				full.WriteString(",\n")
			}
		case html.CommentNode:
			full.WriteString("\n")
		default:
			// No other special handling needed
		}
	}

	if hasChildren {
		full.WriteString(indent)
	}
	full.WriteString(")")
	return full.String(), nil
}

func findTag(node *html.Node, tagName string) *html.Node {
	if node.Type == html.ElementNode && node.Data == tagName {
		return node
	}

	for d := range node.Descendants() {
		if d.Type == html.ElementNode && d.Data == tagName {
			return d
		}
	}

	return nil
}

func ensurePeriod(in string) string {
	if len(in) > 0 && in[len(in)-1] != '.' {
		return in + "."
	}

	return in
}

func ConvertString(s string) (string, error) {
	r := bytes.NewBufferString(s)
	return Convert(r)
}

func Convert(r io.Reader) (string, error) {
	return ConvertConfig(Config{}, r)
}

// Wrap input in body tag. We try this if we don't find anything to render the first try
func wrapInBody(in []byte) []byte {
	out := append([]byte("<body>"), in...)
	out = append(out, []byte("</body>")...)
	return out
}

func tryConvert(config Config, targetTag string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", nil
	}

	root, err := html.Parse(bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}

	// Limit conversion?
	if targetTag != "" {
		root = findTag(root, targetTag)
		if root == nil {
			return "", fmt.Errorf("failed to find %q element", targetTag)
		}
	}

	nodes := make([]*html.Node, 0, childLen(root))
	for child := range root.ChildNodes() {
		nodes = append(nodes, child)
	}

	// Yes, go/format will take care of indenting later, but
	// I wrote this with built-in indenting originally. Now it's nice for debugging
	initialIndent := ""
	if len(nodes) > 1 {
		initialIndent = "\t"
	}

	gomps := make([]string, 0, len(nodes))
	for _, child := range nodes {
		gomp, err := nodeToGomponent(&config, child, initialIndent)
		if err != nil {
			return "", err
		}

		if gomp == "" {
			continue
		}

		gomps = append(gomps, gomp)
	}

	var output string
	switch len(gomps) {
	case 0:
		output = "// nothing found"
		return output, ErrNothingFound
	case 1:
		output = gomps[0]
	default:
		output = fmt.Sprintf("%sGroup{\n%s,\n}", config.PrefixCore, strings.Join(gomps, ",\n"))
	}

	// Ensure the wrapper body tag doesn't appear
	output = strings.TrimSuffix(output, "</body>")

	formatted, err := format.Source([]byte(output))
	if err != nil {
		// Failed, but still return the output just in case it's useful
		return output, fmt.Errorf("formatting failed: %w", err)
	}

	return string(formatted), nil
}

func ConvertConfig(config Config, r io.Reader) (string, error) {
	config.PrefixCore = ensurePeriod(config.PrefixCore)
	config.PrefixHTML = ensurePeriod(config.PrefixHTML)

	fullDoc, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	fullDoc = bytes.TrimSpace(fullDoc)
	if len(fullDoc) == 0 {
		return "", nil
	}

	isFragment := false
	if !bytes.Contains(fullDoc, []byte("<html")) &&
		!bytes.Contains(fullDoc, []byte("<head")) &&
		!bytes.Contains(fullDoc, []byte("<body")) {
		isFragment = true
		fullDoc = wrapInBody(fullDoc)
	}

	var converted string
	if isFragment {
		converted, err = tryConvert(config, "body", fullDoc)
	} else {
		converted, err = tryConvert(config, "", fullDoc)
	}

	return converted, err
}
