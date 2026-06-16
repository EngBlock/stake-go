package secretsauce

import "bytes"

// limitedBuffer accumulates command output up to max bytes. Once the limit is
// reached it stops accepting data, sets overflow, and reports
// errCommandOutputTooLarge to the writer.
type limitedBuffer struct {
	buffer   bytes.Buffer
	max      int
	overflow bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := b.max - b.buffer.Len()
	if remaining <= 0 {
		b.overflow = true
		return 0, errCommandOutputTooLarge
	}
	if len(p) > remaining {
		_, _ = b.buffer.Write(p[:remaining])
		b.overflow = true
		return remaining, errCommandOutputTooLarge
	}
	return b.buffer.Write(p)
}

func (b *limitedBuffer) String() string {
	return b.buffer.String()
}
