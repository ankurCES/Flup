package bench

import (
	"bytes"
	"fmt"
)

// validateResponse checks the response against the config's ExpectStatus
// and ExpectBody. Returns an error string if validation fails, or "" if ok.
// Designed to be called in the hot path — no allocations on success.
func validateResponse(cfg Config, statusCode int, body []byte) string {
	if cfg.ExpectStatus != 0 && statusCode != cfg.ExpectStatus {
		return fmt.Sprintf("status %d, expected %d", statusCode, cfg.ExpectStatus)
	}
	if cfg.ExpectBody != "" && !bytes.Contains(body, []byte(cfg.ExpectBody)) {
		return "body does not contain expected string"
	}
	return ""
}
