package slsa

type DigestSet map[string]string

type Subject struct {
	Name      string    `json:"name,omitempty"`
	DigestSet DigestSet `json:"digest,omitempty"`
}

type Builder struct {
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
}

type ResourceDescriptor struct {
	// A URI used to identify the resource or artifact globally. This field is
	// REQUIRED unless either digest or content is set.
	URI string `json:"uri,omitempty"`

	// A set of cryptographic digests of the contents of the resource or
	// artifact. This field is REQUIRED unless either uri or content is set.
	DigestSet DigestSet `json:"digest,omitempty"`

	// Machine-readable identifier for distinguishing between descriptors.
	Name string `json:"name,omitempty"`

	// The location of the described resource or artifact, if different from the
	// uri.
	DownloadLocation string `json:"downloadLocation,omitempty"`

	// The MIME Type (i.e., media type) of the described resource or artifact.
	MediaType string `json:"mediaType,omitempty"`

	// The http content type (i.e., media type) of the described resource or artifact.
	ContentType string `json:"contentType,omitempty"`

	// The contents of the resource or artifact. This field is REQUIRED unless
	// either uri or digest is set.
	Content []byte `json:"content,omitempty"`

	// This field MAY be used to provide additional information or metadata
	// about the resource or artifact that may be useful to the consumer when
	// evaluating the attestation against a policy.
	Annotations map[string]interface{} `json:"annotations,omitempty"`
}
