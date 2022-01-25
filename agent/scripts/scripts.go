package scripts

import (
	"embed"
	"io/ioutil"
	"os"
	"path/filepath"
)

// scripts embeds the necessary scripts for the agent at runtime
//go:embed debug.cmd
//go:embed perf-archive.sh
var scriptFiles embed.FS

// WriteScripts reads the content of the embedded scripts in the scriptFiles
// variable and writes them next to the agent binary
func WriteScripts() error {
	exeDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return err
	}

	files, err := scriptFiles.ReadDir(".")
	if err != nil {
		return err
	}

	for _, f := range files {
		b, err := scriptFiles.ReadFile(f.Name())
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(exeDir, f.Name()), b, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}
