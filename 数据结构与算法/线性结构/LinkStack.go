package main

import (
	"fmt"
)

// Node 结点
type Node struct {
	data interface{}
	next *Node
}

// LinkStack 链式栈
type LinkStack struct {
	head   *Node
	length int
}

// 创建结点
func NewNode(newVal interface{}) *Node {
	return &Node{
		newVal,
		nil,
	}
}

// 创建链表，初始化一个空头结点
func NewLinkStack() *LinkStack {
	return &LinkStack{
		&Node{
			nil,
			nil,
		},
		0,
	}
}

// 判断栈是否为空
func (stack *LinkStack) StackEmpty() bool {
	// 空链表的头指针指向Null
	return stack.head.next == nil
}

// 压栈操作
func (stack *LinkStack) Push(data interface{}) {
	PushNode := NewNode(data)
	PushNode.next = stack.head.next
	stack.head.next = PushNode
	stack.length++
}

// 出栈操作
func (stack *LinkStack) Pop() interface{} {

	//if stack.head.next == nil {
	//	return -1
	//}
	stack.StackEmpty()

	PopNode := stack.head.next
	data := PopNode.data
	stack.head.next = PopNode.next
	PopNode = nil
	stack.length--
	return data
}

// 获取链栈顶元素
func (stack *LinkStack) Top() interface{} {
	stack.StackEmpty()
	//if stack.head.next == nil {
	//	return -1
	//}

	data := stack.head.next.data
	return data
}

func main() {
	stack := NewLinkStack()
	var (
		num  int    = 8
		str  string = "hi,harmonyos"
		yes  bool   = true
		num2 int    = 0
	)
	stack.Push(num)
	stack.Push(str)
	stack.Push(yes)
	stack.Push(num2)

	fmt.Printf("%d\n", stack.Top())
	stack.Pop()
	fmt.Printf("%v\n", stack.Top())
	stack.Pop()
	fmt.Printf("%v\n", stack.Top())
}
