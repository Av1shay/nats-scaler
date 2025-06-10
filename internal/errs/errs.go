package errs

import (
	"fmt"
)

type HTTPStatusCodeErr struct {
	Code int
	Body []byte
}

func (e *HTTPStatusCodeErr) Error() string {
	bod := "no body"
	if len(e.Body) > 0 {
		bod = string(e.Body)
	}
	return fmt.Sprintf("HTTP Error with status %d: %s", e.Code, bod)
}
