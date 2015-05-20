# nginx-vods-go
This is a rewrite of the nginx-vods repository into Go.  You may find an example of this script in use at [rec.esibun.net](http://rec.esibun.net/)

Configuration
---
- Copy the files to a new directory
- Run `go build` in the directory
- Fill out the config values in *config.go*:
```go
const (
  mysql_database string = "database in which you wish to store vod/api information"
  mysql_username string = "username to log into mysql"
  mysql_password string = "password with which to log into mysql"
  
  twitch_username string = "twitch username to fetch titles from"
  nickname string = "nickname displayed in titles/headers"
)
```
- Optionally, configure your nginx instance to proxy requests to a path or a directory to the *nginx-vods-go* instance.

Usage
---
- Head to /update to update the list of videos in the videos directory and generate thumbnails and metadata for each video.
- You may configure nginx-rtmp's *on_publish* and *on_publish_done* directives to go to the update URL to additionally update live status as well as automatically updating the video list.
