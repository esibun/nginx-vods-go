package main

import (
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/gorp.v1"
	"html/template"
	"strings"
	"time"
)

var dbmap *gorp.DbMap
var r *gin.Engine

type Video struct {
	Filename  string `db:"filename"`
	Time      int64  `db:"time"`
	Name      string `db:"name"`
	Duration  int64  `db:"duration"`
	Thumbnail string `db:"thumbnail"`
}

func initDb() *gorp.DbMap {
	db, err := sql.Open("mysql", "username:password@/vods")
	if err != nil {
		panic(err)
	}

	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{"InnoDB", "UTF8"}}

	dbmap.AddTableWithName(Video{}, "videos")

	err = dbmap.CreateTablesIfNotExists()
	if err != nil {
		panic(err)
	}

	return dbmap
}

func GetLiveStatus() string {
	var status bool
	err := dbmap.SelectOne(&status, `select value from api_cache where cachekey = 'media_is_live'`)
	if err == sql.ErrNoRows {
		return ""
	} else if err != nil {
		panic(err)
	}
	if status == true {
		return "online"
	} else {
		return "offline"
	}
}

func GetVideoTitle(id string) string {
	var title string
	err := dbmap.SelectOne(&title, `select name from videos where filename = '`+id+`'`)
	if err == sql.ErrNoRows {
		return ""
	} else if err != nil {
		panic(err)
	}
	return title
}

func GetVideos() *[]Video {
	var videos []Video
	_, err := dbmap.Select(&videos, `select filename, UNIX_TIMESTAMP(time) AS time, name, duration, thumbnail from videos where duration > 0 order by time desc`)
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		panic(err)
	}
	return &videos
}

func FormatDuration(d int64) string {
	duration := (time.Duration(d) * time.Second)
	hours := fmt.Sprintf("%d", int(duration.Hours()))
	seconds := fmt.Sprintf("%02d", int(duration.Seconds())%60)
	if hours != "0" {
		minutes := fmt.Sprintf("%02d", int(duration.Minutes())%60)
		return strings.Join([]string{hours, minutes, seconds}, ":")
	} else {
		minutes := fmt.Sprintf("%d", int(duration.Minutes())%60)
		return strings.Join([]string{minutes, seconds}, ":")
	}
}

func FormatTime(t int64) string {
	return time.Unix(t, 0).Format("1/2/2006 3:04:05 PM")
}

func GetVideoList(c *gin.Context) {
	obj := gin.H{"name": "esi", "videos": GetVideos(), "livestatus": GetLiveStatus()}
	funcMap := template.FuncMap{
		"formatTime":     FormatTime,
		"formatDuration": FormatDuration,
	}
	t := template.New("ListVideos").Funcs(funcMap)
	p, err := t.ParseFiles("templates/ListVideos.tmpl")
	if err != nil {
		panic(err)
	}

	r.SetHTMLTemplate(p)
	c.HTML(200, "base", obj)
}

func ShowVideo(c *gin.Context) {
	obj := gin.H{"name": "esi", "id": c.Params.ByName("id"), "title": GetVideoTitle(c.Params.ByName("id"))}
	t := template.New("ShowVideos")
	p, err := t.ParseFiles("templates/ShowVideo.tmpl")
	if err != nil {
		panic(err)
	}

	r.SetHTMLTemplate(p)
	c.HTML(200, "base", obj)
}

func main() {
	dbmap = initDb()
	defer dbmap.Db.Close()

	r = gin.New()

	r.Static("/css", "./css")
	r.Static("/images", "./images")
	r.Static("/thumbnails", "./thumbnails")
	r.Static("/videos", "./videos")

	r.GET("/", GetVideoList)
	r.GET("/video/:id", ShowVideo)
	r.Run(":3000")
}
