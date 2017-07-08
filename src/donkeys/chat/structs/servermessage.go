package structs

// ServerMessage contains the parsed parts of a server message
type ServerMessage struct {
	Source     string
	Command    string
	Target     string
	Subcommand string
	Text       string
}
