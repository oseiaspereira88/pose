package pose

// RejectDuplicateJSONKeys reports a JSON document that repeats a key at any level, or holds a
// control character where JSON allows none: such a document means different things to different
// readers (most keep the last repetition), so a consumer that acts on it refuses it.
func RejectDuplicateJSONKeys(raw []byte) error { return rejectDuplicateJSONKeysAndControls(raw) }
