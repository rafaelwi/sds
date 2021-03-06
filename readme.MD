# SDS (Stuff Discord Says)
### A bot that parrots what people in your Discord server say

## What is this?
SDS is a Discord bot that will listen into your Discord server and record what 
people are saying and will, at times, randomly say something back. Due to its
random nature, I (currently) have no control over what messages are sent.

This bot is currently being hosted on my own Raspberry Pi in my house on Wi-Fi
so if it seems a bit slow, its because of that. However through my testing, it
has held up well.

SDS is built in Go using the [discordgo](github.com/bwmarrin/discordgo) wrapper.


## Features
- Listens to your messages
- Parrots them back about every 90 minutes
- A single instance runs the bot on multiple servers
- Typing `./machine` gives statistics on the RPI server :)


## Can I try it out?
Sure! Currently I am hosting an instance of the bot, which you can add to your
server by clicking [here](https://discordapp.com/api/oauth2/authorize?client_id=655575302176768020&permissions=522304&scope=bot). Be warned that since
this bot is currently in active development, it will go offline occassionally and
may spam your server with messsages. I am making it a priority at the moment to 
make only certain servers for testing to prevent this but it will take some time. 


If you do add this bot to your server, please let me know by messaging me on
Discord at @rafael#3399


## Can I host the bot?
Soon, this is a WIP.


## I have an issue while using this bot or I've found a bug. How do I report it?
Report all issues/bugs to [the issues section of this repo.](https://github.com/rafaelwi/sds/issues) I will not be responding to issues related to "Discord is down" or "The bot is down." I already know, it will be back up soon.


## I want to help. Where do I sign up?
Currently I am content with working by myself on this project, however if you are interested in helping or have an interesting idea for the bot, do not be afraid to shoot me a message on Discord at @rafael#3399. 
