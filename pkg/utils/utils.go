package utils


func PixelToRGBA(r, g, b, a uint8) uint32 {
	return uint32(r)<<(8*0) | uint32(g)<<(8*1) | uint32(b)<<(8*2) | uint32(a)<<(8*3)
}

func RGBAFromPixel(p uint32) (r, g, b, a uint8) {
	return uint8(p >> (8 * 0)), uint8(p >> (8 * 1)), uint8(p >> (8 * 2)), uint8(p >> (8 * 3))
}
