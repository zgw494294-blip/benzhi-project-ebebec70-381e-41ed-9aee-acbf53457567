package domain

func (r Release) IsImmutable() bool {
	return r.ReleaseID != "" && r.ContentDigest != "" && r.PublishedText != ""
}
