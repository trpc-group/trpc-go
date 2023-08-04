package stream

import "math"

// define common error strings.
const (
	// can't find this address.
	noSuchAddr string = "no such addr"
	// Couldn't find the stream ID.
	noSuchStreamID string = "no such stream ID"
	// send Close frame error.
	closeSendFail string = "stream: CloseSend fail"
	// stream has been closed.
	streamClosed string = "stream is already closed"
	// unknown frame type.
	unknownFrameType string = "unknown frame type"
	// ServerStreamTransport not implemented.
	streamTransportUnimplemented string = "server StreamTransport is not implemented"
	// msg does not contain FrameHead.
	frameHeadNotInMsg string = "frameHead is not contained in msg"
	// frameHead is invalid, not trpc FrameHead.
	frameHeadInvalid string = "frameHead is invalid"
	// streamFrameInvalid streaming frame type assertion error.
	streamFrameInvalid string = "stream frame assert failed"
	// responseInvalid response type assertion error.
	responseInvalid string = "value type is not response"
)

const (
	// maxInitWindowSize maximum initial window size.
	maxInitWindowSize uint32 = math.MaxUint32
	// defaultInitwindowSize default initialization window size.
	defaultInitWindowSize uint32 = 65535
)

// response contains the received message, including data []byte part and error.
type response struct {
	data []byte
	err  error
}

// getWindowSize gets the window size through the configured parameters.
func getWindowSize(s uint32) uint32 {
	if s <= defaultInitWindowSize {
		return defaultInitWindowSize
	}
	if s >= maxInitWindowSize {
		return maxInitWindowSize
	}
	return s
}
