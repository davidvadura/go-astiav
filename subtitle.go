package astiav

//#include <libavcodec/avcodec.h>
import "C"
import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"unsafe"
)

type Subtitle struct {
	c *C.AVSubtitle
}

func AllocSubtitle() *Subtitle {
	c := (*C.AVSubtitle)(C.av_mallocz(C.size_t(unsafe.Sizeof(C.AVSubtitle{}))))
	return &Subtitle{
		c: c,
	}
}

func (s *Subtitle) Free() {
	C.avsubtitle_free(s.c)
	C.av_free(unsafe.Pointer(s.c))
}

func (s *Subtitle) Debug() {
	if s.c == nil {
		fmt.Println("subtitles: nil")
		return
	}
	fmt.Printf("subtitles: num=%d s:e=%d:%d (d=%d)\n",
		s.c.num_rects,
		s.c.start_display_time,
		s.c.end_display_time,
		s.c.end_display_time-s.c.start_display_time,
	)
}

func (s *Subtitle) GetRectCount() int {
	if s.c == nil {
		return 0
	}
	return int(s.c.num_rects)
}

func (s *Subtitle) GetImage() (image.Image, error) {
	x, y, w, h := s.calcBoundingBox()
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	for i := 0; i < s.GetRectCount(); i++ {
		if err := s.copyPixels(i, x, y, img); err != nil {
			return nil, fmt.Errorf("astiav: copy subtitle pixels to image: %w", err)
		}
	}

	return img, nil
}

// calcBoundingBox calculates bounding box for containing all rects in the subtitle
func (s *Subtitle) calcBoundingBox() (int, int, int, int) {
	if s.c == nil {
		return 0, 0, 0, 0
	}
	sub := s.c

	minX, minY, maxX, maxY := 9999, 9999, 0, 0

	for i := 0; i < s.GetRectCount(); i++ {
		rect := *(**C.AVSubtitleRect)(unsafe.Pointer(
			uintptr(unsafe.Pointer(sub.rects)) + uintptr(i)*unsafe.Sizeof(*sub.rects),
		))

		x1 := int(rect.x)
		y1 := int(rect.y)
		x2 := int(rect.x + rect.w)
		y2 := int(rect.y + rect.h)

		if x1 < minX {
			minX = x1
		}
		if y1 < minY {
			minY = y1
		}
		if x2 > maxX {
			maxX = x2
		}
		if y2 > maxY {
			maxY = y2
		}
	}

	return minX, minY, maxX - minX, maxY - minY
}

func (s *Subtitle) copyPixels(index, minX, minY int, img *image.RGBA) error {
	if s.c == nil {
		return nil
	}
	sub := s.c

	if int(sub.num_rects) <= index {
		return errors.New("astiav: index out of range")
	}

	rect := *(**C.AVSubtitleRect)(unsafe.Pointer(
		uintptr(unsafe.Pointer(sub.rects)) + uintptr(index)*unsafe.Sizeof(*sub.rects),
	))

	if rect._type != C.SUBTITLE_BITMAP {
		return errors.New("astiav: not a bitmap subtitle")
	}

	ox := int(rect.x) - minX
	oy := int(rect.y) - minY
	w := int(rect.w)
	h := int(rect.h)

	data := unsafe.Pointer(rect.data[0])
	palette := unsafe.Pointer(rect.data[1])
	hasPalette := palette != nil
	linesize := int(rect.linesize[0])

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			offset := uintptr(y*linesize + x)
			index := *(*uint8)(unsafe.Pointer(uintptr(data) + offset))

			var r, g, b, a uint8

			if hasPalette {
				colorEntry := *(*uint32)(unsafe.Pointer(uintptr(palette) + uintptr(index)*4))
				a = uint8((colorEntry >> 24) & 0xFF)
				r = uint8((colorEntry >> 16) & 0xFF)
				g = uint8((colorEntry >> 8) & 0xFF)
				b = uint8(colorEntry & 0xFF)
			} else {
				r, g, b, a = 255, 255, 255, index
			}

			img.SetRGBA(ox+x, oy+y, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}

	return nil
}
