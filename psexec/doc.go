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
// # Command Configuration
//
// Commands can be configured using method chaining:
//
//	cmd := psexec.NewCommand("my-program").
//		WithDir("/tmp").
//		WithEnv([]string{"DEBUG=1"}).
//		WithTimeout(30 * time.Second).
//		WithStdin(inputReader)
//
// # PTY Support
//
// Enable PTY for commands that require terminal emulation:
//
//	cmd := psexec.NewCommand("vim").WithPTY()
//
// # Interactive Mode
//
// For fully interactive commands that need bidirectional terminal I/O:
//
//	cmd := psexec.NewCommand("bash").AsInteractive()
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
