package main

import (
	"fmt"
	"sync"
)

// Node 结点
type Node struct {
	value int
}

// NodeQueue 结点队列
type NodeQueue struct {
	nodes []Node
	lock  sync.RWMutex
}

// Graph 图 数据结构
type Graph struct {
	nodes []*Node          // 节点集
	edges map[Node][]*Node // 邻接表表示的无向图
	lock  sync.RWMutex     // 保证线程安全
}

// AddNode 增加结点（顶点）
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
	if g.edges == nil {
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

// NewNodeQueue 生成结点队列
func NewNodeQueue() *NodeQueue {
	q := NodeQueue{}
	q.lock.Lock()
	defer q.lock.Unlock()

	// 空切片
	q.nodes = []Node{}
	return &q
}

// Enqueue 入队
func (q *NodeQueue) Enqueue(node Node) {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.nodes = append(q.nodes, node)
}

// Dequeue 出队
func (q *NodeQueue) Dequeue() *Node {
	q.lock.Lock()
	defer q.lock.Unlock()
	node := q.nodes[0]
	q.nodes = q.nodes[1:]
	return &node
}

// IsEmpty 判空
func (q *NodeQueue) IsEmpty() bool {
	q.lock.Lock()
	defer q.lock.Unlock()
	return len(q.nodes) == 0
}

// BFS 实现 BFS 遍历
func (g *Graph) BFS(f func(node *Node)) {
	g.lock.RLock()
	defer g.lock.RUnlock()

	// 初始化队列
	q := NewNodeQueue()
	// 取图的第一个结点入队
	head := q.nodes[0]
	q.Enqueue(head)
	// 标识结点是否已被访问
	visited := make(map[Node]bool)
	visited[head] = true

	// 遍历所有结点直到队列为空
	for {
		if q.IsEmpty() {
			break
		}

		node := q.Dequeue()
		visited[*node] = true
		nexts := g.edges[*node]

		// 将所有未访问的邻结点入队列
		for _, next := range nexts {
			// 如果结点被访问过
			if visited[*next] {
				continue
			}

			q.Enqueue(*next)
			visited[*next] = true
		}

		// 对每个正在遍历的结点进行回调
		if f != nil {
			f(node)
		}
	}
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

	fmt.Print(g.nodes[0])
	g.BFS(func(node *Node) {
		fmt.Printf("[Current Traverse Node]: %v\n", node)
	}(g.nodes[0]))
}
