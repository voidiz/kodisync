<p align="center">
  <img src="./assets/kodisync.svg" width="200" alt="kodisync">
</p>
<h2 align="center">kodisync</h2>
<p align="center">
  A program used to synchronize the playback of Kodi media players through the built-in webserver.
</p>

## Features
- Synchronizing desynchronized players (i.e. players that are ahead or behind other players)
- Pausing/playing all connected players when pausing/playing one of them

## Usage
1. `go get github.com/voidiz/kodisync` or download the [latest release](https://github.com/voidiz/kodisync/releases/latest).
2. [Enable the webserver](https://kodi.wiki/view/Webserver#Enabling_the_webserver) in Kodi (remember to set a username and password!) on every client you wish to sync. Remember to enable the settings "Allow remote control from applications on this/other system(s)" depending on where the clients are running ([however, do so with great caution](https://kodi.tv/article/kodi-remote-access-security-recommendations)).
3. Create a text file `identifiers.txt` in the working directory containing lines with the following format for each client: `hostname:port,username,password`. For example: `192.168.0.5:9090,kodi,mypassword`. (Note that the port isn't the port of the webserver, but the one specified in the [advanced settings](https://kodi.wiki/view/Advancedsettings.xml#jsonrpc) file which by default is set to 9090. In other words, if you're unsure, try 9090.)
4. Start the video you wish to play on all clients.
5. Launch the program.
