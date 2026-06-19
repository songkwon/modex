package store

import _ "embed"

// Seed markdown keeps the optional local development dataset self-contained.
var (
	//go:embed seeddata/demo-guide.md
	seedDemoGuideMD string
	//go:embed seeddata/demo-maintenance.md
	seedDemoMaintenanceMD string
	//go:embed seeddata/cbb-build-cache.md
	seedCBBBuildCacheMD string
)
