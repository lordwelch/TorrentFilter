package TorrentFilter

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

type EpisodeTorrent []SceneVideoTorrent
type SeasonTorrent map[string]EpisodeTorrent
type SeriesTorrent map[string]SeasonTorrent

var (
	Tags    = make(map[string]int, 5)
	Res     scene.Res
	Release = make(map[string]int, 5)
)

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

func (Vt SceneVideoTorrent) Score() (score int) {
	if filepath.Ext(Vt.Name) == ".mkv" {
		score += 1000
	}
	if Vt.Resolution == Res {
		score += 900
	} else {
		score += int(Vt.Resolution) * 100
	}
	score += Release[Vt.Release] + 1
	for k := range Vt.Tags {
		score += Tags[k]
	}
	return
}

func (Vt SceneVideoTorrent) String() string {
	return Vt.Scene.String()
}

func (Et EpisodeTorrent) Len() int {
	return len(Et)
}

func (Et EpisodeTorrent) Swap(i, j int) {
	Et[i], Et[j] = Et[j], Et[i]
}

func (Et EpisodeTorrent) Less(i, j int) bool {
	return Et[i].Score() > Et[j].Score()
}

func (Et EpisodeTorrent) Sort() {
	sort.Stable(Et)
}

func (St SeriesTorrent) SearchHash(hash string) *SceneVideoTorrent {
	for _, v := range St {
		for _, v2 := range v {
			for _, v3 := range v2 {
				if v3.Meta.Hash == hash {
					return &v3
				}
			}
		}
	}
	return nil
}

func (St SeriesTorrent) Addtorrent(torrent SceneVideoTorrent) {
	_, ok := St[torrent.Season]
	if !ok {
		St[torrent.Season] = make(SeasonTorrent, 20)
	}

	St[torrent.Season][torrent.Episode] = append(St[torrent.Season][torrent.Episode], torrent)

}
