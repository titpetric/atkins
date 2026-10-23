package version

import (
	"debug/buildinfo"
	"fmt"
	"io"
	"os"
)

// Info contains injected build environment information.
type Info struct {
	Version    string
	Commit     string
	CommitTime string
	Branch     string
}

// Run writes the injected build values and those of the running binary to w.
func Run(w io.Writer, info Info) error {
	fmt.Fprintf(w, "  Version:     %s\n", info.Version)

	// Print VCS information
	if info.Commit != "unknown" {
		shortCommit := info.Commit
		if len(info.Commit) > 12 {
			shortCommit = info.Commit[:12]
		}
		fmt.Fprintf(w, "  Commit:      %s\n", shortCommit)
	}

	if info.CommitTime != "unknown" {
		fmt.Fprintf(w, "  CommitTime:  %s\n", info.CommitTime)
	}

	if info.Branch != "unknown" {
		fmt.Fprintf(w, "  Branch:      %s\n", info.Branch)
	}

	// Get build information from embedded metadata
	exePath, err := os.Executable()
	var bi *buildinfo.BuildInfo
	if err == nil {
		bi, _ = buildinfo.ReadFile(exePath)
	}

	if bi != nil {
		// Module information
		fmt.Fprintf(w, "  Module:      %s\n", bi.Path)
		if bi.Main.Path != "" {
			fmt.Fprintf(w, "  MainModule:  %s\n", bi.Main.Path)
		}
		if bi.Main.Sum != "" {
			fmt.Fprintf(w, "  Sum:         %s\n", bi.Main.Sum)
		}

		// Go version
		if bi.GoVersion != "" {
			fmt.Fprintf(w, "  GoVersion:   %s\n", bi.GoVersion)
		}

		// Extract OS/Arch from settings
		var osVal, archVal string
		for _, setting := range bi.Settings {
			switch setting.Key {
			case "GOOS":
				osVal = setting.Value
			case "GOARCH":
				archVal = setting.Value
			}
		}
		if osVal != "" && archVal != "" {
			fmt.Fprintf(w, "  OS/Arch:     %s/%s\n", osVal, archVal)
		}

		// Print all build settings for reference
		if len(bi.Settings) > 0 {
			fmt.Fprintf(w, "\n  Build Settings:\n")
			for _, setting := range bi.Settings {
				// Skip GOOS/GOARCH as we already printed them
				if setting.Key == "GOOS" || setting.Key == "GOARCH" {
					continue
				}
				fmt.Fprintf(w, "    %s=%s\n", setting.Key, setting.Value)
			}
		}
	}
	return nil
}
