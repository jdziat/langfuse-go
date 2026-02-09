package langfuse

import "github.com/jdziat/langfuse-go/pkg/config"

// Region represents a Langfuse cloud region.
type Region = config.Region

// Region constants.
const (
	// RegionEU is the European cloud region.
	RegionEU = config.RegionEU
	// RegionUS is the US cloud region.
	RegionUS = config.RegionUS
	// RegionHIPAA is the HIPAA-compliant US region.
	RegionHIPAA = config.RegionHIPAA
)

// regionBaseURLs maps regions to their base URLs.
var regionBaseURLs = config.RegionBaseURLs
