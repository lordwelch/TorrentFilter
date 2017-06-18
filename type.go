package main

import (
	"strings"

	"github.com/zeebo/bencode"
)

type Src int
type Fmt int
type Std int

const (
	HDTV Src = iota
	CAM
	DVD
	X264 Fmt = iota
	Xvid
	Ntsc Std = iota
	Pal
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
	Release  string
	Source   Src
	Format   Fmt
	Standard Std
	Retail   bool
	Proper   bool
	Internal bool
	Dl       bool
	Recode   bool
	Repack   bool
	Nuked    bool
	p720     bool
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

func (T TorrentVideo) Process() {
	reader := strings.NewReader(T.Name)

}
