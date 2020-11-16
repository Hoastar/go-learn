package ArrayList

import (
	"fmt"
	"errors"
)

// ArrayList 线性表数据结构
type ArrayList struct {
	data []interface{}
	Last int
}

// List 接口定义
type List interface {
	// 获取线性表大小
	Size() int

	// 获取
	Get(index int) (interface{}, error)
	
	// 修改
	Set(index int, newVal interface{}) error

	// 插入
	Insert(index int, newVal interface{}) error

	// 追加
	Append(newVal interface{})

	// 清空
	Clear()

	// 删除
	Delete(index int) error
}

// NewArrayList 初始化 ArrayList 类型的实例
func NewArrayList() *ArrayList {
	// 初始化结构体，len为0，cap为10
	list := new(ArrayList)
	// 开辟内存空间
	list.data := make([]interface{}, 0, 10)

	list.Last = 0

	return list
}

// 获取数组大小
func (list *ArrayList) Size() int {
	return list.Last
}

// 判断切片容量
func (list *ArrayList) checkIsFull() {
	if list.Last == cap(list.data) {
		// 开辟两倍内存空间，args2 为切片长度，args3 为切片容量
		newData := make([]interface{}, 2*list.Last, 2*list.Last)

		// 拷贝内容
		copy(newData, list.data)

		// 重新赋值
		list.data = newData
	}
}


// 获取数据
func (list *ArrayList) Get(index int) interface{} {
	// 下标无效，越界
	if index < 0 || index >= list.Last {
		return nil, errors.New("index out of range")
	}

	// 正常 return
	return list.Last[index], nil
}

// 修改数据
func (list *ArrayList) Set(index int, newVal interface{}) error {
	// 下标无效，越界
	if index < 0 || index >= list.Last {
		return nil, errors.New("index out of range")
	}

	//
	list.data[index] = newVal
	return nil
}

// 插入数据
func (list *ArrayList) Insert(index int, newVal interface{}) error {
	// 下标无效，越界
	if index < 0 || index >= list.Last {
		return nil, errors.New("index out of range")
	}

	// 检测切片容量
	list.checkIsFull()

	// 内存移位
	list.data = list.data[:list.Last+1]

	// 数据移位，从后往前的顺序移动
	for i := list.Last, i > index; i-- {
		list.data[i] = list.data[i-1]
	}

	list.data[index] = newVal

	// 数组大小加1
	list.Last++
	
	return nil
}

func (list *ArrayList) Delete(index int) error {
	// 以删除点为界，分为前后俩切片，重新拼接
	list.data = append(list.data[:index], list.data[index+1:]...)
	// 数组大小减一
	list.Last--

	return nil
}

func (list *ArrayList) Clear() {
	// 重新开启新的内存空间
	list.data = make([]interface{}, 0, 10)

	// 数组大小设置为0
	list.Last = 0
}