package main

import (
	"bytes"
	"net/http"
	"net/http/cookiejar"
	"strconv"

	"fmt"
	"time"

	"regexp"

	"encoding/json"

	"io/ioutil"

	"os"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/mmcdole/gofeed"
)

type Anime struct {
	ID                     uint           `json:"id"`
	Name                   string         `gorm:"index" json:"name"`
	Thumbnail              string         `json:"thumbnail"`
	AutoDownloadResolution uint           `json:"auto_download_resolution"`
	AutoDownloadGroupID    uint           `json:"auto_download_group_id"`
	SubbingGroups          []SubbingGroup `json:"subbing_groups" gorm:"many2many:anime_subbing_groups;"`
	Episodes               []Episode      `json:"episodes"`
}

type Episode struct {
	ID       uint      `json:"id"`
	AnimeID  uint      `json:"anime_id"`
	Number   uint      `gorm:"index" json:"number"`
	Torrents []Torrent `json:"torrents"`
}

type Torrent struct {
	ID             uint      `json:"id"`
	EpisodeID      uint      `json:"episode_id"`
	SubbingGroupID uint      `json:"subbing_group_id"`
	Link           string    `json:"link"`
	Extension      string    `json:"extension"`
	PubDate        time.Time `gorm:"index" json:"pub_date"`
	Resolution     uint      `json:"resolution"`
	Title          string    `json:"title"`
	Downloaded     bool      `gorm:"index" json:"downloaded"`
}

type SubbingGroup struct {
	ID   uint   `json:"id"`
	Name string `gorm:"index" json:"name"`
}

type GoogleSearchResult struct {
	Kind string `json:"kind"`
	URL  struct {
		Type     string `json:"type"`
		Template string `json:"template"`
	} `json:"url"`
	Queries struct {
		Request []struct {
			Title          string `json:"title"`
			TotalResults   string `json:"totalResults"`
			SearchTerms    string `json:"searchTerms"`
			Count          int    `json:"count"`
			StartIndex     int    `json:"startIndex"`
			InputEncoding  string `json:"inputEncoding"`
			OutputEncoding string `json:"outputEncoding"`
			Safe           string `json:"safe"`
			Cx             string `json:"cx"`
			SearchType     string `json:"searchType"`
			ImgSize        string `json:"imgSize"`
		} `json:"request"`
		NextPage []struct {
			Title          string `json:"title"`
			TotalResults   string `json:"totalResults"`
			SearchTerms    string `json:"searchTerms"`
			Count          int    `json:"count"`
			StartIndex     int    `json:"startIndex"`
			InputEncoding  string `json:"inputEncoding"`
			OutputEncoding string `json:"outputEncoding"`
			Safe           string `json:"safe"`
			Cx             string `json:"cx"`
			SearchType     string `json:"searchType"`
			ImgSize        string `json:"imgSize"`
		} `json:"nextPage"`
	} `json:"queries"`
	Context struct {
		Title string `json:"title"`
	} `json:"context"`
	SearchInformation struct {
		SearchTime            float64 `json:"searchTime"`
		FormattedSearchTime   string  `json:"formattedSearchTime"`
		TotalResults          string  `json:"totalResults"`
		FormattedTotalResults string  `json:"formattedTotalResults"`
	} `json:"searchInformation"`
	Items []struct {
		Kind        string `json:"kind"`
		Title       string `json:"title"`
		HTMLTitle   string `json:"htmlTitle"`
		Link        string `json:"link"`
		DisplayLink string `json:"displayLink"`
		Snippet     string `json:"snippet"`
		HTMLSnippet string `json:"htmlSnippet"`
		Mime        string `json:"mime"`
		Image       struct {
			ContextLink     string `json:"contextLink"`
			Height          int    `json:"height"`
			Width           int    `json:"width"`
			ByteSize        int    `json:"byteSize"`
			ThumbnailLink   string `json:"thumbnailLink"`
			ThumbnailHeight int    `json:"thumbnailHeight"`
			ThumbnailWidth  int    `json:"thumbnailWidth"`
		} `json:"image"`
	} `json:"items"`
}

var db *gorm.DB

type AutoDownloadRequest struct {
	Resolution     uint `json:"resolution"`
	SubbingGroupID uint `json:"subbing_group_id"`
}

func main() {
	var err error
	db, err = gorm.Open("sqlite3", "data/nyaatorrentler.db")
	if err != nil {
		panic("Failed to connect to database...")
	}
	// Migrate the schema
	db.AutoMigrate(&Anime{}, &Episode{}, &Torrent{}, &SubbingGroup{})

	initTicker()

	e := echo.New()
	e.Use(middleware.Logger())
	e.Static("/", "public")
	e.GET("/api/animes", func(c echo.Context) error {
		var anime []Anime
		//db.Joins("LEFT JOIN episodes ON episodes.anime_id = animes.ID").Joins("LEFT JOIN torrents ON torrents.episode_id = episodes.ID").Limit(100).Order("ID desc").Find(&anime)
		db.
			Preload("Episodes").
			Preload("SubbingGroups").
			Preload("Episodes.Torrents", func(db *gorm.DB) *gorm.DB {
				return db.Order("torrents.pubDate DESC")
			}).
			Find(&anime)
		return c.JSON(http.StatusOK, anime)
	})

	e.POST("/api/animes/:id/toggle", func(c echo.Context) error {
		id, _ := strconv.Atoi(c.Param("id"))
		anime := Anime{ID: uint(id)}
		var autoDownloadRequest AutoDownloadRequest
		c.Bind(&autoDownloadRequest)
		db.First(&anime)
		if anime.AutoDownloadResolution == 0 {
			anime.AutoDownloadResolution = autoDownloadRequest.Resolution
			anime.AutoDownloadGroupID = autoDownloadRequest.SubbingGroupID
		} else {
			anime.AutoDownloadResolution = 0
			anime.AutoDownloadGroupID = 0
		}
		db.Save(anime)
		return c.JSON(http.StatusOK, &anime)
	})

	e.POST("/api/torrent/:id/download", func(c echo.Context) error {
		var torrent Torrent
		db.First(&torrent, c.Param("id"))
		download(torrent)
		return c.JSON(http.StatusOK, torrent)
	})

	e.Logger.Fatal(e.Start(":1323"))
}

func UpdateThumbnail(a *Anime) {
	client := http.Client{}
	req, _ := http.NewRequest("GET", "https://www.googleapis.com/customsearch/v1", nil)
	q := req.URL.Query()
	q.Add("q", a.Name)
	q.Add("searchType", "image")
	q.Add("imgSize", "large")
	q.Add("key", os.Getenv("CH_COMPILE_NYAA_GOOGLE_KEY"))
	q.Add("cx", os.Getenv("CH_COMPILE_NYAA_GOOGLE_CX"))

	req.URL.RawQuery = q.Encode()

	res, _ := client.Do(req)

	resBody, _ := ioutil.ReadAll(res.Body)
	var searchResult GoogleSearchResult
	json.Unmarshal(resBody, &searchResult)
	fmt.Printf("%+v\n", searchResult)

	a.Thumbnail = searchResult.Items[0].Link
	fmt.Printf("THUMBUP: %+v\n", a)
}

func (a Anime) AfterUpdate(db *gorm.DB) {
	fmt.Println("Afterupdate!")
	if a.AutoDownloadResolution != 0 {
		var torrents []Torrent
		db.
			Joins("INNER JOIN episodes ON torrents.episode_id = episodes.ID").
			Joins("INNER JOIN animes ON episodes.anime_id = animes.ID").
			Where("animes.ID = ? AND torrents.Resolution = ? AND torrents.Downloaded = ? AND torrents.SubbingGroupID = ?", a.ID, a.AutoDownloadResolution, false, a.AutoDownloadGroupID).
			Find(&torrents)

		for _, t := range torrents {
			fmt.Printf("Downloading %+v", t)
			download(t)
		}
	}
}

// DelugeMethod represents a generic Deluge JSON Method
type DelugeMethod struct {
	ID     uint     `json:"id"`
	Method string   `json:"method"`
	Params []string `json:"params"`
}

// DelugeWebAddTorrentMethod represents a generic Deluge JSON Method
type DelugeWebAddTorrentMethod struct {
	ID     uint              `json:"id"`
	Method string            `json:"method"`
	Params [][]DelugeTorrent `json:"params"`
}

// DelugeTorrent represents a torrent
type DelugeTorrent struct {
	Path    string               `json:"path"`
	Options DelugeTorrentOptions `json:"options"`
}

// DelugeTorrentOptions reprents
type DelugeTorrentOptions struct {
	FilePriorities            []string `json:"file_priorities"`
	CompactAllocation         bool     `json:"compact_allocation"`
	DownloadLocation          string   `json:"download_location"`
	MoveCompleted             bool     `json:"move_completed"`
	MoveCompletedPath         string   `json:"move_completed_path"`
	MaxConnections            int      `json:"max_connections"`
	MaxDownloadSpeed          int      `json:"max_download_speed"`
	MaxUploadSlots            int      `json:"max_upload_slots"`
	MaxUploadSpeed            int      `json:"max_upload_speed"`
	PrioritizeFirstLastPieces bool     `json:"prioritize_first_last_pieces"`
}

func download(t Torrent) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
	}

	login, _ := json.Marshal(&DelugeMethod{
		Method: "auth.login",
		Params: []string{""},
		ID:     1,
	})

	fmt.Printf("data: %+v\n", string(login))

	res, err := client.Post(os.Getenv("CH_COMPILE_NYAA_DELUGE_URL")+"/json", "application/json", bytes.NewReader(login))

	fmt.Println("err:", err)
	body, _ := ioutil.ReadAll(res.Body)
	fmt.Printf("status: %d\nbody:%+v\n", res.StatusCode, string(body))

	torrentJson := DelugeTorrent{
		Path: t.Link,
		Options: DelugeTorrentOptions{
			CompactAllocation:         false,
			DownloadLocation:          "/downloads",
			FilePriorities:            []string{},
			MaxConnections:            -1,
			MaxDownloadSpeed:          -1,
			MaxUploadSlots:            -1,
			MaxUploadSpeed:            -1,
			MoveCompleted:             false,
			MoveCompletedPath:         "/downloads",
			PrioritizeFirstLastPieces: false,
		},
	}

	webAddTorrents, _ := json.Marshal(&DelugeWebAddTorrentMethod{
		ID:     2,
		Method: "web.add_torrents",
		Params: [][]DelugeTorrent{[]DelugeTorrent{torrentJson}},
	})

	fmt.Printf("Sending to DELUGE: %+v", string(webAddTorrents))

	res, err = client.Post(os.Getenv("CH_COMPILE_NYAA_DELUGE_URL")+"/json", "application/json", bytes.NewReader(webAddTorrents))
	fmt.Println("err:", err)
	body, _ = ioutil.ReadAll(res.Body)
	fmt.Printf("status: %d\nbody:%+v\n", res.StatusCode, string(body))
}

func initTicker() {
	ticker := time.NewTicker(time.Second * 10)
	go func() {
		titleRe := regexp.MustCompile(`^\[(.+?)\]\s+([^\[\]]+?)\s*-\s+(\d+)\s+.*(720|1080|480).*\.(mp4|mkv)$`)

		for range ticker.C {
			fp := gofeed.NewParser()
			feed, err := fp.ParseURL("https://nyaa.si/?page=rss&m=true&c=1_2&f=0&q=720p")

			if err != nil {
				panic(err)
			}

			fmt.Println(feed.Title)

			for _, element := range feed.Items {
				matches := titleRe.FindStringSubmatch(element.Title)
				fmt.Println(matches)
				if len(matches) == 0 {
					continue
				}
				// 0: full
				// 1: group
				// 2: Anime
				// 3: Episode Number
				// 4: Resolution
				// 5: Extension
				var anime Anime
				fmt.Println(matches[2])
				//db.Where(&Anime{Name: matches[2]}).FirstOrCreate(&anime)
				db.FirstOrInit(&anime, &Anime{Name: matches[2]})

				fmt.Println(anime)
				if db.NewRecord(anime) {
					UpdateThumbnail(&anime)
				}

				subbingGroup := SubbingGroup{}
				db.FirstOrCreate(&subbingGroup, &SubbingGroup{Name: matches[1]})

				anime.SubbingGroups = append(anime.SubbingGroups, subbingGroup)

				db.Save(&anime)

				episode := Episode{}
				episodeNr, _ := strconv.Atoi(matches[3])
				db.FirstOrCreate(&episode, &Episode{Number: uint(episodeNr), AnimeID: anime.ID})

				resolution, _ := strconv.Atoi(matches[4])

				torrent := Torrent{}
				db.FirstOrInit(&torrent, &Torrent{
					Title: element.Title,
				})

				torrent.EpisodeID = episode.ID
				torrent.Extension = matches[5]
				torrent.Link = element.Link
				torrent.PubDate = *element.PublishedParsed
				torrent.Resolution = uint(resolution)
				torrent.Title = element.Title
				torrent.SubbingGroupID = subbingGroup.ID

				if db.NewRecord(&torrent) {
					db.Save(&torrent)
				}
			}

		}
	}()
}
