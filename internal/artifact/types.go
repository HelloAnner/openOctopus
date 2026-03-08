package artifact

type Store struct {
	sessionDir   string
	artifactsDir string
	lineagePath  string
	indexPath    string
}

type PublishRequest struct {
	ArtifactName  string
	StageID       string
	RoleID        string
	TaskID        string
	SourceRef     string
	ConclusionRef string
	TurnOutputRef string
}

type ArtifactVersion struct {
	ArtifactName  string
	Version       int
	StageID       string
	RoleID        string
	TaskID        string
	SourceRef     string
	ContentRef    string
	ManifestRef   string
	DiffRef       string
	ContentHash   string
	SourceKind    string
	CreatedAt     string
	SizeBytes     int64
	LineCount     int
	FileCount     int
	PreviousVersion int
}

type lineageRecord struct {
	ArtifactName  string
	Version       int
	StageID       string
	RoleID        string
	TaskID        string
	SourceRef     string
	ContentRef    string
	ManifestRef   string
	ConclusionRef string
	TurnOutputRef string
	CreatedAt     string
}
