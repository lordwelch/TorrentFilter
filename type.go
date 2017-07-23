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
	P2080
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
	size    int64
}

type MediaTorrent struct {
	Torrent
	Title  string
	Format Fmt
	Source Src
	Date   string
	Tags   map[string]bool //ALTERNATIVE.CUT, CONVERT, COLORIZED, DC, DIRFIX, DUBBED, EXTENDED, FINAL, INTERNAL, NFOFIX, OAR, OM, PPV, PROPER, REAL, REMASTERED, READNFO, REPACK, RERIP, SAMPLEFIX, SOURCE.SAMPLE, SUBBED, UNCENSORED, UNRATED, UNCUT, WEST.FEED, and WS
}

type EpisodeTorrent struct {
	VT      []MediaTorrent
	Episode string
	Season  string
}

type SeriesTorrent struct {
	Episodes []EpisodeTorrent
	title    string
}

type Interface interface {
	Title() string
}

func NewTorrent(mt MetaTorrent) (T *Torrent) {
	T = new(Torrent)
	if mt.Info.Length == 0 {
		for i, path := range mt.Info.Files {
			for _, file := range path.Path {
				if file[len(file)-3:] == "mkv" || file[len(file)-3:] == "mp4" {
					T.size = mt.Info.Files[i].Length
					T.Name = file
				}
			}
		}
	} else {
		T.Name = mt.Info.Name
		T.size = mt.Info.Length
	}
	T.Comment = mt.Comment
	T.Creator = mt.CreatedBy
	T.Meta = mt
	return T
}

func (Mt *MetaTorrent) Load(r io.Reader) error {
	return bencode.NewDecoder(r).Decode(Mt)
}

func (T *MediaTorrent) Process() error {
	var (
		err                    error
		r                      rune
		exit, tag, year, month bool
		dateIndex
		indexes                []integer
		//str                    string
		re = [4]*regexp.Regexp{regexp.MustCompile(`^[s](\d{2})[s](\d{2})$`), regexp.MustCompile(`^\d{4}$`), regexp.MustCompile(`^\d\d$`), regexp.MustCompile(`^([A-Za-z]{3,10})\[([A-Za-z]{4,6})\]$`)}
	)
	strs := strings.Split(T.Name)
	for i, str := range strs {
		if re[1].MatchString(strings.ToLower(str)) {
			indexes = append(indexes, i)
		}
	}
	if len(indexes) > 0 {
		T.Date = indexes[len(indexes)-1]
	}
	for i, str := range strs {
		fmt.Println(str)
		if tag {
			T.tag(str)
			if i == len(strs)-1 && strings.Contains(str, "-") {
				tags := strings.Split(str, "-")
				T.tag(tags[0])
				eztv := strings.Split(tags, "[")
				T.Release = eztv[0]
				T.Creator = eztv[1][:len(eztv[1])-1]
			}
		} else {
			switch {
			case year:
				year = false
				if re[2].MatchString(strings.ToLower(str)) {
					month = true
					T.Date += "." + str
					continue
				}
				if re[1].MatchString(strings.ToLower(str)) {
					T.Title += T.Date
					T.Date = str
					tag = true
					continue
				}
				T.tag(str)
				T.Title += " " + str
			case month:
				T.Date += "." + str
				tag = true
			case re[0].MatchString(strings.ToLower(str)):
				tag = true
				match := re[0].FindStringSubmatch(strings.ToLower(str))
				T.Season = match[1]
				T.Episode = match[2]
			case re[1].MatchString(strings.ToLower(str)):
				year = true
				T.Date = str
				continue
			default:
				T.Title += " " + str
			}
		}
	}

	T.Title = strings.TrimSpace(T.Title)
	return nil
}

func (T *MediaTorrent) tag(str string) {
	switch strings.ToLower(str) {
	case "x264", "h", "264", "h264", "h264":
		T.Format = X264
	case "xvid", "divx":
		T.Format = XVID
	case "720p":
		T.P720 = true
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
