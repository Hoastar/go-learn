package main

import (
	"fmt"
	"sync"
)

type Node struct {
	value int
}

type Graph struct {
	nodes []*Node          // 节点集
	edges map[Node][]*Node // 邻接表表示的无向图
	lock  sync.RWMutex     // 保证线程安全
}
// AddNode 增加结点
func (g *Graph) AddNode(node *Node) {
    g.lock.Lock()
    defer g.lock.Unlock()
    g.nodes = append(g.nodes, node)
}

// AddEdge 增加边
func (g *Graph) AddEdge(u, v *Node) {
    g.lock.Lock()
    defer g.lock.Unlock()
    // 首次建立图
    if g.edges = nil {
        g.edges = make(map[Node][]*Node)
    }

    g.edges[*u] = append(g.edges[*u], v)
    g.edges[*v] = append(g.edges[*v], u)
}
// 输出节点
func (node *Node) String() string {
    return fmt.Sprintf("%v", node.value)
}

// 输出图
func (g *Graph) String() {
    g.lock.RLock()
    defer g.lock.RUnlock()
    str := ""
    for _, iNode := range g.nodes {
        str += iNode.String() + " -> "
        nexts := g.edges[*iNode]
        for _, next := range nexts {
            str += next.String() + " "
        }
        str += "\n"
    }
    fmt.Println(str)
}

func main() {
	g := Graph{}
	n1, n2, n3, n4, n5 := Node{1}, Node{2}, Node{3}, Node{4}, Node{5}

	g.AddNode(&n1)
	g.AddNode(&n2)
	g.AddNode(&n3)
	g.AddNode(&n4)
	g.AddNode(&n5)

	g.AddEdge(&n1, &n2)
	g.AddEdge(&n1, &n5)
	g.AddEdge(&n2, &n3)
	g.AddEdge(&n2, &n4)
	g.AddEdge(&n2, &n5)
	g.AddEdge(&n3, &n4)
	g.AddEdge(&n4, &n5)

    g.String()
}