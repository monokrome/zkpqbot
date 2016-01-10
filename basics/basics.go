package basics

import (
	"fmt"

	"github.com/aarondl/ultimateq/bot"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/irc"
)

func init() {
	bot.RegisterExtension("basics", &Handler{})
}

// Handler extension
type Handler struct {
	b                *bot.Bot
	privmsgHandlerID uint64
	joinHandlerID    uint64
}

// Init the extension
func (h *Handler) Init(b *bot.Bot) error {
	h.privmsgHandlerID = b.Register(irc.PRIVMSG, h)
	h.joinHandlerID = b.Register(irc.JOIN, h)
	b.RegisterCmd(cmd.MkAuthCmd(
		"basics",
		"Ops or voices a user if they have o or v flags respectively.",
		"up",
		h,
		cmd.PRIVMSG, cmd.ALLSCOPES, 0, "", "#chan",
	))

	return nil
}

// Deinit the extension
func (h *Handler) Deinit(b *bot.Bot) error {
	b.UnregisterCmd("basics", "up")
	b.Unregister(h.joinHandlerID)
	b.Unregister(h.privmsgHandlerID)
	return nil
}

// Cmd handler
func (_ *Handler) Cmd(_ string, _ irc.Writer, _ *cmd.Event) error {
	return nil
}

// Up lets a user with proper access voice/op themselves.
func (h *Handler) Up(w irc.Writer, ev *cmd.Event) error {
	user := ev.StoredUser
	ch := ev.TargetChannel
	if ch == nil {
		return fmt.Errorf("Must be a channel that the bot is on.")
	}
	chname := ch.Name()

	if !putPeopleUp(ev.Event, chname, user, w) {
		return cmd.MakeFlagsError("ov")
	}
	return nil
}

// HandleRaw to check for join messages to auto-op auto-voice people on
func (h *Handler) HandleRaw(w irc.Writer, ev *irc.Event) {
	if ev.Name == irc.JOIN {
		store := h.b.Store()
		a := store.AuthedUser(ev.NetworkID, ev.Sender)
		ch := ev.Target()
		putPeopleUp(ev, ch, a, w)
	}
}

func putPeopleUp(ev *irc.Event, ch string,
	a *data.StoredUser, w irc.Writer) (did bool) {
	if a != nil {
		nick := ev.Nick()
		if a.HasFlags(ev.NetworkID, ch, "o") {
			w.Sendf("MODE %s +o :%s", ch, nick)
			did = true
		} else if a.HasFlags(ev.NetworkID, ch, "v") {
			w.Sendf("MODE %s +v :%s", ch, nick)
			did = true
		}
	}
	return
}
