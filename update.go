package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Status struct {
	Mature              bool
	Status              string
	BroadcasterLanguage string
	DisplayName         string
	Game                string
	Delay               int64
	Language            string
	Id                  int64
	Name                string
	Created             string
	Updated             string
	Logo                string
	Banner              string
	VideoBanner         string
	Background          string
	ProfileBanner       string
	ProfileBannerColor  string
	Partner             bool
	Url                 string
	Views               int64
	Followers           int64
	Links               LinkList
}

type LinkList struct {
	Self          string
	Follows       string
	Commercial    string
	StreamKey     string
	Chat          string
	Features      string
	Subscriptions string
	Editors       string
	Teams         string
	Videos        string
}

type CacheKeys struct {
	Status string
	Time   int64
}

func GetTitle() string {
	var expiry int64
	var title string
	err := dbmap.SelectOne(&expiry, `select unix_timestamp(expiry) as expiry from api_cache where cachekey = 'media_status'`)
	if err != nil && err != sql.ErrNoRows {
		panic(err)
	}
	if time.Now().Unix() > expiry {
		_, err := dbmap.Exec(`delete from api_cache where cachekey = 'media_status'`)
		if err != nil {
			panic(err)
		}
		r, err := http.Get("https://api.twitch.tv/kraken/channels/" + twitch_username)
		if err != nil {
			panic(err)
		}
		defer r.Body.Close()
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		var j Status
		json.Unmarshal(b, &j)
		_, err = dbmap.Exec(`insert into api_cache (cachekey, value, expiry) values ('media_status', ?, from_unixtime(?))`, j.Status, fmt.Sprintf("%d", time.Now().Unix()+60))
		if err != nil {
			panic(err)
		}
		return j.Status
	} else {
		err := dbmap.SelectOne(&title, `select value from api_cache where cachekey = 'media_status'`)
		if err != nil {
			panic(err)
		}
		return title
	}
}

func GetDuration(video string) int64 {
	c1 := exec.Command("ffprobe", "-loglevel", "error", "-show_streams", "./videos/"+video+".mp4")
	c2 := exec.Command("grep", "duration=")
	c3 := exec.Command("cut", "-f2", "-d=")
	c4 := exec.Command("tail", "-n", "1")

	c2.Stdin, _ = c1.StdoutPipe()
	c3.Stdin, _ = c2.StdoutPipe()
	c4.Stdin, _ = c3.StdoutPipe()

	c1.Start()
	c2.Start()
	c3.Start()
	out, err := c4.Output()
	c3.Wait()
	c2.Wait()
	c1.Wait()

	if err != nil {
		panic(err)
	}

	durationF, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0
	}

	return int64(durationF)
}

func GetVideoInfo(array []Video, name string) (Video, error) {
	for _, value := range array {
		if value.Filename == name {
			return value, nil
		}
	}
	return array[0], errors.New("Unable to find video.")
}

func InsertVideo(v Video) error {
	_, err := dbmap.Exec(`insert into videos values (?, from_unixtime(?), ?, ?, ?)`, v.Filename, v.Time, v.Name, v.Duration, v.Thumbnail)
	return err
}

func ParseTime(path string) int64 {
	path = strings.SplitN(path, "-", 2)[1]
	loc, err := time.LoadLocation("Local")
	if err != nil {
		panic(err)
	}
	timestamp, err := time.ParseInLocation("2006-01-02_15-04-05", path, loc)
	if err != nil {
		panic(err)
	}
	return timestamp.Unix()
}

func GenerateThumbnail(path string) error {
	return exec.Command("ffmpeg", "-i", "./videos/"+path+".mp4", "-vf", "thumbnail,scale=320:180", "-frames:v", "1", "./thumbnails/"+path+".thumb.png").Run()
}

func UpdateStatus(c *gin.Context) {
	var status bool
	var response bytes.Buffer
	var videos []Video

	response.Write([]byte("Processing new videos:\n"))

	c.Request.ParseForm()

	call := c.Request.Form.Get("call")
	name := c.Request.Form.Get("name")
	if call == "publish" {
		status = true
	} else {
		status = false
	}
	expiry := time.Now().Unix() + 60
	_, err := dbmap.Exec(`update api_cache set value = ?, expiry = from_unixtime(?) where cachekey = 'media_is_live'`, status, expiry)
	if err != nil {
		panic(err)
	}
	_, err = dbmap.Exec(`update api_cache set value = ?, expiry = from_unixtime(?) where cachekey = 'media_live_on'`, name, expiry)
	if err != nil {
		panic(err)
	}

	videos = *GetVideos()

	filepath.Walk("./videos", func(path string, _ os.FileInfo, _ error) error {
		if path == "./videos" {
			return nil
		}

		path = strings.Replace(path, "videos/", "", 1)
		path = strings.Replace(path, ".mp4", "", 1)

		info, err := GetVideoInfo(videos, path)
		if err != nil {
			duration := GetDuration(path)
			if duration > 0 {
				var thumbnail string
				err = GenerateThumbnail(path)
				if err != nil {
					thumbnail = "no"
				} else {
					thumbnail = "yes"
				}
				err = InsertVideo(Video{
					Filename:  path,
					Time:      ParseTime(path),
					Name:      GetTitle(),
					Duration:  duration,
					Thumbnail: thumbnail,
				})
				if err != nil {
					panic(err)
				}
				response.Write([]byte("new vid: " + path + "\n"))
			}
		} else {
			duration := GetDuration(path)

			if duration != info.Duration {
				_, err := dbmap.Delete(&info)
				if err != nil {
					panic(err)
				}
				err = InsertVideo(Video{
					Filename:  path,
					Time:      info.Time,
					Name:      GetTitle(),
					Duration:  duration,
					Thumbnail: info.Thumbnail,
				})
				if err != nil {
					panic(err)
				}
				response.Write([]byte("updated vid: " + path + "\n"))
			}

		}
		return nil
	})
	if response.Len() == 23 {
		response.Write([]byte("(no new videos to process)"))
	}
	c.String(http.StatusOK, response.String())
}
