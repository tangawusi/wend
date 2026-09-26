package normalize

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
)

var blockElements = map[string]bool{
	"p": true, "h1": true, "h2": true, "h3": true, "h4": true,
	"h5": true, "h6": true, "li": true, "blockquote": true,
	"pre": true, "figcaption": true, "dd": true, "dt": true,
}

var noiseElements = map[string]bool{
	"script": true, "style": true, "nav": true, "header": true,
	"footer": true, "aside": true, "form": true, "iframe": true,
	"noscript": true, "svg": true, "template": true, "button": true,
	"select": true, "option": true, "textarea": true,
}

var noiseTokens = []string{
	"nav", "menu", "sidebar", "side-bar", "footer", "header",
	"comment", "related", "share", "social", "promo", "advert",
	"ad-", "-ad", "sponsor", "cookie", "subscribe", "newsletter",
	"breadcrumb", "pagination", "toolbar", "sharebar", "byline-bar",
	"masthead", "site-header", "site-footer", "skip-link", "popup",
	"modal", "overlay", "paywall", "meter", "utility",
}

const MinParagraphRunes = 25

func ExtractBody(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	doc, err := html.Parse(bytes.NewReader(raw))
	if err != nil {
		return ""
	}
	root := findContentRoot(doc)
	if root == nil {
		return ""
	}
	stripNoise(root)

	var blocks []string
	walkBlocks(root, &blocks)
	return joinBlocks(blocks)
}

func findContentRoot(doc *html.Node) *html.Node {
	if n := firstByTag(doc, "article"); n != nil && hasSubstantialText(n) {
		return n
	}
	if n := firstByTag(doc, "main"); n != nil && hasSubstantialText(n) {
		return n
	}
	if n := firstByAttr(doc, "role", "main"); n != nil && hasSubstantialText(n) {
		return n
	}
	return densestParagraphNode(doc)
}

func firstByTag(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := firstByTag(c, tag); found != nil {
			return found
		}
	}
	return nil
}

func firstByAttr(n *html.Node, key, val string) *html.Node {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == key && a.Val == val {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := firstByAttr(c, key, val); found != nil {
			return found
		}
	}
	return nil
}

func hasSubstantialText(n *html.Node) bool {
	return len([]rune(textOf(n))) >= 4*MinParagraphRunes
}

func densestParagraphNode(doc *html.Node) *html.Node {
	var best *html.Node
	var bestLen int

	var visit func(n *html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "div" || n.Data == "section") {
			if total := paragraphTextLen(n); total > bestLen {
				bestLen = total
				best = n
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)

	if bestLen < 4*MinParagraphRunes {
		return nil
	}
	return best
}

func paragraphTextLen(n *html.Node) int {
	total := 0
	var visit func(*html.Node)
	visit = func(x *html.Node) {
		if x.Type == html.ElementNode && blockElements[x.Data] {
			total += len([]rune(textOf(x)))
			return
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(n)
	return total
}

func stripNoise(n *html.Node) {
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		if isNoise(c) {
			n.RemoveChild(c)
		} else {
			stripNoise(c)
		}
		c = next
	}
}

func isNoise(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	if noiseElements[n.Data] {
		return true
	}
	for _, a := range n.Attr {
		if a.Key == "role" {
			switch a.Val {
			case "navigation", "banner", "complementary", "contentinfo", "search":
				return true
			}
		}
		if a.Key == "class" || a.Key == "id" {
			lower := strings.ToLower(a.Val)
			for _, tok := range noiseTokens {
				if strings.Contains(lower, tok) {
					return true
				}
			}
		}
		if a.Key == "aria-hidden" && a.Val == "true" {
			return true
		}
	}
	return false
}

func walkBlocks(n *html.Node, out *[]string) {
	if n.Type == html.ElementNode && blockElements[n.Data] {
		text := CleanText(textOf(n))
		if len([]rune(text)) >= MinParagraphRunes {
			*out = append(*out, text)
		}
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkBlocks(c, out)
	}
}

func textOf(n *html.Node) string {
	var b strings.Builder
	var visit func(*html.Node)
	visit = func(x *html.Node) {
		if x.Type == html.TextNode {
			if b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(x.Data)
			return
		}
		if x.Type == html.ElementNode && (x.Data == "br" || x.Data == "hr") {
			b.WriteByte(' ')
			return
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(n)
	return b.String()
}
