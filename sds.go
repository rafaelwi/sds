package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.Parse()
}

var token string
var buffer = make([][]byte, 0)

func main() {
	// Check if a token has been provided
	if token == "" {
		fmt.Println("[ERR!] No token has been provided. Rerun with ./sds -t " +
			"<token>")
		return
	}

	// Create a new Discord session using the provided token, check if any
	// errors occur and return if so
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("[ERR!] Could not create Discord session: ", err)
		return
	}

	// Add handlers to do things
	dg.AddHandler(ready)
	dg.AddHandler(messageCreate)

	// Open the websocket and begin listening
	err = dg.Open()
	if err != nil {
		fmt.Println("[ERR!] Could not open Discord session : ", err)
		return
	}

	// Wait here until CTRL-C is recieved
	fmt.Println("[INFO] SDS is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Close the session cleanly
	dg.Close()
	fmt.Println("\n[INFO] Bot has successfully closed. Goodnight sweet prince")
}

// Ran when recieves the "ready" status from Discord
func ready(s *discordgo.Session, event *discordgo.Ready) {

	// Set the playing status.
	s.UpdateStatus(0, "listening to your conversations")
}

// Function called every time a new message is created in a bot-authorized chan
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Otherwise send the message
	s.ChannelMessageSend(m.ChannelID, m.Content)
}
