package quoter

import (
	"strconv"
	"time"

	"github.com/aarondl/quotes"
	"github.com/aarondl/ultimateq/bot"
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/irc"
)

const (
	dateFormat = "January 02, 2006 at 3:04pm MST"
)

func init() {
	bot.RegisterExtension("quoter", &Quoter{})
}

// Quoter extension
type Quoter struct {
	db *quotes.QuoteDB
}

// Cmd lets reflection hook up the commands, instead of doing it here.
func (q *Quoter) Cmd(_ string, _ irc.Writer, _ *cmd.Event) error {
	return nil
}

// Init the extension
func (q *Quoter) Init(b *bot.Bot) error {
	qdb, err := quotes.OpenDB("quotes.sqlite3")
	if err != nil {
		return err
	}

	q.db = qdb
	qdb.StartServer(":8000")

	b.RegisterCmd(cmd.MkCmd(
		"quote",
		"Retrieves a quote. Randomly selects a quote if no id is provided.",
		"quote",
		q,
		cmd.PRIVMSG, cmd.ALLSCOPES, "[id]",
	))
	b.RegisterCmd(cmd.MkCmd(
		"quote",
		"Shows the number of quotes in the database.",
		"quotes",
		q,
		cmd.PRIVMSG, cmd.ALLSCOPES,
	))
	b.RegisterCmd(cmd.MkCmd(
		"quote",
		"Gets the details for a specific quote.",
		"details",
		q,
		cmd.PRIVMSG, cmd.ALLSCOPES, "id",
	))
	b.RegisterCmd(cmd.MkCmd(
		"quote",
		"Adds a quote to the database.",
		"addquote",
		q,
		cmd.PRIVMSG, cmd.ALLSCOPES, "quote...",
	))
	b.RegisterCmd(cmd.MkAuthCmd(
		"quote",
		"Removes a quote from the database.",
		"delquote",
		q,
		cmd.PRIVMSG, cmd.ALLSCOPES, 0, "Q", "id",
	))
	b.RegisterCmd(cmd.MkAuthCmd(
		"quote",
		"Edits an existing quote.",
		"editquote",
		q,
		cmd.PRIVMSG, cmd.ALLSCOPES, 0, "Q", "id", "quote...",
	))
	b.RegisterCmd(cmd.MkCmd(
		"quote",
		"Shows the address for the quote webserver.",
		"quoteweb",
		q,
		cmd.PRIVMSG, cmd.ALLSCOPES,
	))

	return nil
}

// Deinit the extension
func (q *Quoter) Deinit(b *bot.Bot) error {
	defer q.db.Close()

	cmdNames := []string{"quote", "quotes", "details", "addquote",
		"delquote", "editquote", "quoteweb"}

	for _, cmd := range cmdNames {
		b.UnregisterCmd("quote", cmd)
	}

	return nil
}

// Addquote to db
func (q *Quoter) Addquote(w irc.Writer, ev *cmd.Event) error {
	nick := ev.Nick()
	quote := ev.Arg("quote")
	if len(quote) == 0 {
		return nil
	}

	id, err := q.db.AddQuote(nick, quote)
	if err != nil {
		w.Noticef(nick, "\x02Quote:\x02 %v", err)
	} else {
		w.Noticef(nick, "\x02Quote:\x02 Added quote #%d", id)
	}
	return nil
}

// Delquote from db
func (q *Quoter) Delquote(w irc.Writer, ev *cmd.Event) error {
	nick := ev.Nick()
	id, err := strconv.Atoi(ev.Arg("id"))

	if err != nil {
		w.Notice(nick, "\x02Quote:\x02 Not a valid id.")
		return nil
	}
	if did, err := q.db.DelQuote(int(id)); err != nil {
		w.Noticef(nick, "\x02Quote:\x02 %v", err)
	} else if !did {
		w.Notice(nick, "\x02Quote:\x02 Could not find quote %d.", id)
	} else {
		w.Noticef(nick, "\x02Quote:\x02 Quote %d deleted.", id)
	}
	return nil
}

// Editquote in db
func (q *Quoter) Editquote(w irc.Writer, ev *cmd.Event) error {
	nick := ev.Nick()
	quote := ev.Arg("quote")
	id, err := strconv.Atoi(ev.Arg("id"))

	if len(quote) == 0 {
		return nil
	}

	if err != nil {
		w.Notice(nick, "\x02Quote:\x02 Not a valid id.")
		return nil
	}
	if did, err := q.db.EditQuote(int(id), quote); err != nil {
		w.Noticef(nick, "\x02Quote:\x02 %v", err)
	} else if !did {
		w.Notice(nick, "\x02Quote:\x02 Could not find quote %d.", id)
	} else {
		w.Noticef(nick, "\x02Quote:\x02 Quote %d updated.", id)
	}
	return nil
}

// Quote returns a random quote
func (q *Quoter) Quote(w irc.Writer, ev *cmd.Event) error {
	strid := ev.Arg("id")
	nick := ev.Nick()

	var quote string
	var id int
	var err error
	if len(strid) > 0 {
		getid, err := strconv.Atoi(strid)
		id = int(getid)
		if err != nil {
			w.Notice(nick, "\x02Quote:\x02 Not a valid id.")
			return nil
		}
		quote, err = q.db.GetQuote(id)
	} else {
		id, quote, err = q.db.RandomQuote()
	}
	if err != nil {
		w.Noticef(nick, "\x02Quote:\x02 %v", err)
		return nil
	}

	if len(quote) == 0 {
		w.Notify(ev.Event, nick, "\x02Quote:\x02 Does not exist.")
	} else {
		w.Notifyf(ev.Event, nick, "\x02Quote (\x02#%d\x02):\x02 %s",
			id, quote)
	}
	return nil
}

// Quotes gets the number of quotes
func (q *Quoter) Quotes(w irc.Writer, ev *cmd.Event) error {
	nick := ev.Nick()

	w.Notifyf(ev.Event, nick, "\x02Quote:\x02 %d quote(s) in database.",
		q.db.NQuotes())
	return nil
}

// Details provides more detail on a given quote
func (q *Quoter) Details(w irc.Writer, ev *cmd.Event) error {
	nick := ev.Nick()
	id, err := strconv.Atoi(ev.Arg("id"))

	if err != nil {
		w.Notice(nick, "\x02Quote:\x02 Not a valid id.")
		return nil
	}

	if date, author, err := q.db.GetDetails(int(id)); err != nil {
		w.Noticef(nick, "\x02Quote:\x02 %v", err)
	} else {
		w.Notifyf(ev.Event, nick,
			"\x02Quote (\x02#%d\x02):\x02 Created on %s by %s",
			id, time.Unix(date, 0).UTC().Format(dateFormat), author)
	}

	return nil
}

// Quoteweb provides a server to see the quotes
func (q *Quoter) Quoteweb(w irc.Writer, ev *cmd.Event) error {
	w.Notify(ev.Event, ev.Nick(), "\x02Quote:\x02 http://bitforge.ca:8000")
	return nil
}
