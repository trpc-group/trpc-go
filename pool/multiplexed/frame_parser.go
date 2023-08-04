package multiplexed

import "io"

// FrameParser is the interface to parse a single frame.
type FrameParser interface {
	// Parse parses vid and frame from io.ReadCloser. rc.Close must be called before Parse return.
	Parse(rc io.Reader) (vid uint32, buf []byte, err error)
}
