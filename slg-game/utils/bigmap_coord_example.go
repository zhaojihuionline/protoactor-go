package utils

import "fmt"

// ExampleBigmapCoordUsage BigmapCoord使用示例
func ExampleBigmapCoordUsage() {
	fmt.Println("=== BigmapCoord 使用示例 ===")

	// 创建坐标
	coord1 := NewBigmapCoord(100, 200)
	coord2 := EncodeCoord(300, 400)

	fmt.Printf("坐标1: %s\n", coord1.String())
	fmt.Printf("坐标2: %s\n", coord2.String())

	// 解码坐标
	x1, y1 := coord1.Decode()
	x2, y2 := coord2.Decode()
	fmt.Printf("坐标1解码: (%d, %d)\n", x1, y1)
	fmt.Printf("坐标2解码: (%d, %d)\n", x2, y2)

	// 验证有效性
	fmt.Printf("坐标1在1000x1000范围内有效: %v\n", coord1.IsValid(1000, 1000))

	// 计算距离
	distance := coord1.Distance(coord2)
	fmt.Printf("两坐标间距离: %d\n", distance)

	// 坐标运算示例
	fmt.Println("\n=== 坐标运算示例 ===")
	for y := int32(0); y < 3; y++ {
		for x := int32(0); x < 3; x++ {
			coord := EncodeCoord(x*100, y*100)
			fmt.Printf("%s ", coord.String())
		}
		fmt.Println()
	}
}
