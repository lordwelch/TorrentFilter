package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	tf "timmy.narnian.us/git/timmy/TorrentFilter"
	"timmy.narnian.us/git/timmy/scene"

	"github.com/alexflint/go-arg"
	"github.com/lordwelch/transmission"
)

var (
	CurrentTorrents map[string]*tf.SeriesTorrent
	unselectedDir   string
	Transmission    *transmission.Client
	CurrentHashes   []string

	args struct {
		RES      string   `arg:"help:Resolution preference [480/720/1080]"`
		RELEASE  []string `arg:"-r,help:Release group preference order."`
		NRELEASE []string `arg:"-R,help:Release groups to use only as a lost resort."`
		TAGS     []string `arg:"-t,help:Tags to prefer -t internal would choose an internal over another. Whichever file with the most tags is chosen. Release Group takes priority"`
		Series   []string `arg:"required,positional,help:TV series to download torrent file for"`
		NEW      bool     `arg:"-n,help:Only modify new torrents"`
		PATH     string   `arg:"-P,required,help:Path to torrent files"`
		HOST     string   `arg:"-H,help:Host for transmission"`
	}
)

func main() {
	var (
		stdC = make(chan tf.SceneVideoTorrent)
	)
	initialize()
	go stdinLoop(stdC)

	for i := 0; true; i++ {
		fmt.Println(i)
		select {
		case TIME := <-time.After(time.Second):
			fmt.Println(TIME)
			download()

		case current := <-stdC:
			CurrentTorrents[current.Title].Addtorrent(current)
		}
	}
}

func stdinLoop(C chan tf.SceneVideoTorrent) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		fmt.Println("url:", url)
		torrentName := filepath.Base(url)
		torrentPath := filepath.Join(unselectedDir, torrentName)
		_, err := os.Stat(torrentPath)
		if !os.IsNotExist(err) {
			fmt.Println(err)
			continue
		}
		cmd := exec.Command("wget", url, "-t", "5", "--waitretry=5", "-q", "-O", torrentPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err = cmd.Run()

		if err != nil {
			fmt.Println("url failed: ", url)
			fmt.Println(err)
			continue
		}
		if filepath.Ext(torrentPath) == ".torrent" {
			current := process(torrentPath)
			for _, title := range args.Series {
				if current.Title == title {
					C <- current
					break
				}
			}
		}
		fmt.Println("Torrent:", torrentName)
	}
}

// Get hashes of torrents that were previously selected then remove the link to them
func getLinks() (hash []string) {
	// get files in selected dir
	selectedFolder, _ := os.Open(args.PATH)
	defer selectedFolder.Close()
	selectedNames, _ := selectedFolder.Readdirnames(0)
	sort.Strings(selectedNames)
	// Add hashes of currently selected torrents
	for _, lnk := range selectedNames {
		target, err := os.Readlink(filepath.Join(args.PATH, lnk))

		if err == nil {
			if filepath.Dir(target) == unselectedDir {
				used := false
				fakeTorrent := process(target)
				for _, v := range args.Series {
					if fakeTorrent.Title == v && len((*CurrentTorrents[fakeTorrent.Title])[fakeTorrent.Season][fakeTorrent.Episode]) > 0 {
						used = true
						break
					}
				}

				if used {
					realTorrent := (*CurrentTorrents[fakeTorrent.Title])[fakeTorrent.Season][fakeTorrent.Episode][0]
					if realTorrent.Meta.Hash != fakeTorrent.Meta.Hash {
						fmt.Printf("Better file found for: %s S%sE%s\n", realTorrent.Title, realTorrent.Season, realTorrent.Episode)
						err = os.Remove(filepath.Join(args.PATH, filepath.Base(fakeTorrent.Meta.FilePath)))
						os.Symlink(realTorrent.Meta.FilePath, filepath.Join(args.PATH, filepath.Base(realTorrent.Meta.FilePath)))
						hash = append(hash, fakeTorrent.Meta.Hash)
					}
				}
			}
		}
	}
	return
}

func download() {
	var (
		run bool = true
		err error
	)

	hash := getLinks()
	CurrentHashes = make([]string, 0, len(CurrentHashes))
	for _, s := range CurrentTorrents {
		for _, se := range *s {
			for _, ep := range se {
				_, err = os.Open(filepath.Join(args.PATH, filepath.Base(ep[0].Meta.FilePath)))
				if os.IsNotExist(err) {
					os.Remove(filepath.Join(args.PATH, filepath.Base(ep[0].Meta.FilePath)))
					err = os.Symlink(ep[0].Meta.FilePath, filepath.Join(args.PATH, filepath.Base(ep[0].Meta.FilePath)))
					if err != nil {
						fmt.Println(err)
					}
					fmt.Printf("File found for: %s S%sE%s\n", ep[0].Title, ep[0].Season, ep[0].Episode)
				}
				CurrentHashes = append(CurrentHashes, ep[0].Meta.Hash)
			}
		}
	}
	if Transmission != nil {
		stopDownloads(hash)
		time.Sleep(time.Second * 30)
		tmap, _ := Transmission.GetTorrentMap()
		if err != nil {
			run = false
			if timeoutErr, ok := err.(net.Error); ok && timeoutErr.Timeout() {
				tmap, err = Transmission.GetTorrentMap()
				if err != nil {
					run = true
				}
			} else {
				Transmission = nil
			}
		}
		if run {
			for _, s := range CurrentTorrents {
				for _, se := range *s {
					for _, ep := range se {
						v, ok := tmap[ep[0].Meta.Hash]
						if ok {
							v.Set(transmission.SetTorrentArg{
								SeedRatioMode:  1,
								SeedRatioLimit: 1.0,
							})
						}
					}
				}
			}
		}
	}
}

func stopDownloads(hash []string) {
	var (
		run  bool = true
		tmap transmission.TorrentMap
		err  error
	)
	if Transmission != nil {
		tmap, err = Transmission.GetTorrentMap()
		if err != nil {
			run = false
			if timeoutErr, ok := err.(net.Error); ok && timeoutErr.Timeout() {
				tmap, err = Transmission.GetTorrentMap()
				if err != nil {
					run = true
				}
			} else {
				Transmission = nil
			}
		}
		if run {
			// Stops torrents from transmission that are not selected this time
			for _, CHash := range hash {
				v, ok := tmap[CHash]
				if ok {
					v.Stop()
				}
			}

			for _, CHash := range CurrentHashes {
				v, ok := tmap[CHash]
				if ok {
					if v.UploadRatio < 1 {
						v.Start()
					}
				}
			}
		}
	}
}

func initialize() {
	var (
		err error
	)

	args.PATH, _ = os.Getwd()
	args.RES = "-1"
	args.HOST = "localhost"
	arg.MustParse(&args)

	RES, _ := strconv.Atoi(args.RES)
	tf.Res = scene.Res(RES)
	for i, v := range args.RELEASE {
		if i+1 == 0 {
			panic("You do not exist in a world that I know of")
		}
		tf.Release[v] = i + 1
	}
	for i, v := range args.NRELEASE {
		if i+1 == 0 {
			panic("You do not exist in a world that I know of")
		}
		tf.Release[v] = (i + 10) * -1
	}
	for i, v := range args.TAGS {
		if i+1 == 0 {
			panic("You do not exist in a world that I know of")
		}
		tf.Tags[v] = i + 1
	}
	tf.Tags["nuked"] = -99999

	CurrentTorrents = make(map[string]*tf.SeriesTorrent, len(args.Series))
	for _, title := range args.Series {
		fmt.Println(title)
		CurrentTorrents[title] = new(tf.SeriesTorrent)
		*CurrentTorrents[title] = make(tf.SeriesTorrent, 10)
	}

	args.PATH, _ = filepath.Abs(args.PATH)
	unselectedDir = filepath.Join(args.PATH, "unselected/")

	// Load all downloaded torrents
	if !args.NEW {
		unselectedFolder, _ := os.Open(unselectedDir)
		defer unselectedFolder.Close()
		unselectedNames, _ := unselectedFolder.Readdirnames(0)
		sort.Strings(unselectedNames)
		for _, name := range unselectedNames {
			if filepath.Ext(name) == ".torrent" {
				current := process(filepath.Join(unselectedDir, name))
				for _, title := range args.Series {
					if current.Title == title {
						CurrentTorrents[title].Addtorrent(current)
						break
					}
				}
			}
		}
	}

	username := "lordwelch"
	passwd := "hello"
	cl := &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).Dial,
		},
		Timeout: time.Second * 30,
	}
	req, err := http.NewRequest("GET", "http://"+args.HOST+":9091/transmission/rpc", nil)
	req.SetBasicAuth(username, passwd)
	resp, err := cl.Do(req)
	if err != nil {
		fmt.Println(err)
	} else {
		resp.Body.Close()
		Transmission, _ = transmission.New(transmission.Config{
			User:       username,
			Password:   passwd,
			Address:    "http://" + args.HOST + ":9091/transmission/rpc",
			HTTPClient: cl,
		})
	}

	for _, s := range CurrentTorrents {
		for _, se := range *s {
			for _, ep := range se {
				ep.Sort()
			}
		}
	}
	download()
}

func process(torrentFile string) tf.SceneVideoTorrent {
	var (
		mt = new(tf.MetaTorrent)
		vt = new(tf.SceneVideoTorrent)
	)

	f, _ := os.OpenFile(torrentFile, os.O_RDONLY, 755)
	defer f.Close()
	mt.ReadFile(f)
	vt.Torrent = tf.NewTorrent(*mt)
	vt.Parse(strings.TrimSuffix(vt.Name, filepath.Ext(vt.Name)))
	//fmt.Println(vt.Original)
	//fmt.Println(vt)
	return *vt
}
