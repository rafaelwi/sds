/*
	discordgoAPI.go
	Functions that correspond to the discordgo wrapper's API functions.
	github.com/rafaelwi
*/

package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Ran when recieves the "ready" status from Discord
func ready(s *discordgo.Session, event *discordgo.Ready) {
	// Set the playing status.
	updateListening(s)
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

		// Convert neofetch output to string and send the message
		output := string(out)
		s.ChannelMessageSend(m.ChannelID, "```"+output+"```")
	} else {
		// Check if the message is blank, if it is then do not save it
		if m.Content == "" && len(m.Attachments) == 0 {
			fmt.Println("[INFO] Blank message sent, nothing will be logged")
			return
		}

		// Otherwise log the message
		newMsg := discordMessage{m.Content, m.GuildID}

		/* Check if there are attachments, if there are then append their URLs
		   to the end of the message */
		if len(m.Attachments) > 0 {
			for _, file := range m.Attachments {
				newMsg.msg += " " + file.URL
			}
		}

		// Place message in the queue
		msgQueue = append(msgQueue, newMsg)

		/* Debug code for queues
		fmt.Println("Current queue: ")
		for _, msgs := range msgQueue {
			fmt.Println("[" + msgs.guild + " : " + msgs.msg + "]")
		}
		*/
	}
}
