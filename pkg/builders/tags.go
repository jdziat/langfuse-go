package builders

// TagsBuilder provides a type-safe way to build tags.
//
// Example:
//
//	tags := NewTags().
//	    Add("production").
//	    Add("api", "v2").
//	    AddIf(isPremium, "premium").
//	    Build()
//
//	trace.Tags(tags).Create(ctx)
type TagsBuilder struct {
	tags []string
}

// NewTags creates a new TagsBuilder.
func NewTags() *TagsBuilder {
	return &TagsBuilder{tags: make([]string, 0)}
}

// Add adds one or more tags.
func (t *TagsBuilder) Add(tags ...string) *TagsBuilder {
	t.tags = append(t.tags, tags...)
	return t
}

// AddIf conditionally adds a tag.
func (t *TagsBuilder) AddIf(condition bool, tag string) *TagsBuilder {
	if condition {
		t.tags = append(t.tags, tag)
	}
	return t
}

// AddIfNotEmpty adds a tag only if it's not empty.
func (t *TagsBuilder) AddIfNotEmpty(tag string) *TagsBuilder {
	if tag != "" {
		t.tags = append(t.tags, tag)
	}
	return t
}

// Environment adds an environment tag (e.g., "env:production").
func (t *TagsBuilder) Environment(env string) *TagsBuilder {
	if env != "" {
		t.tags = append(t.tags, "env:"+env)
	}
	return t
}

// Version adds a version tag (e.g., "version:1.2.3").
func (t *TagsBuilder) Version(version string) *TagsBuilder {
	if version != "" {
		t.tags = append(t.tags, "version:"+version)
	}
	return t
}

// Build returns the constructed tags slice.
func (t *TagsBuilder) Build() []string {
	return t.tags
}
