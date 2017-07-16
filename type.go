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

const (
	HDTV Src = iota + 1
	CAM
	DVD
)
const (
	X264 Fmt = iota + 1
	XVID
)

type MetaTorrent struct {
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
	Name    string
	Comment string
	Creator string
	size    int64
}

type TorrentVideo struct {
	*Torrent
	Title    string
	Release  string
	Source   Src
	Format   Fmt
	Proper   bool
	Internal bool
	Repack   bool
	Nuked    bool
	P720     bool
}

type TorrentEpisode struct {
	*TorrentVideo
	Episode string
	Season  string
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
	return T
}

func (Mt *MetaTorrent) Load(r io.Reader) error {
	return bencode.NewDecoder(r).Decode(Mt)
}

func (T *TorrentVideo) Process() error {
	var (
		err       error
		r         rune
		exit, tag bool
		str       string
		re        = [2]*regexp.Regexp{regexp.MustCompile(`[Ss](\d{2})[Ee](\d{2})`), regexp.MustCompile(`([A-Za-z]{3,10})\[([A-Za-z]{4,6})\]`)}
	)
	reader := strings.NewReader(T.Name)
	for err == nil && !exit {
		for err == nil && r != '.' && r != '-' {
			r, _, err = reader.ReadRune()
			if err == io.EOF {
				exit = true
				break
			} else if err != nil {
				return err
			}
			if r != '.' && r != '-' {
				str += string(r)
			}
		}
		fmt.Println(str)
		if tag {
			switch str {
			case "NUKED":
				T.Nuked = true
			case "REPACK":
				T.Repack = true
			case "x264", "H", "264", "h264", "H264":
				T.Format = X264
			case "XviD", "DivX":
				T.Format = XVID
			case "720p":
				T.P720 = true
			case "PROPER", "READ", "NFO":
				T.Proper = true
			case "INTERNAL":
				T.Internal = true
			case "HDTV", "DVDRip":
				T.Source = HDTV
			case "CAM":
				T.Source = CAM
			case "DVD":
				T.Source = DVD
			}
			if re[1].MatchString(str) {
				match := re[1].FindStringSubmatch(str)
				T.Release = match[1]
				T.Creator = match[2]
			}

		} else {
			if re[0].MatchString(str) {
				tag = true
				match := re[0].FindStringSubmatch(str)
				T.Season = match[1]
				T.Episode = match[2]
			} else {
				T.Title += str
			}
		}
		fmt.Println(re[1])
		fmt.Printf("tag: %t\n", re[1].MatchString(str))

		if r == '.' || r == '-' {
			r = ' '
			str = ""
		}
	}
	return nil
}
