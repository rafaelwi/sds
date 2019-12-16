package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/carlescere/scheduler"
)

func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.Parse()
}

// Create the array/slice that will act as a queue and save messages
var msgQueue = make([]string, 0)
var token string
var buffer = make([][]byte, 0)
var nextSDSMsg = time.Now().Add(time.Second * 15)

func main() {
	fmt.Println("next write: ", nextSDSMsg)
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

	// Set up the scheduled job to run every so often
	job := writeMsgsToFile
	scheduler.Every(10).Seconds().Run(job)

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
	s.UpdateListeningStatus("your conversations")
}

// Function called every time a new message is created in a bot-authorized chan
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// If user requests for information about the server
	if strings.HasPrefix(m.Content, "./machine") {
		// Run neofetch command and get output
		out, err := exec.Command("neofetch", "--disable", "underline",
			"--stdout").Output()

		// Error check and make sure everything happened correctly
		if err != nil {
			fmt.Println("[ERR!] Could not run ./machine command")
			return
		}

		// Convert neofetch output to string
		output := string(out)

		// Send the message
		s.ChannelMessageSend(m.ChannelID, "```"+output+"```")
	} else {
		// Otherwise send the message
		s.ChannelMessageSend(m.ChannelID, m.Content)
		msgQueue = append(msgQueue, m.Content)
		fmt.Println("Current queue: ")
		for _, msgs := range msgQueue {
			fmt.Println("[" + msgs + "]")
		}
	}
}

func writeMsgsToFile() {
	// Open file
	f, err := os.OpenFile("msglog.txt", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Println("[ERR!] Could not open file!", err)
		return
	}

	defer f.Close()

	// Write to file
	for _, msg := range msgQueue {
		f.WriteString(msg + "\xffff")
	}
	fmt.Println("Wrote queue to file")

	// Reset timer
	nextSDSMsg = time.Now().Add(time.Second * 10)

	// Clear queue
	msgQueue = make([]string, 0)
}
