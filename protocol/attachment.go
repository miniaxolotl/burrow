package protocol

const AttachmentHeader = "X-Burrow-Attachment"
const AttachmentIDHeader = "X-Burrow-Attachment-ID"
const AttemptHeader = "X-Burrow-Attempt"
const ExpectedGenerationHeader = "X-Burrow-Expected-Generation"
const CapabilityPath = "/.well-known/burrow-capabilities"

type Capabilities struct {
	Version              int `json:"version"`
	Registration         int `json:"registration"`
	PersistentAttachment int `json:"persistent_attachment"`
}

type Reservation struct {
	TunnelID   string `json:"tunnel_id"`
	State      string `json:"state"`
	Persistent bool   `json:"persistent"`
}
