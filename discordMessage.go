/*
	discordMessage.go
	github.com/rafaelwi
*/

package main

// Struct for storing messages in the queue
type discordMessage struct {
	msg   string
	guild string
}
