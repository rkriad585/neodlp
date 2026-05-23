package version

var (
	Version        = "v0.2.1"
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
