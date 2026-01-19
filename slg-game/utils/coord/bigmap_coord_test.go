package coord

import (
	"testing"
)

func TestBigmapCoord(t *testing.T) {
	// 测试编码解码
	testCases := []struct {
		x, y int32
	}{
		{0, 0},
		{1, 1},
		{100, 200},
		{4095, 4095}, // 最大值
	}

	for _, tc := range testCases {
		// 编码
		coord := EncodeCoord(tc.x, tc.y)

		// 解码
		dx, dy := coord.Decode()

		if dx != tc.x || dy != tc.y {
			t.Errorf("EncodeCoord(%d, %d) -> Decode() = (%d, %d), expected (%d, %d)",
				tc.x, tc.y, dx, dy, tc.x, tc.y)
		}

		// 测试单独的X()和Y()方法
		if coord.X() != tc.x {
			t.Errorf("coord.X() = %d, expected %d", coord.X(), tc.x)
		}
		if coord.Y() != tc.y {
			t.Errorf("coord.Y() = %d, expected %d", coord.Y(), tc.y)
		}
	}
}

func TestBigmapCoord_IsValid(t *testing.T) {
	coord := EncodeCoord(50, 100)

	// 测试有效范围
	if !coord.IsValid(100, 200) {
		t.Error("Expected coordinate to be valid")
	}

	// 测试无效范围
	if coord.IsValid(30, 50) {
		t.Error("Expected coordinate to be invalid")
	}
}

func TestBigmapCoord_Distance(t *testing.T) {
	coord1 := EncodeCoord(0, 0)
	coord2 := EncodeCoord(3, 4)

	distance := coord1.Distance(coord2)
	expected := int32(7) // |3-0| + |4-0| = 7

	if distance != expected {
		t.Errorf("Distance between (0,0) and (3,4) = %d, expected %d", distance, expected)
	}
}

func TestBigmapCoord_String(t *testing.T) {
	coord := EncodeCoord(123, 456)
	str := coord.String()
	expected := "BigmapCoord(123,456)"

	if str != expected {
		t.Errorf("String() = %s, expected %s", str, expected)
	}
}
