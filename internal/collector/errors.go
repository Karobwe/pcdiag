package collector

import (
	"errors"
	"fmt"
)

// multiError accumulates non-fatal errors from a collector.
type multiError struct{ errs []error }

func (m *multiError) add(scope string, err error) {
	if err != nil {
		m.errs = append(m.errs, fmt.Errorf("%s: %w", scope, err))
	}
}

func (m *multiError) err() error { return errors.Join(m.errs...) }
