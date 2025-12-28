package langfuse

// Region represents a Langfuse cloud region.
type Region string

const (
	// RegionEU is the European cloud region.
	RegionEU Region = "eu"
	// RegionUS is the US cloud region.
	RegionUS Region = "us"
	// RegionHIPAA is the HIPAA-compliant US region.
	RegionHIPAA Region = "hipaa"
)

// regionBaseURLs maps regions to their base URLs.
var regionBaseURLs = map[Region]string{
	RegionEU:    "https://cloud.langfuse.com/api/public",
	RegionUS:    "https://us.cloud.langfuse.com/api/public",
	RegionHIPAA: "https://hipaa.cloud.langfuse.com/api/public",
}

// BaseURL returns the API base URL for this region.
func (r Region) BaseURL() string {
	if url, ok := regionBaseURLs[r]; ok {
		return url
	}
	return regionBaseURLs[RegionEU]
}

// String returns the string representation of the region.
func (r Region) String() string {
	return string(r)
}
