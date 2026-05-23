package version

var (
	Version        = "v0.1.0"
	Commit         = "unknown"
	PublisherName  = "rkriad585"
	PublisherEmail = "rkriad585@gmail.com"
)

func init() {
	if Commit == "" {
		Commit = "unknown"
	}
}

func String() string {
	if Commit != "unknown" {
		return Version + " (" + Commit + ")"
	}
	return Version
}
