// Package invariant_flatjson supplies non-recording assertions to shared/encoding/flatjson. It
// stays separate because shared/invariant/default uses that encoder for coverage reports.
// Importing the default package from the encoder would create an import cycle.
package invariant_flatjson

import core "local/james-orcales/shared/invariant"

// Always lets the flat JSON encoder retain assertion enforcement without a dependency cycle.
func Always[T ~bool](condition T, message string) {
	core.Recorder_Always(&core.Recorder{}, condition, message)
}
