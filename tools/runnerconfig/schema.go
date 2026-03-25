package runnerconfig

import "os"

const OverrideEnvVar = "GWC_RUNNER_CONFIG"

type Paths struct {
	GeneratedProjectRoot   string `json:"generatedProjectRoot,omitempty"`
	ArtifactRoot           string `json:"artifactRoot,omitempty"`
	WorkspaceBuildRoot     string `json:"workspaceBuildRoot,omitempty"`
	WASMExecJS             string `json:"wasmExecJS,omitempty"`
	GoWASMExec             string `json:"goWasmExec,omitempty"`
	BrowserWorkspace       string `json:"browserWorkspace,omitempty"`
	LivereloadWorkspace    string `json:"livereloadWorkspace,omitempty"`
	LivereloadClientScript string `json:"livereloadClientScript,omitempty"`
}

type Overrides struct {
	Paths Paths `json:"paths,omitempty"`
}

type FS struct {
	Getwd       func() (string, error)
	UserHomeDir func() (string, error)
	ReadFile    func(string) ([]byte, error)
	Stat        func(string) (os.FileInfo, error)
}

func (fs FS) withDefaults() FS {
	if fs.Getwd == nil {
		fs.Getwd = os.Getwd
	}
	if fs.UserHomeDir == nil {
		fs.UserHomeDir = os.UserHomeDir
	}
	if fs.ReadFile == nil {
		fs.ReadFile = os.ReadFile
	}
	if fs.Stat == nil {
		fs.Stat = os.Stat
	}
	return fs
}

func (fs FS) pathExists(path string) bool {
	fs = fs.withDefaults()
	if path == "" {
		return false
	}
	_, err := fs.Stat(path)
	return err == nil
}
