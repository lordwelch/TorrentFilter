package main

import (
	"io"

	"github.com/lordwelch/SceneParse"
	"github.com/zeebo/bencode"
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
	Scene.Scene
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

func (s SeriesTorrent) Title() string {
	return s[0][0].Title
}
