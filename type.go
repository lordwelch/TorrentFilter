package main

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"timmy.narnian.us/git/timmy/scene"

	"github.com/zeebo/bencode"
)

type MetaTorrent struct {
	Hash         string     `bencode:"-"`
	FilePath     string     `bencode:"-"`
	Path         string     `bencode:"path"`
	Announce     string     `bencode:"announce"`
	Announcelist [][]string `bencode:"announce-list,omitempty"`
	Comment      string     `bencode:"comment,omitempty"`
	CreatedBy    string     `bencode:"created by,omitempty"`
	Info         struct {
		Name        string `bencode:"name"`
		PieceLength int64  `bencode:"piece length"`
		Length      int64  `bencode:"length,omitempty"`
		Pieces      string `bencode:"pieces"`
		Files       []struct {
			Length int64    `bencode:"length"`
			Path   []string `bencode:"path"`
		} `bencode:"files,omitempty"`
	} `bencode:"info"`
}

type Torrent struct {
	Meta    MetaTorrent
	Name    string
	Comment string
	Creator string
	Size    int64
}

type SceneVideoTorrent struct {
	Torrent
	scene.Scene
}

type EpisodeTorrent struct {
	Ep      []*SceneVideoTorrent
	Tags    map[string]int
	Res     scene.Res
	Release map[string]int
}
type SeasonTorrent map[string]*EpisodeTorrent
type SeriesTorrent map[string]SeasonTorrent

func NewTorrent(mt MetaTorrent) (T Torrent) {
	if mt.Info.Length == 0 {
		for _, path := range mt.Info.Files {
			for _, file := range path.Path {
				if file[len(file)-3:] == "mkv" || file[len(file)-3:] == "mp4" {
					T.Size = path.Length
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

func (Mt *MetaTorrent) ReadFile(r *os.File) error {
	err := bencode.NewDecoder(r).Decode(Mt)
	if err != nil {
		return err
	}
	str, _ := bencode.EncodeString(Mt.Info)
	Mt.Hash = fmt.Sprintf("%x", sha1.Sum([]byte(str)))
	Mt.FilePath, err = filepath.Abs(r.Name())
	if err != nil {
		return err
	}

	return nil
}

func (Vt SceneVideoTorrent) String() string {
	return Vt.Scene.String()
}

func (Et *EpisodeTorrent) Len() int {
	return len(Et.Ep)
}

func (Et *EpisodeTorrent) Swap(i, j int) {
	Et.Ep[i], Et.Ep[j] = Et.Ep[j], Et.Ep[i]
	//fmt.Println(Et.Ep)
}

func (Et *EpisodeTorrent) Less(i, j int) bool {
	return Et.score(i) > Et.score(j)
}

func (Et *EpisodeTorrent) score(i int) int {
	var score int
	if filepath.Ext(Et.Ep[i].Meta.FilePath) == ".mkv" {
		score += 1000
	}
	if Et.Ep[i].Resolution == Et.Res {
		score += 900
	} else {
		score += int(Et.Ep[i].Resolution) * 100
	}
	score += Et.Release[Et.Ep[i].Release] + 1
	for k := range Et.Ep[i].Tags {
		score += Et.Tags[k]
	}
	score += len(Et.Ep[i].Tags)
	return score
}

func (Et *EpisodeTorrent) Add(Vt *SceneVideoTorrent) {
	Et.Ep = append(Et.Ep, Vt)
	sort.Stable(Et)
}

func (St SeriesTorrent) SearchHash(hash string) *SceneVideoTorrent {
	for _, v := range St {
		for _, v2 := range v {
			for _, v3 := range v2.Ep {
				if v3.Meta.Hash == hash {
					return v3
				}
			}
		}
	}
	return &SceneVideoTorrent{}
}
