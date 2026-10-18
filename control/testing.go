package control

// SetTotalVramForTesting makes TotalVramMB report mb (0 = no GPU) until the
// returned restore func is called. For tests of code that depends on the
// pre-flight guard; production code never calls it.
func SetTotalVramForTesting(mb int) (restore func()) {
	prev := vramProbe
	vramProbe = func() int { return mb }
	return func() { vramProbe = prev }
}
