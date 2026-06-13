package store

import _ "embed"

// Seed markdown showcases the Mintlify-style component set rendered by the
// frontend MDX engine. They are embedded so the demo data is rich out of the
// box, even before any real documentation artifact is published.
var (
	//go:embed seeddata/demo-guide.md
	seedDemoGuideMD string
	//go:embed seeddata/demo-maintenance.md
	seedDemoMaintenanceMD string
	//go:embed seeddata/cbb-build-cache.md
	seedCBBBuildCacheMD string
)
