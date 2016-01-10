package queryer

import (
	"errors"
	"regexp"
	"strings"

	"github.com/aarondl/query"
	"github.com/aarondl/ultimateq/bot"
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/irc"
)

var (
	sanitizeNewline = strings.NewReplacer("\r\n", " ", "\n", " ")
	rgxSpace        = regexp.MustCompile(`\s{2,}`)
	queryConf       query.Config
)

func init() {
	bot.RegisterExtension("queryer", &Queryer{})
}

// Queryer allows for various HTTP queries to different servers.
type Queryer struct {
	privmsgHandlerID uint64
}

// Init the extension
func (q *Queryer) Init(b *bot.Bot) error {
	if conf := query.NewConfig("wolfid.toml"); conf != nil {
		queryConf = *conf
	} else {
		return errors.New("Error loading queryer configuration.")
	}

	q.privmsgHandlerID = b.Register(irc.PRIVMSG, q)
	b.RegisterCmd(cmd.MkCmd(
		"query",
		"Submits a query to Google.",
		"google",
		*q,
		cmd.PRIVMSG, cmd.ALLSCOPES, "query...",
	))
	b.RegisterCmd(cmd.MkCmd(
		"query",
		"Submits a query to Wolfram Alpha.",
		"calc",
		*q,
		cmd.PRIVMSG, cmd.ALLSCOPES, "query...",
	))
	b.RegisterCmd(cmd.MkCmd(
		"query",
		"Fetches a weather report from yr.no.",
		"weather",
		*q,
		cmd.PRIVMSG, cmd.ALLSCOPES, "query...",
	))

	return nil
}

// Deinit the extension
func (q *Queryer) Deinit(b *bot.Bot) error {
	cmdNames := []string{"google", "calc", "weather"}
	for _, cmd := range cmdNames {
		b.UnregisterCmd("query", cmd)
	}
	b.Unregister(q.privmsgHandlerID)
	return nil
}

// Cmd handler to satisfy the interface, but let reflection look up
// all our methods.
func (q Queryer) Cmd(_ string, _ irc.Writer, _ *cmd.Event) error {
	return nil
}

// PrivmsgChannel traps youtube links
func (q Queryer) PrivmsgChannel(w irc.Writer, ev *irc.Event) {
	if out, err := query.YouTube(ev.Message()); len(out) != 0 {
		w.Privmsg(ev.Target(), out)
	} else if err != nil {
		nick := ev.Nick()
		w.Notice(nick, err.Error())
	}
}

// Calc something using wolfram alpha
func (_ Queryer) Calc(w irc.Writer, ev *cmd.Event) error {
	q := ev.Arg("query")
	nick := ev.Nick()

	if out, err := query.Wolfram(q, &queryConf); len(out) != 0 {
		out = sanitize(out)

		// Ensure two lines only
		// ircmaxlen - maxhostsize - PRIVMSG - targetsize - spacing - colons
		maxlen := 2 * (510 - 62 - 7 - len(ev.Target()) - 3 - 2)
		if len(out) > maxlen {
			out = out[:maxlen-3]
			out += "..."
		}

		w.Notify(ev.Event, nick, out)
	} else if err != nil {
		w.Notice(nick, err.Error())
	}

	return nil
}

// Google some query and return the first result
func (_ Queryer) Google(w irc.Writer, ev *cmd.Event) error {
	q := ev.Arg("query")
	nick := ev.Nick()

	if out, err := query.Google(q); len(out) != 0 {
		out = sanitize(out)
		w.Notify(ev.Event, nick, out)
	} else if err != nil {
		w.Notice(nick, err.Error())
	}

	return nil
}

// Weather for the given place
func (_ Queryer) Weather(w irc.Writer, ev *cmd.Event) error {
	q := ev.Arg("query")
	nick := ev.Nick()

	if out, err := query.Weather(q, &queryConf); len(out) != 0 {
		out = sanitize(out)
		w.Notify(ev.Event, nick, out)
	} else if err != nil {
		w.Notice(nick, err.Error())
	}

	return nil
}

func sanitize(str string) string {
	return rgxSpace.ReplaceAllString(sanitizeNewline.Replace(str), " ")
}
