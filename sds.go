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
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

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
var buffer = make([][]byte, 0)
var errLog = make([]string, 0)
var timeToNextMsg = rand.Int()
var token string

func main() {
	var guildMap = make(map[string]guildData)
	var reverseGuildMap = make(map[int]guildData)
	var totalGuilds = 1
	var isFirstSDSTime = true

	// Check if a token has been provided
	if token == "" {
		errorLogger("No token provided. Rerun with ./sds -t <token>", 3)
		return
	}

	// Create a new Discord session using the provided token, check if any
	// errors occur and return if so
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		errorLogger("Could not create the Discord session: "+err.Error(), 3)
		return
	}

	// Open the websocket and begin listening
	err = dg.Open()
	if err != nil {
		errorLogger("Could not open the Discord session: "+err.Error(), 3)
		return
	}

	// Add handlers to do things
	dg.AddHandler(ready)
	dg.AddHandler(messageCreate)

	// Set up the scheduled job to run every so often
	scheduler.Every(90).Seconds().Run(func() {
		writeMsgsToFile(guildMap, reverseGuildMap, &totalGuilds)
	})

	// Schedule a job to send the SDS message
	scheduler.Every(90).Minutes().Run(func() {
		sendSDSMsg(&isFirstSDSTime, guildMap, reverseGuildMap, totalGuilds, dg)
	})

	/* Schedule a job to update the listening rich presense every 3 mins, kinf
	of acts as a heartbeat */
	scheduler.Every(3).Minutes().Run(func() {
		updateListening(dg)
	})

	// Schedule logging
	scheduler.Every(5).Minutes().Run(func() { logMessages() })

	// Wait here until CTRL-C is recieved
	fmt.Println("[INFO] SDS is now running. Press CTRL-C to exit.")
	errorLogger("Bot has started :D", 1)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Close the session cleanly
	dg.Close()
	writeMsgsToFile(guildMap, reverseGuildMap, &totalGuilds)
	logMessages()
	fmt.Println("\n** Bot has successfully closed. Goodnight sweet prince **")
}

func updateListening(s *discordgo.Session) {
	s.UpdateListeningStatus("your conversations")
}

func sendSDSMsg(isFirstTime *bool, guildMap map[string]guildData,
	reverseGuildMap map[int]guildData, totalGuilds int, s *discordgo.Session) {
	if *isFirstTime {
		errorLogger("First time SEND has been envoked, not sending", 1)
		*isFirstTime = false
		return
	}

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

		// Open the log file and find that message
		filename := currentGuild.guildID + "_msglog.txt"
		file, err := os.Open(filename)
		if err != nil {
			errorLogger("Could not open file "+filename+" for reading", 3)
			continue
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
				errorLogger(err.Error(), 3)
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
			errorLogger("Unsendable msg for "+currentGuild.guildID+
				", skipping", 2)
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
				msgToSend := string(msg)
				reIStatements := regexp.MustCompile(`(?i)sds`)
				reIsStatements := regexp.MustCompile(`(?i)sds is`)
				msgToSend = reIsStatements.ReplaceAllString(msgToSend, "I am")
				msgToSend = reIStatements.ReplaceAllString(msgToSend, "I")
				s.ChannelMessageSend(c.ID, msgToSend[1:])
			}
		}
	}
}

// Writes messages to the log
func writeMsgsToFile(guildMap map[string]guildData,
	reverseGuildMap map[int]guildData, largestMapVal *int) {
	// Check if queue is empty, if so then do not write to file
	if len(msgQueue) == 0 {
		errorLogger("Queue is empty, nothing written ", 1)
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
			newData.logMsgCount = countMsgsInLog(newData.guildID +
				"_msglog.txt")

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
			errorLogger("No slice exists for this msg! Offending: [ "+
				msg.guild+"] : "+msg.msg, 3)
			continue
		}

		loc := locObj.sliceID

		// Assuming that we do have a slice for this message, put the message
		// in the respective slice
		sortedMsgs[loc] = append(sortedMsgs[loc], msg)
	}

	// Go through each slice and write to the respective files
	for i := 1; i < *largestMapVal; i++ {
		file, err := os.OpenFile(reverseGuildMap[i].guildID+"_msglog.txt",
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			errorLogger("Could not open file "+reverseGuildMap[i].guildID, 3)
			continue
		}

		defer file.Close()

		// Write to file
		for j := 0; j < len(sortedMsgs[i]); j++ {
			file.WriteString(sortedMsgs[i][j].msg + "\xff")
		}

		errorLogger("Wrote queue for guild "+reverseGuildMap[i].guildID, 1)

		// Update that guild's message count
		guildDataCpy := reverseGuildMap[i]
		guildDataCpy.logMsgCount += len(sortedMsgs[i])
		guildMap[guildDataCpy.guildID] = guildDataCpy
		reverseGuildMap[i] = guildDataCpy
	}

	// Clear queue
	msgQueue = make([]discordMessage, 0)
}

func countNumGuilds() int {
	count := 0

	// Get the list of files in the current directory
	files, err := ioutil.ReadDir(".")
	if err != nil {
		errorLogger("Current directory could not be read. Exiting program", 3)
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
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		errorLogger("Could not open file "+filename+" for reading", 3)
		os.Exit(1)
	}

	defer file.Close()

	buffer := make([]byte, 1)

	for {
		_, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			errorLogger(err.Error(), 3)
		}

		if err == io.EOF {
			break
		}

		if buffer[0] == byte('ÿ') || rune(buffer[0]) == '�' ||
			buffer[0] == '\xff' {
			msgCount++
		}
	}

	return msgCount
}

func errorLogger(msg string, msgType int) {
	var msgTypeStr string

	// Determine what kind of error this is
	if msgType == 0 {
		msgTypeStr = "DBUG"
	} else if msgType == 1 {
		msgTypeStr = "INFO"
	} else if msgType == 2 {
		msgTypeStr = "WARN"
	} else if msgType == 3 {
		msgTypeStr = "ERR!"
	} else {
		msgTypeStr = "UNKN"
	}

	// Print message to screen
	fmt.Printf("[%s : %s] %s\n", msgTypeStr,
		string(time.Now().Format("01-02-2006 15:04:05")), msg)
	msgToSave := "[" + msgTypeStr + " : " +
		string(time.Now().Format("01-02-2006 15:04:05")) + "] " + msg

	// Save message to slice
	errLog = append(errLog, msgToSave)
}

func logMessages() {
	if len(errLog) == 0 {
		return
	}

	file, err := os.OpenFile("_errlog.txt",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Println("[ERR!] Could not open file _errlog.txt for reading")
		os.Exit(1)
	}
	defer file.Close()

	for _, m := range errLog {
		fmt.Fprintln(file, m)
	}

	errLog = make([]string, 0)
}
