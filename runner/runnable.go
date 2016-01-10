package runnable

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aarondl/ultimateq/bot"
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/irc"
)

func init() {
	bot.RegisterExtension("runnable", Runnable{})
}

// Runnable extension
type Runnable struct {
}

// Init the extension
func (r Runnable) Init(b *bot.Bot) error {
	b.RegisterCmd(cmd.MkCmd(
		"runnable",
		"Runs a snippet of sandboxed go code.",
		"go",
		r,
		cmd.PRIVMSG, cmd.ALLSCOPES, "code...",
	))
	b.RegisterCmd(cmd.MkCmd(
		"runnable",
		"Runs a snippet of sandboxed go code inside fmt.Println().",
		"gop",
		r,
		cmd.PRIVMSG, cmd.ALLSCOPES, "code...",
	))

	return nil
}

// Deinit the extension
func (_ Runnable) Deinit(b *bot.Bot) error {
	cmdNames := []string{"google", "calc", "weather"}
	for _, cmd := range cmdNames {
		b.UnregisterCmd("runnable", cmd)
	}
	return nil
}

// Cmd is empty to let reflection deal with command lookup
func (_ Runnable) Cmd(_ string, _ irc.Writer, _ *cmd.Event) error {
	return nil
}

// Go runs code in main.
func (_ Runnable) Go(w irc.Writer, ev *cmd.Event) error {
	return sandboxGo(w, ev, "package main\n\nfunc main() {\n%s\n}")
}

// Gop runs code in main inside a fmt.Println()
func (_ Runnable) Gop(w irc.Writer, ev *cmd.Event) error {
	return sandboxGo(w, ev, "package main\n\nfunc main() {\nfmt.Println(%s)\n}")
}

func sandboxGo(w irc.Writer, ev *cmd.Event, basecode string) error {
	var err error
	var f *os.File

	code := ev.Arg("code")
	nick := ev.Nick()
	targ := ev.Target()

	tmp := os.TempDir()
	frand := rand.Uint32()
	srcfile := filepath.Join(tmp, fmt.Sprintf("%d.go", frand))
	exefile := filepath.Join(tmp, fmt.Sprintf("%d", frand))
	defer os.Remove(srcfile)
	defer os.Remove(exefile)

	f, err = os.Create(srcfile)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(f, basecode, code)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}

	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}

	putStdErr := func(msg string, buf *bytes.Buffer, e error) {
		errMsg := strings.Replace(e.Error(), "\n", "; ", -1)
		outmsg := bytes.Replace(buf.Bytes(), []byte{'\n'}, []byte{';', ' '}, -1)
		w.Notifyf(ev.Event, nick, "\x02go:\x02 %s: %v; %s", msg, errMsg, outmsg)
	}

	goimps := exec.Command("goimports", "-w", srcfile)
	goimps.Stderr = stderr
	if err = goimps.Run(); err != nil {
		putStdErr("Failed to format source", stderr, err)
		return nil
	}
	stderr.Reset()

	build := exec.Command("go", "build", "-o", exefile, srcfile)
	build.Env = os.Environ()
	build.Env = append(build.Env, "GOOS=nacl")
	build.Env = append(build.Env, "GOARCH=amd64p32")
	build.Stderr = stderr
	if err = build.Run(); err != nil {
		putStdErr("Failed to compile", stderr, err)
		return nil
	}
	stderr.Reset()

	run := exec.Command("sel_ldr_x86_64", exefile)
	run.Stderr = stderr
	run.Stdout = stdout
	if err = run.Start(); err != nil {
		putStdErr("Failed to run", stderr, err)
		return nil
	}

	doneChan := make(chan error)
	go func() {
		err := run.Wait()
		doneChan <- err
	}()

	select {
	case err = <-doneChan:
		if err != nil {
			putStdErr("Failed to run", stderr, err)
			return nil
		}
	case <-time.After(time.Second * 4):
		run.Process.Kill()
		w.Notifyf(ev.Event, nick,
			"\x02go:\x02 Program took too long, terminated.")
		return nil
	}

	outbytes := bytes.Replace(stdout.Bytes(), []byte{1}, []byte{}, -1)
	out := fmt.Sprintf("\x02go:\x02 %s", outbytes)
	// ircmaxlen - maxhostsize - PRIVMSG - targetsize - spacing - colons
	maxlen := 2 * (510 - 62 - 7 - len(targ) - 3 - 2)
	if len(out) > maxlen {
		out = out[:maxlen-3]
		out += "..."
	}
	w.Notifyf(ev.Event, nick, out)
	return nil
}
