package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"timmy.narnian.us/git/timmy/scene"

	"github.com/alexflint/go-arg"
	"github.com/lordwelch/transmission"
)

var (
	CurrentTorrents map[string]SeriesTorrent
	unselectedDir   string
	Transmission    *transmission.Client
	mutex           = new(sync.Mutex)

	args struct {
		RES      string   `arg:"help:Resolution preference [480/720/1080]"`
		RELEASE  []string `arg:"-r,help:Release group preference order."`
		NRELEASE []string `arg:"-R,help:Release groups to use only as a lost resort."`
		TAGS     []string `arg:"-t,help:Tags to prefer -t internal would choose an internal over another. Whichever file with the most tags is chosen. Release Group takes priority"`
		Series   []string `arg:"required,positional,help:TV series to download torrent file for"`
		NEW      bool     `arg:"-n,help:Only modify new torrents"`
		PATH     string   `arg:"-P,required,help:Path to torrent files"`
	}
)

func main() {
	var (
		stdC = make(chan *SceneVideoTorrent)
	)
	initialize()
	go stdinLoop(stdC)

	for {
		select {
		case <-time.After(time.Minute * 15):
			download()

		case current := <-stdC:
			mutex.Lock()
			addtorrent(CurrentTorrents[current.Title], current)
			mutex.Unlock()
		}
	}
}

func stdinLoop(C chan *SceneVideoTorrent) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		torrentName := filepath.Base(url)
		torrentPath := filepath.Join(unselectedDir, torrentName)
		_, err := os.Stat(torrentPath)
		if !os.IsNotExist(err) {
			continue
		}
		cmd := exec.Command("wget", url, "-q", "-O", torrentPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err = cmd.Run()

		if err != nil {
			fmt.Println("url failed: ", url)
			fmt.Println(err)
			continue
		}
		current := process(torrentPath)
		for _, title := range args.Series {
			if current.Title == title {
				C <- current
				break
			}
		}
	}
}

func download() {
	mutex.Lock()
	defer mutex.Unlock()

	hash := removeLinks()
	removeDownloads(hash)

	for _, s := range CurrentTorrents {
		for _, se := range s {
			for _, ep := range se {
				fmt.Println("symlink", ep.Ep[0].Title, ep.Ep[0].Season, ep.Ep[0].Episode)
				os.Symlink(ep.Ep[0].Meta.FilePath, filepath.Join(filepath.Join(ep.Ep[0].Meta.FilePath, "../../"), filepath.Base(ep.Ep[0].Meta.FilePath)))
			}
		}
	}

}

func addtorrent(St SeriesTorrent, torrent *SceneVideoTorrent) {
	_, ok := St[torrent.Season]
	if !ok {
		St[torrent.Season] = make(SeasonTorrent, 20)
	}
	Ep := St[torrent.Season][torrent.Episode]
	if Ep == nil {
		RES, _ := strconv.Atoi(args.RES)
		St[torrent.Season][torrent.Episode] = &EpisodeTorrent{
			Tags:    make(map[string]int),
			Release: make(map[string]int),
			Res:     scene.Res(RES),
		}

		for i, v := range args.RELEASE {
			if i+1 == 0 {
				panic("You do not exist in a world that I know of")
			}
			St[torrent.Season][torrent.Episode].Release[v] = i + 1
		}
		for i, v := range args.NRELEASE {
			if i+1 == 0 {
				panic("You do not exist in a world that I know of")
			}
			St[torrent.Season][torrent.Episode].Release[v] = i + 1000000
		}
		for i, v := range args.TAGS {
			if i+1 == 0 {
				panic("You do not exist in a world that I know of")
			}
			St[torrent.Season][torrent.Episode].Tags[v] = i + 1
		}
	}
	St[torrent.Season][torrent.Episode].Add(torrent)
}

// Get hashes of torrents that were previously selected then remove the link to them
func removeLinks() (hash []string) {
	selectedDir := filepath.Join(unselectedDir, "../")
	selectedFolder, _ := os.Open(selectedDir)
	//fmt.Println("selected dir", selectedDir)
	defer selectedFolder.Close()

	selectedNames, _ := selectedFolder.Readdirnames(0)
	for _, lnk := range selectedNames {
		target, err := os.Readlink(filepath.Join(selectedDir, lnk))
		//fmt.Println(target)
		//fmt.Println(err)

		if err == nil {
			if filepath.Base(filepath.Dir(target)) == "unselected" {
				hash = append(hash, process(target).Meta.Hash)

				os.Remove(filepath.Join(selectedDir, lnk))
			}
		}
	}
	selectedNames, _ = selectedFolder.Readdirnames(0)
	fmt.Println(selectedNames)
	return
}

func removeDownloads(hash []string) {
	tmap, err := Transmission.GetTorrentMap()
	if err != nil {
		panic(err)
	}
	thash := make([]string, len(hash))
	// Removes torrents from transmission that are not selected this time
	for _, CHash := range hash {
		v, ok := tmap[CHash]
		if ok {
			current := scene.Parse(v.Name)
			if CurrentTorrents[current.Title][current.Season][current.Episode].Ep[0].Meta.Hash != CHash {
				thash = append(thash, v)
			}
		}
	}
	Transmission.RemoveTorrents(false, thash...)
}

func initialize() {
	var (
		err error
	)

	args.PATH, _ = os.Getwd()
	args.RES = "-1"
	arg.MustParse(&args)

	CurrentTorrents = make(map[string]SeriesTorrent, len(args.Series))
	for _, title := range args.Series {
		fmt.Println(title)
		CurrentTorrents[title] = make(SeriesTorrent, 10)
	}

	unselectedDir, _ = filepath.Abs(filepath.Join(args.PATH, "unselected/"))

	//fmt.Println("unselected dir:", unselectedDir)
	// Load all downloaded torrents
	unselectedFolder, _ := os.Open(unselectedDir)
	defer unselectedFolder.Close()
	unselectedNames, _ := unselectedFolder.Readdirnames(0)
	sort.Strings(unselectedNames)
	for _, name := range unselectedNames {
		current := process(filepath.Join(unselectedDir, name))
		for _, title := range args.Series {
			if current.Title == title {
				addtorrent(CurrentTorrents[title], current)
				break
			}
		}
	}

	Transmission, err = transmission.New(transmission.Config{
		Address: "http://timmy:9091/transmission/rpc",
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	download()
}

func process(torrentFile string) *SceneVideoTorrent {
	var (
		mt = new(MetaTorrent)
		vt = new(SceneVideoTorrent)
	)

	f, _ := os.OpenFile(torrentFile, os.O_RDONLY, 755)
	defer f.Close()
	mt.ReadFile(f)
	vt.Torrent = NewTorrent(*mt)
	vt.Parse(strings.TrimSuffix(vt.Name, filepath.Ext(vt.Name)))
	//fmt.Println(vt.Original)
	fmt.Println(vt.Title)
	return vt
}
