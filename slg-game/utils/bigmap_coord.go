package utils

import "fmt"

// BigmapCoord 大地图坐标编码类型
// 使用int64位移编码，高位存储Y坐标，低位存储X坐标
type BigmapCoord int64

// 坐标编码常量
const (
	COORD_BITS = 12                    // 坐标位数 (支持4096x4096坐标空间)
	COORD_MASK = (1 << COORD_BITS) - 1 // 坐标掩码
)

// NewBigmapCoord 创建新的大地图坐标
func NewBigmapCoord(x, y int32) BigmapCoord {
	return BigmapCoord(int64(y)<<COORD_BITS | int64(x))
}

// EncodeCoord 将坐标编码为BigmapCoord
func EncodeCoord(x, y int32) BigmapCoord {
	return BigmapCoord(int64(y)<<COORD_BITS | int64(x))
}

// DecodeCoord 从BigmapCoord解码出坐标
func (bc BigmapCoord) Decode() (x, y int32) {
	id := int64(bc)
	return int32(id & COORD_MASK), int32(id >> COORD_BITS)
}

// X 获取X坐标
func (bc BigmapCoord) X() int32 {
	id := int64(bc)
	return int32(id & COORD_MASK)
}

// Y 获取Y坐标
func (bc BigmapCoord) Y() int32 {
	id := int64(bc)
	return int32(id >> COORD_BITS)
}

// String 字符串表示
func (bc BigmapCoord) String() string {
	x, y := bc.Decode()
	return fmt.Sprintf("BigmapCoord(%d,%d)", x, y)
}

// IsValid 验证坐标是否在有效范围内
func (bc BigmapCoord) IsValid(maxX, maxY int32) bool {
	x, y := bc.Decode()
	return x >= 0 && x < maxX && y >= 0 && y < maxY
}

// Distance 计算到另一个坐标的曼哈顿距离
func (bc BigmapCoord) Distance(other BigmapCoord) int32 {
	x1, y1 := bc.Decode()
	x2, y2 := other.Decode()
	return abs(x1-x2) + abs(y1-y2)
}

// abs 绝对值
func abs(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}
