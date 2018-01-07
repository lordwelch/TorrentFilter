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

func OrderedBy(fns ...func(int, int) bool) func(int, int) bool {
	return func(i, j int) bool {
		// Try all but the last comparison.
		for _, less := range fns {
			switch {
			case less(i, j):
				// i < j, so we have a decision.
				return true
			case less(j, i):
				// i > j, so we have a decision.
				return false
			}
			// i == j; try the next comparison.
		}
		// All comparisons to here said "equal", so just return whatever
		// the final comparison reports.
		return fns[len(fns)-1](i, j)
	}
}

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
	return Vt.Torrent.Meta.FilePath
}

func (Et *EpisodeTorrent) ByRelease(i, j int) bool {
	var (
		ii  int
		ij  int
		ret bool
	)
	ii = Et.Release[Et.Ep[i].Release]
	ij = Et.Release[Et.Ep[j].Release]
	if ii == 0 {
		ii = 999999
	}
	if ij == 0 {
		ij = 999999
	}
	if ii == ij {
		ret = Et.Ep[i].Release > Et.Ep[j].Release
		//fmt.Println(Et.Ep[i].Release, ">", Et.Ep[j].Release, "=", ret, Et)
	} else {
		ret = ii < ij
	}
	return ret
}

func (Et *EpisodeTorrent) ByTag(i, j int) bool {
	var (
		ii  int
		ij  int
		ret bool
	)
	for k := range Et.Ep[i].Tags {
		if Et.Tags[k] > 0 {
			ii++
		}
	}
	for k := range Et.Ep[j].Tags {
		if Et.Tags[k] > 0 {
			ij++
		}
	}

	if ii == ij {
		ret = len(Et.Ep[i].Tags) < len(Et.Ep[j].Tags)
		//fmt.Println(len(Et.Ep[i].Tags), "<", len(Et.Ep[j].Tags), "=", ret)
	} else {
		ret = ii > ij
	}

	return ret
}

func (Et *EpisodeTorrent) ByRes(i, j int) bool {
	var ret bool
	ret = Et.Ep[i].Resolution > Et.Ep[j].Resolution
	//fmt.Println(Et.Ep[i].Resolution, ">", Et.Ep[j].Resolution, "=", ret)

	if Et.Res == Et.Ep[i].Resolution && Et.Ep[i].Resolution != Et.Ep[j].Resolution {
		ret = true
	}
	return ret
}

func (Et *EpisodeTorrent) Len() int {
	return len(Et.Ep)
}

func (Et *EpisodeTorrent) Swap(i, j int) {
	Et.Ep[i], Et.Ep[j] = Et.Ep[j], Et.Ep[i]
	//fmt.Println(Et.Ep)
}

func (Et *EpisodeTorrent) Less(i, j int) bool {
	return OrderedBy(Et.ByRelease, Et.ByRes, Et.ByTag)(i, j)
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
