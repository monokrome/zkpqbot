package main

import (
	"math/rand"
	"strings"
	"time"

	"github.com/aarondl/ultimateq/bot"
	"github.com/aarondl/ultimateq/irc"
	"github.com/inconshreveable/log15"

	_ "github.com/aarondl/zkpqbot/queryer"
	_ "github.com/aarondl/zkpqbot/quoter"
	_ "github.com/aarondl/zkpqbot/runner"
)

// Handler extension
type Handler struct {
}

// PrivmsgUser allows the "do" command from a hardcoded bot owner
func (h *Handler) PrivmsgUser(w irc.Writer, ev *irc.Event) {
	flds := strings.Fields(ev.Message())
	if ev.Nick() == "Aaron" && flds[0] == "do" {
		w.Send(strings.Join(flds[1:], " "))
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	h := &Handler{}

	err := bot.Run(func(b *bot.Bot) {
		b.Register(irc.PRIVMSG, h)
	})

	if err != nil {
		log15.Error(err.Error())
	}
}
