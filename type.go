package main

import (
	"io"
	"regexp"
	"strings"

	"github.com/zeebo/bencode"
)

type Src int
type Fmt int

const (
	HDTV Src = iota
	CAM
	DVD
	X264 Fmt = iota
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
	size    int
}

type TorrentVideo struct {
	*Torrent
	Episode  string
	Season   string
	Release  string
	Source   Src
	Format   Fmt
	Proper   bool
	Internal bool
	Repack   bool
	Nuked    bool
	P720     bool
}

func NewTorrent(mt MetaTorrent) (T *Torrent) {
	if mt.Info.Length == 0 {
		for i, path := range mt.Info.Files {
			for _, file := range path {
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
}

func (Mt *MetaTorrent) Load(r io.Reader) error {
	return bencode.NewDecoder(r).Decode(Mt)
}

func (T *TorrentVideo) Process() error {
	var (
		err       error
		r         rune
		exit, tag = bool
		str       string
		i         int
		re        = [...]regexp.Regexp{regexp.MustCompile(`[Ss](\d{2})[Ee](\d{2})`), regexp.MustCompile(`([A-Za-z]{3-10})\[([A-Z]{4-6})\]`)}
	)
	reader := strings.NewReader(T.Name)
	for err == nil && !exit {
		for err == nil && r != '.' && r != '-' {
			r, _, err = reader.ReadRune()
			if err != nil {
				return err
			}
			if r != '.' && r != '-' {
				str += string(r)
			}
		}
		if tag {
			switch str {
			case "NUKED":
				T.Nuked = true
			case "INTERNAL":
				T.Internal = true
			case "REPACK":
				T.Repack = true
			case "x264", "H", "264":
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
			switch {
			case re[0].Match(str):
				tag = true
				match := re[1].FindStringSubmatch(str)
				T.Season = match[1]
				T.Episode = match[2]
			case re[1].Match(str):
				match := re[1].FindStringSubmatch(str)
				T.Release = match[1]
				T.Creator = match[2]
			}
		}
	}
	return nil
}
