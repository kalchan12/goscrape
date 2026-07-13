package tree

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	st := New("/")
	assert.Equal(t, "/", st.Root.Name)
	assert.Equal(t, "/", st.Root.Path)

	st = New("/exams")
	assert.Equal(t, "/exams", st.Root.Name)
	assert.Equal(t, "/exams", st.Root.Path)
}

func TestBuildTree(t *testing.T) {
	st := New("/")
	st.urls = map[string]int{
		"/":      5,
		"/about": 2,
		"/exams/computer-science": 3,
		"/exams/math":            1,
		"/exams/computer-science/2024": 4,
	}
	st.buildTree()

	paths := st.Paths()
	assert.Contains(t, paths, "/about")
	assert.Contains(t, paths, "/exams/computer-science")
	assert.Contains(t, paths, "/exams/math")
	assert.Contains(t, paths, "/exams/computer-science/2024")
}

func TestRender(t *testing.T) {
	st := New("/")
	st.urls = map[string]int{
		"/":         1,
		"/about":    2,
		"/contact":  1,
	}
	st.buildTree()

	output := st.Render()
	assert.Contains(t, output, "about")
	assert.Contains(t, output, "contact")
	assert.Contains(t, output, "├──")
	assert.Contains(t, output, "└──")
}

func TestRenderEmpty(t *testing.T) {
	st := New("/")
	output := st.Render()
	assert.Empty(t, strings.TrimSpace(output))
}

func TestRenderWithCounts(t *testing.T) {
	st := New("/")
	st.urls = map[string]int{
		"/about": 5,
	}
	st.buildTree()

	output := st.Render()
	assert.Contains(t, output, "(5)")
}

func TestPaths(t *testing.T) {
	st := New("/")
	st.urls = map[string]int{
		"/":             1,
		"/a/b/c":        1,
		"/a/b":          1,
	}
	st.buildTree()

	paths := st.Paths()
	assert.Contains(t, paths, "/a/b/c")
	assert.Contains(t, paths, "/a/b")
}

func TestJoinPath(t *testing.T) {
	assert.Equal(t, "/child", joinPath("/", "child"))
	assert.Equal(t, "/parent/child", joinPath("/parent", "child"))
}

func TestCollectPaths(t *testing.T) {
	root := &Node{Name: "/", Path: "/"}
	child1 := &Node{Name: "a", Path: "/a"}
	child2 := &Node{Name: "b", Path: "/b"}
	grandchild := &Node{Name: "c", Path: "/a/c"}
	child1.Children = append(child1.Children, grandchild)
	root.Children = append(root.Children, child1, child2)

	st := &SiteTree{Root: root}
	paths := st.Paths()
	assert.Contains(t, paths, "/a")
	assert.Contains(t, paths, "/b")
	assert.Contains(t, paths, "/a/c")
}
