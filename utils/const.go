package utils

const (
	// message to send to stop media
	STOPBODY = `{"id":1,"jsonrpc":"2.0","method":"Player.Stop","params":{"playerid": %d}}`

	// get player id
	GETPLAYERBODY = `{"id":1, "jsonrpc":"2.0","method":"Player.GetActivePlayers"}`

	// the message to lauch local media
	BODY = `{
	"id":1,"jsonrpc":"2.0",
	"method":"Player.Open",
	"params": {
		"item": {
		   "file": "%s"
		 }
	 }
 }`

	YOUTUBEAPI = `{"jsonrpc": "2.0",
	"method": "Player.Open",
	"params":{"item": {"file" : "plugin://plugin.video.youtube/?action=play_video&videoid=%s" }},
	"id" : "1"}`

	PLAYSENDTOKODIAPI = `{"jsonrpc": "2.0",
	"method": "Player.Open",
	"params":{"item": {"file" : "plugin://plugin.video.sendtokodi/?%s" }},
	"id" : "1"}`

	ADDSENDTOKODIAPI = `{"jsonrpc": "2.0",
	"method": "Playlist.Add",
	"params":{
    "playlistid": 1,
    "item": {"file" : "plugin://plugin.video.sendtokodi/?%s" }
  },
	"id" : "1"}`
)
