package main

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/zeebo/bencode"
)

type Src int
type Fmt int
type Res int

const (
	HDTV Src = iota + 1
	AHDTV
	HRPDTV
)

const (
	X264 Fmt = iota + 1
	XVID
)

const (
	P480 Res = iota + 1
	P720
	P1080
	I1080
)

type MetaTorrent struct {
	Path         string
	Announce     string     `bencode:"announce"`
	Announcelist [][]string `bencode:"announce-list"`
	Comment      string     `bencode:"comment"`
	CreatedBy    string     `bencode:"created by"`
	Info         struct {
		Name         string `bencode:"name"`
		Piece_length int64  `bencode:"piece length"`
		Pieces       int64  `bencode:"pieces"`
		Length       int64  `bencode:"length"`
		Files        []struct {
			Length int64    `bencode:"length"`
			Path   []string `bencode:"path"`
		} `bencode:"files"`
	} `bencode:"info"`
}

type Torrent struct {
	Meta    MetaTorrent
	Name    string
	Comment string
	Creator string
	Size    int64
}

type MediaTorrent struct {
	Torrent
	Title      string
	Format     Fmt
	Source     Src
	Date       string
	Release    string
	Tags       map[string]bool //ALTERNATIVE.CUT, CONVERT, COLORIZED, DC, DIRFIX, DUBBED, EXTENDED, FINAL, INTERNAL, NFOFIX, OAR, OM, PPV, PROPER, REAL, REMASTERED, READNFO, REPACK, RERIP, SAMPLEFIX, SOURCE.SAMPLE, SUBBED, UNCENSORED, UNRATED, UNCUT, WEST.FEED, and WS
	Episode    string
	Season     string
	Resolution Res
}

type EpisodeTorrent []MediaTorrent

type SeriesTorrent []EpisodeTorrent

type SeriesInterface interface {
	Title() string
}

func NewTorrent(mt MetaTorrent) (T Torrent) {
	if mt.Info.Length == 0 {
		for i, path := range mt.Info.Files {
			for _, file := range path.Path {
				if file[len(file)-3:] == "mkv" || file[len(file)-3:] == "mp4" {
					T.Size = mt.Info.Files[i].Length
					T.Name = file
				}
			}
		}
	} else {
		T.Name = mt.Info.Name
		T.Size = mt.Info.Length
	}
	T.Comment = mt.Comment
	T.Creator = mt.CreatedBy
	T.Meta = mt
	return T
}

func (Mt *MetaTorrent) Load(r io.Reader) error {
	return bencode.NewDecoder(r).Decode(Mt)
}

func (T *MediaTorrent) Process() {
	var (
		date, tag, year, month bool
		dateIndex              int
		indexes                []int
		re                     = [3]*regexp.Regexp{regexp.MustCompile(`^[s](\d{2})[s](\d{2})$`), regexp.MustCompile(`^\d{4}$`), regexp.MustCompile(`^\d\d$`)}
	)
	strs := strings.Split(T.Name, ".")
	for i, str := range strs {
		if re[1].MatchString(strings.ToLower(str)) {
			indexes = append(indexes, i)
		}
	}

	if len(indexes) > 0 {
		dateIndex = indexes[len(indexes)-1]
	}

	for i, str := range strs {
		fmt.Println(str)
		if tag {
			if i == len(strs)-1 && strings.Contains(str, "-") {
				tags := strings.Split(str, "-")
				str = tags[0]
				if strings.Contains(tags[1], "[") {
					eztv := strings.Split(tags[1], "[")
					T.Release = eztv[0]
					T.Creator = eztv[1][:len(eztv[1])-2]
				} else {
					T.Release = tags[1]
				}
			}
			T.tag(str)
		} else {
			switch {
			case i == dateIndex:
				T.Date = str
				date = true
				year = true
			case date:
				switch {
				case year:
					year = false
					if re[2].MatchString(strings.ToLower(str)) {
						month = true
						T.Date += "." + str
						continue
					}
					date = false
					T.tag(str)
					tag = true
				case month:
					T.Date += "." + str
					date = false
					tag = true
				}
			case re[0].MatchString(strings.ToLower(str)):
				tag = true
				match := re[0].FindStringSubmatch(strings.ToLower(str))
				T.Season = match[1]
				T.Episode = match[2]
			default:
				T.Title += " " + str
			}
		}
	}

	T.Title = strings.TrimSpace(T.Title)
}

func (T *MediaTorrent) tag(str string) {
	switch strings.ToLower(str) {
	case "x264", "h", "264", "h264":
		T.Format = X264
	case "xvid", "divx":
		T.Format = XVID
	case "480p":
		T.Resolution = P480
	case "720p":
		T.Resolution = P720
	case "1080p":
		T.Resolution = P1080
	case "1080i":
		T.Resolution = I1080
	case "hdtv":
		T.Source = HDTV
	case "ahdtv":
		T.Source = AHDTV
	case "hrpdtv":
		T.Source = HRPDTV
	default:
		T.Tags[strings.ToLower(str)] = true
	}
}

func (s SeriesTorrent) Title() string {
	return s[0][0].Title
}
