package config

import (
	"os/exec"
)

// execLookPath wraps exec.LookPath for use in config resolution.
var execLookPath = exec.LookPath