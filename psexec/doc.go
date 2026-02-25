// Package psexec provides process execution capabilities with support for
// interactive terminals, PTY allocation, and streaming output over various
// transports including websockets.
//
// # Basic Usage
//
// Create an executor and run a simple command:
//
//	exec := psexec.New()
//	cmd := psexec.NewCommand("echo", "hello")
//	result := exec.Run(context.Background(), cmd)
//	fmt.Println(result.Output())
//
// # Shell Commands
//
// Execute commands through the shell:
//
//	cmd := psexec.NewShellCommand("echo $HOME && ls -la")
//	result := exec.Run(ctx, cmd)
//
// Or use the executor's ShellCommand method which respects DefaultShell:
//
//	exec := psexec.NewWithOptions(&psexec.Options{DefaultShell: "bash"})
//	cmd := exec.ShellCommand("echo hello")
//
// # Command Configuration
//
// Commands can be configured by setting struct fields directly:
//
//	cmd := psexec.NewCommand("my-program")
//	cmd.Dir = "/tmp"
//	cmd.Env = []string{"DEBUG=1"}
//	cmd.Timeout = 30 * time.Second
//	cmd.Stdin = inputReader
//
// Or using a struct literal:
//
//	cmd := &psexec.Command{
//		Name:    "my-program",
//		Dir:     "/tmp",
//		Env:     []string{"DEBUG=1"},
//		Timeout: 30 * time.Second,
//	}
//
// # Executor Defaults
//
// Configure default settings for all commands:
//
//	exec := psexec.NewWithOptions(&psexec.Options{
//		DefaultDir:     "/workspace",
//		DefaultEnv:     []string{"CI=true"},
//		DefaultTimeout: 5 * time.Minute,
//		DefaultShell:   "bash",
//	})
//
// # PTY Support
//
// Enable PTY for commands that require terminal emulation:
//
//	cmd := psexec.NewCommand("vim")
//	cmd.UsePTY = true
//
// # Interactive Mode
//
// For fully interactive commands that need bidirectional terminal I/O:
//
//	cmd := psexec.NewCommand("bash")
//	cmd.Interactive = true
//
// # Process Management
//
// For fine-grained control over process I/O, use Start to get a Process handle:
//
//	proc, err := exec.Start(ctx, cmd)
//	if err != nil {
//		return err
//	}
//	defer proc.Close()
//
//	// Read/write directly
//	proc.Write([]byte("input\n"))
//	buf := make([]byte, 1024)
//	n, _ := proc.Read(buf)
//
//	// Or use Pipe for bidirectional copy
//	err = proc.Pipe(stdoutWriter, stdinReader)
//
// # WebSocket Integration
//
// The Process type is designed for websocket transport:
//
//	proc, _ := exec.Start(ctx, cmd)
//	defer proc.Close()
//
//	// PTY() returns the file handle for direct I/O
//	go io.Copy(proc.PTY(), websocketConn)
//	io.Copy(websocketConn, proc.PTY())
//
// # Result Interface
//
// All execution methods return a Result interface:
//
//	result := exec.Run(ctx, cmd)
//	if result.Success() {
//		fmt.Println("Output:", result.Output())
//	} else {
//		fmt.Printf("Failed with exit code %d: %v\n",
//			result.ExitCode(), result.Err())
//		fmt.Println("Stderr:", result.ErrorOutput())
//	}
package psexec
