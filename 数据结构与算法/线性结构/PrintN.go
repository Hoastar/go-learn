package main

import (
	"fmt"
	"time"
)

/*
// 循环实现
func PrintN(num int) {
	var i int
	for i = 1; i <= num; i++ {
		fmt.Printf("%d\n", i)
	}
	return
}

// 递归实现
func PrintN(num int) {
	if num != 0 {
		PrintN(num - 1)
		fmt.Printf("%d\n", num)
	}
}

func main() {
	PrintN(10)
}
*/

func timeCost() func() {
	startTime := time.Now()
	return func() {
		tc := time.Since(startTime)
		fmt.Printf("time cost = %v\n", tc)
	}
}

