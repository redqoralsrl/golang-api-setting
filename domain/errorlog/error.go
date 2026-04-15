package errorlog

import (
	"fmt"
)

func NewDatabaseError(operation string, cause error) error {
	return fmt.Errorf("database operation %q failed: %w", operation, cause)
}
