/*
	SDS Bot (Stuff Discord Says)
	github.com/rafaelwi
	12.14.19 - ?
*/

package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/carlescere/scheduler"
)

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
	var guildMap = make(map[string]guildData)
	var reverseGuildMap = make(map[int]guildData)
	var totalGuilds = 1
	var isFirstSDSTime = true
	//var guildData = make([]serverData, countNumGuilds())

	// Create the initial hashmaps and slice of data for the different servers
	//var guildDataArr = buildGuildDataArr()

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

	// Schedule a job to send the SDS message
	scheduler.Every(30).Seconds().Run(func() {
		sendSDSMsg(&isFirstSDSTime, guildMap, reverseGuildMap, totalGuilds, dg)
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
	updateListening(s)
}

func updateListening(s *discordgo.Session) {
	s.UpdateListeningStatus("your conversations")
}

func sendSDSMsg(isFirstTime *bool, guildMap map[string]guildData, reverseGuildMap map[int]guildData, totalGuilds int, s *discordgo.Session) {
	if *isFirstTime {
		fmt.Println("This is the first time that the sendSDSMsg function has been envoked. Will not send anything.")
		*isFirstTime = false
		return
	}

	fmt.Println("This is not the first time that the sendSDSMsg function has been envoked. Will now send something.")
	for i := 1; i <= totalGuilds; i++ {
		// Get the guild's data from the reverse map
		currentGuild := reverseGuildMap[i]

		/* Do a check to make sure that there are messages in this guild, if
		there are no messages, then continue to the next guild. */
		if currentGuild.logMsgCount == 0 {
			continue
		}

		// Determine which message to display by generating a random number
		msgNum := rand.Intn(currentGuild.logMsgCount)

		filename := currentGuild.guildID + "_msglog.txt"

		// Open the log file and find that message
		file, err := os.Open(filename)
		if err != nil {
			fmt.Println("[ERR!] Could not open file " + filename + " for reading")
			os.Exit(1)
		}

		defer file.Close()

		// Varaibles needed for reading the message
		buffer := make([]byte, 1)
		msg := []byte{}
		msgCount := 0

		for {
			// Check for errors
			_, err := file.Read(buffer)
			if err != nil && err != io.EOF {
				fmt.Println("[ERR!]", err)
			}

			if err == io.EOF {
				break
			}

			// If the current character is the delimiter, then add 1 to the
			// msgCount. Exit when msgCount is equal to msgNum
			if buffer[0] == byte('ÿ') {
				msgCount++
			}

			// Now we start reading in the message one character at a time
			if msgCount == msgNum {
				msg = append(msg, buffer[0])
			}

			/* If the msgCount is greater than the number of the message to
			be read in, then exit the loop */
			if msgCount > msgNum {
				break
			}
		}

		/* Check if message is longer than 1 character. If it is not, then
		skip printing a message this round */
		if len(msg) <= 1 {
			fmt.Println("[DBUG] Message is 1 char long, cannot send, skipping this round")
			continue
		}

		/* Get a list of the channels in that guild and find one named
		"general". Print the message there. */
		listOfChannels, _ := s.GuildChannels(currentGuild.guildID)

		/* Loop through the list until one is found with the name "general"
		and send a message there. */
		for _, c := range listOfChannels {
			// Make sure that the channel type is a text channel
			if c.Type != discordgo.ChannelTypeGuildText {
				continue
			}

			if c.Name == "general" {
				s.ChannelMessageSend(c.ID, string(msg)[1:])
			}
		}
	}
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
		// Check if the message is blank, if it is then do not save it
		if m.Content == "" {
			fmt.Println("[INFO] Blank message sent, nothing will be logged")
			return
		}

		// Otherwise log the message
		newMsg := discordMessage{m.Content, m.GuildID}
		msgQueue = append(msgQueue, newMsg)
		fmt.Println("Current queue: ")
		for _, msgs := range msgQueue {
			fmt.Println("[" + msgs.guild + " : " + msgs.msg + "]")
		}
	}
}

// Writes messages to the log
func writeMsgsToFile(guildMap map[string]guildData, reverseGuildMap map[int]guildData, largestMapVal *int) {
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
			// Create new data for the guild
			var newData = guildData{*largestMapVal, msg.guild, 0}

			// Count how many msgs have already been received from the guild
			newData.logMsgCount = countMsgsInLog(newData.guildID + "_msglog.txt")

			// Map the data as needed
			guildMap[msg.guild] = newData
			reverseGuildMap[*largestMapVal] = newData
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
		locObj, ok := guildMap[msg.guild]

		// Quick error check to see if we have a slice for this message
		if !ok {
			fmt.Println("[ERR!] No slice exists for this msg! Offending msg: [" + msg.guild + "] : " + msg.msg)
			continue
		}

		loc := locObj.sliceID

		// Assuming that we do have a slice for this message, put the message
		// in the respective slice
		sortedMsgs[loc] = append(sortedMsgs[loc], msg)
	}

	// Go through each slice and write to the respective files
	for i := 1; i < *largestMapVal; i++ {
		f, err := os.OpenFile(reverseGuildMap[i].guildID+"_msglog.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			fmt.Println("[ERR!] Could not open file!")
			continue
		}

		defer f.Close()

		// Write to file
		for j := 0; j < len(sortedMsgs[i]); j++ {
			f.WriteString(sortedMsgs[i][j].msg + "\xff")
		}
		fmt.Println("[INFO] Wrote queue for guild " + reverseGuildMap[i].guildID)

		// Update that guild's message count
		guildDataCpy := reverseGuildMap[i]
		guildDataCpy.logMsgCount += len(sortedMsgs[i])
		guildMap[guildDataCpy.guildID] = guildDataCpy
		reverseGuildMap[i] = guildDataCpy

		fmt.Println("[DBUG] "+reverseGuildMap[i].guildID+" msg count: ", reverseGuildMap[i].logMsgCount)
	}

	// Clear queue
	msgQueue = make([]discordMessage, 0)
}

func countNumGuilds() int {
	count := 0

	// Get the list of files in the current directory
	files, err := ioutil.ReadDir(".")

	if err != nil {
		fmt.Println("[ERR!] Current directory could not be read. Exiting program...")
		os.Exit(1)
	}

	// Loop through the files slice and get the name of the files. Check if
	// the suffix of the files is "_msglog.txt". If so, then add 1 to the count
	for _, file := range files {
		if strings.HasSuffix(file.Name(), "_msglog.txt") {
			count++
		}
	}

	return count
}

func countMsgsInLog(filename string) int {
	var msgCount = 0

	// Open file, read character by character, and count each time there is a
	// character '/xff', which is our delimiter
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("[ERR!] Could not open file " + filename + " for reading")
		os.Exit(1)
	}

	defer file.Close()

	buffer := make([]byte, 1)

	for {
		_, err := file.Read(buffer)
		fmt.Println("read ", rune(buffer[0]))
		if err != nil && err != io.EOF {
			fmt.Println("[ERR!]", err)
		}

		if err == io.EOF {
			break
		}

		if buffer[0] == byte('ÿ') || rune(buffer[0]) == '�' || buffer[0] == '\xff' {
			msgCount++
		}
	}

	return msgCount
}
