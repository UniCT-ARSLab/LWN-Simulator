package services

// acquireHardwareMutation coordinates CRUD operations with a running
// simulation while allowing services to be used without a runtime in tests
// and during profile bootstrap.
func acquireHardwareMutation(runtime interface{ AcquireHardwareMutation() func() }) func() {
	if runtime == nil {
		return func() {}
	}

	return runtime.AcquireHardwareMutation()
}
