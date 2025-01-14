package ids

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TraceFromString(id string) (pcommon.TraceID, error) {
	var (
		hash       = sha256.Sum256([]byte(fmt.Sprintf("%st", id)))
		traceIDHex = hex.EncodeToString(hash[:])
		traceID    pcommon.TraceID
	)

	_, err := hex.Decode(traceID[:], []byte(traceIDHex[:32]))
	if err != nil {
		return pcommon.TraceID{}, err
	}

	return traceID, nil
}

func SpanFromRandom() (pcommon.SpanID, error) {
	spanID := pcommon.SpanID{}
	n, err := rand.Reader.Read(spanID[:])
	if err != nil {
		return pcommon.NewSpanIDEmpty(), err
	}

	if n < len(spanID) {
		// we've run out of entropy
		panic("short read generating span ID")
	}

	return spanID, nil
}
