/*
	SDS Bot (Stuff Discord Says)
	github.com/rafaelwi
	12.14.19 - ?
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/carlescere/scheduler"
)

// Struct for storing messages in the queue
type discordMessage struct {
	msg   string
	guild string
}

// Handles command line args
func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.Parse()
}

// Create the array/slice that will act as a queue and save messages
var msgQueue = make([]discordMessage, 0)
var token string
var buffer = make([][]byte, 0)

func main() {
	// Set up the hashmap
	var guildMap = make(map[string]int)
	var reverseGuildMap = make(map[int]string)
	var totalGuilds = 1

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

	// Open the websocket and begin listening
	err = dg.Open()
	if err != nil {
		fmt.Println("[ERR!] Could not open Discord session : ", err)
		return
	}

	// Add handlers to do things
	dg.AddHandler(ready)
	dg.AddHandler(messageCreate)

	// Set up the scheduled job to run every so often
	scheduler.Every(30).Seconds().Run(func() {
		writeMsgsToFile(guildMap, reverseGuildMap, &totalGuilds)
	})

	// Wait here until CTRL-C is recieved
	fmt.Println("[INFO] SDS is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Close the session cleanly
	dg.Close()
	writeMsgsToFile(guildMap, reverseGuildMap, &totalGuilds)
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
		// Otherwise log the message
		//s.ChannelMessageSend(m.ChannelID, m.Content)
		newMsg := discordMessage{m.Content, m.GuildID}
		msgQueue = append(msgQueue, newMsg)
		fmt.Println("Current queue: ")
		for _, msgs := range msgQueue {
			fmt.Println("[" + msgs.guild + " : " + msgs.msg + "]")
		}
	}
}

// Writes messages to the log
func writeMsgsToFile(guildMap map[string]int, reverseGuildMap map[int]string, largestMapVal *int) {
	// Check if queue is empty, if so then do not write to file
	if len(msgQueue) == 0 {
		fmt.Println("[INFO] Queue is empty, nothing will be written to msglog")
		return
	}
	// TODO: Build a system where we filter messages into different slices
	// depending on what guild they are from. Possibly make a map of all the
	// guilds first, then make slices for them, then store messages in the
	// appropriate slices, then process slice by slice.

	// Look through the queue of messages and check if their guild ID exists in
	// the map
	for _, msg := range msgQueue {
		// Check if key exists. If key does not exist then add it to the map
		_, ok := guildMap[msg.guild]

		if !ok {
			guildMap[msg.guild] = *largestMapVal
			reverseGuildMap[*largestMapVal] = msg.guild
			*largestMapVal++
		}
	}

	// Now make slices for the messages
	sortedMsgs := make([][]discordMessage, *largestMapVal)
	for i := range sortedMsgs {
		sortedMsgs[i] = make([]discordMessage, 0)
	}

	// Now sort the messages to the appropriate slice
	for _, msg := range msgQueue {
		loc, ok := guildMap[msg.guild]

		// Quick error check to see if we have a slice for this message
		if !ok {
			fmt.Println("[ERR!] No slice exists for this msg! Offending msg: [" + msg.guild + "] : " + msg.msg)
			continue
		}

		// Assuming that we do have a slice for this message, put the message
		// in the respective slice
		sortedMsgs[loc] = append(sortedMsgs[loc], msg)
	}

	// Go through each slice and write to the respective files
	for i := 1; i < *largestMapVal; i++ {
		f, err := os.OpenFile(reverseGuildMap[i]+"_msglog.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			fmt.Println("[ERR!] Could not open file!")
			continue
		}

		defer f.Close()

		// Write to file
		for j := 0; j < len(sortedMsgs[i]); j++ {
			f.WriteString(sortedMsgs[i][j].msg + "\xff")
		}
		fmt.Println("[INFO] Wrote queue for guild " + reverseGuildMap[i])
	}

	/*
		// Open file
		f, err := os.OpenFile("msglog.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			fmt.Println("[ERR!] Could not open file!", err)
			return
		}

		defer f.Close()

		// Write to file
		for _, msg := range msgQueue {
			f.WriteString(msg.msg + "\xff")
		}
		fmt.Println("[INFO] Wrote queue to file")
	*/

	// Clear queue
	msgQueue = make([]discordMessage, 0)
}
