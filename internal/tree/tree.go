package tree

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
)

type Node struct {
	Name     string
	Path     string
	Children []*Node
	Count    int
}

type SiteTree struct {
	Root *Node
	urls map[string]int
	mu   sync.Mutex
}

func New(seedPath string) *SiteTree {
	rootName := "/"
	if seedPath != "" && seedPath != "/" {
		rootName = seedPath
	}
	return &SiteTree{
		Root: &Node{Name: rootName, Path: seedPath},
		urls: make(map[string]int),
	}
}

func (st *SiteTree) Crawl(rawURL string, maxDepth int) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	seedPath := parsed.Path
	if seedPath == "" {
		seedPath = "/"
	}

	c := colly.NewCollector(
		colly.MaxDepth(maxDepth),
		colly.Async(true),
		colly.UserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36"),
	)
	c.SetRequestTimeout(60 * time.Second)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 3,
	})

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	})

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("href"))
		if link == "" {
			return
		}
		linkParsed, err := url.Parse(link)
		if err != nil || linkParsed.Host != parsed.Host {
			return
		}

		path := linkParsed.Path
		if path == "" {
			path = "/"
		}
		if path == seedPath {
			return
		}

		st.mu.Lock()
		st.urls[path]++
		st.mu.Unlock()

		e.Request.Visit(link)
	})

	c.OnError(func(r *colly.Response, err error) {
		// ignore errors during tree discovery
	})

	if err := c.Visit(rawURL); err != nil {
		return err
	}
	c.Wait()

	st.buildTree()
	return nil
}

func (st *SiteTree) buildTree() {
	paths := make([]string, 0, len(st.urls))
	for p := range st.urls {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		node := st.Root
		parts := strings.Split(strings.Trim(p, "/"), "/")
		for _, part := range parts {
			if part == "" {
				continue
			}
			found := false
			for _, child := range node.Children {
				if child.Name == part {
					node = child
					found = true
					break
				}
			}
			if !found {
				child := &Node{
					Name: part,
					Path: joinPath(node.Path, part),
				}
				node.Children = append(node.Children, child)
				node = child
			}
		}
		node.Count = st.urls[p]
	}
}

func joinPath(parent, child string) string {
	if parent == "/" {
		return "/" + child
	}
	return parent + "/" + child
}

func (st *SiteTree) Render() string {
	var b strings.Builder
	st.renderNode(st.Root, &b)
	return b.String()
}

func (st *SiteTree) renderNode(node *Node, b *strings.Builder) {
	for i, child := range node.Children {
		isLast := i == len(node.Children)-1
		if isLast {
			b.WriteString("└── ")
		} else {
			b.WriteString("├── ")
		}

		b.WriteString(child.Name)
		if child.Count > 0 {
			b.WriteString(fmt.Sprintf("  (%d)", child.Count))
		}
		b.WriteString("\n")

		ext := "    "
		if !isLast {
			ext = "│   "
		}
		st.renderChildren(child, ext, b)
	}
}

func (st *SiteTree) renderChildren(node *Node, prefix string, b *strings.Builder) {
	for i, child := range node.Children {
		isLast := i == len(node.Children)-1
		if isLast {
			b.WriteString(prefix + "└── ")
		} else {
			b.WriteString(prefix + "├── ")
		}

		b.WriteString(child.Name)
		if child.Count > 0 {
			b.WriteString(fmt.Sprintf("  (%d)", child.Count))
		}
		b.WriteString("\n")

		ext := prefix + "    "
		if !isLast {
			ext = prefix + "│   "
		}
		st.renderChildren(child, ext, b)
	}
}

func (st *SiteTree) Paths() []string {
	var paths []string
	collectPaths(st.Root, &paths)
	sort.Strings(paths)
	return paths
}

func collectPaths(node *Node, out *[]string) {
	for _, child := range node.Children {
		*out = append(*out, child.Path)
		collectPaths(child, out)
	}
}
