# kodisync
A program used to synchronize the playback of Kodi media players through the built-in webserver using the JSON-RPC protocol.

## Features
- Synchronizing desynchronized players (i.e. players that are ahead or behind other players)
- Pausing/playing all connected players when pausing/playing one of them

## Usage
1. `go get github.com/voidiz/kodisync && go build`
2. [Enable the webserver](https://kodi.wiki/view/Webserver#Enabling_the_webserver) in Kodi (remember to set a username and password!) on every client you wish to sync.
3. Create a text file `identifiers.txt` in the same directory as the executable containing lines with the following format for each client: `hostname:port,username,password`. For example: `192.168.0.5:9090,kodi,mypassword`. (Note that the port isn't the port of the webserver, but the one specified in the [advanced settings](https://kodi.wiki/view/Advancedsettings.xml#jsonrpc) file which by default is set to 9090. In other words, if you're unsure, try 9090.)
4. Start the video you wish to play on all clients.
5. Launch the program.
