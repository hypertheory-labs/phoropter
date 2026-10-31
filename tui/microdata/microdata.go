// Package microdata reads HTML microdata the way the WHATWG spec does, with
// nothing phoropter-specific in it: items, their properties, and the rel
// links on a page.
package microdata

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// Item is an element with itemscope. Phoropter's items have no itemtype, so
// an item is known by where it sits and which properties it has.
type Item struct {
	Node  *html.Node
	ID    string // the element's id, which phoropter uses for entry fragments
	Props map[string][]*Value
}

// Value is one property value. Item is set when the property's element is
// itself an item; otherwise Text is the spec's value (href, content, value,
// datetime, or text content).
type Value struct {
	Node *html.Node
	Text string
	Item *Item
	Rel  []string
}

// Link is any <a> or <link> with a rel, whether or not it is also a property.
type Link struct {
	Node *html.Node
	Rel  []string
	Href string
	Text string
}

// Document is a parsed page: its top-level items, in order, and its rel links.
type Document struct {
	Root  *html.Node
	Title string
	Items []*Item
	Links []Link
}

func Parse(root *html.Node, base *url.URL) *Document {
	d := &Document{Root: root}
	var walk func(n *html.Node, cur *Item)
	walk = func(n *html.Node, cur *Item) {
		next := cur
		if n.Type == html.ElementNode {
			if n.Data == "title" && d.Title == "" {
				d.Title = strings.TrimSpace(TextContent(n))
			}
			if rel := attr(n, "rel"); rel != "" && (n.Data == "a" || n.Data == "link") {
				d.Links = append(d.Links, Link{Node: n, Rel: strings.Fields(rel), Href: resolve(base, attr(n, "href")), Text: Squash(TextContent(n))})
			}
			props := strings.Fields(attr(n, "itemprop"))
			var item *Item
			if hasAttr(n, "itemscope") {
				item = &Item{Node: n, ID: attr(n, "id"), Props: map[string][]*Value{}}
				next = item
				if len(props) == 0 {
					d.Items = append(d.Items, item)
				}
			}
			if len(props) > 0 && cur != nil {
				v := &Value{Node: n, Item: item, Rel: strings.Fields(attr(n, "rel"))}
				if item == nil {
					v.Text = valueOf(n, base)
				}
				for _, p := range props {
					cur.Props[p] = append(cur.Props[p], v)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, next)
		}
	}
	walk(root, nil)
	return d
}

func valueOf(n *html.Node, base *url.URL) string {
	switch n.Data {
	case "meta":
		return attr(n, "content")
	case "a", "area", "link":
		return resolve(base, attr(n, "href"))
	case "img", "audio", "video", "source", "iframe", "embed", "track":
		return resolve(base, attr(n, "src"))
	case "object":
		return resolve(base, attr(n, "data"))
	case "data", "meter":
		return attr(n, "value")
	case "time":
		if hasAttr(n, "datetime") {
			return attr(n, "datetime")
		}
	}
	return TextContent(n)
}

// Get returns the first value of a property, or nil.
func (it *Item) Get(name string) *Value {
	if it == nil || len(it.Props[name]) == 0 {
		return nil
	}
	return it.Props[name][0]
}

// Str is the first value's text, trimmed and with whitespace squashed.
func (it *Item) Str(name string) string {
	if v := it.Get(name); v != nil {
		return Squash(v.Text)
	}
	return ""
}

// Items are the property's values that are items.
func (it *Item) Items(name string) []*Item {
	if it == nil {
		return nil
	}
	var out []*Item
	for _, v := range it.Props[name] {
		if v.Item != nil {
			out = append(out, v.Item)
		}
	}
	return out
}

// Within reports whether n is inside an element with the given tag.
func Within(n *html.Node, tag string) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Data == tag {
			return true
		}
	}
	return false
}

func TextContent(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func Squash(s string) string { return strings.Join(strings.Fields(s), " ") }

func Attr(n *html.Node, key string) string { return attr(n, key) }

func HasClass(n *html.Node, class string) bool {
	for _, c := range strings.Fields(attr(n, "class")) {
		if c == class {
			return true
		}
	}
	return false
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasAttr(n *html.Node, key string) bool {
	for _, a := range n.Attr {
		if a.Key == key {
			return true
		}
	}
	return false
}

func resolve(base *url.URL, ref string) string {
	if base == nil || ref == "" {
		return ref
	}
	u, err := base.Parse(ref)
	if err != nil {
		return ref
	}
	return u.String()
}
