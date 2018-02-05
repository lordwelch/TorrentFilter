package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/alexflint/go-arg"
	tf "timmy.narnian.us/git/timmy/TorrentFilter"
	"timmy.narnian.us/git/timmy/scene"
)

var (
	args struct {
		RES      string   `arg:"help:Resolution preference [480/720/1080]"`
		RELEASE  []string `arg:"-r,help:Release group preference order."`
		NRELEASE []string `arg:"-R,help:Release groups to use only as a lost resort."`
		TAGS     []string `arg:"-t,help:Tags to prefer -t internal would choose an internal over another. Whichever file with the most tags is chosen. Release Group takes priority"`
	}
)

func main() {

	args.RES = "-1"
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
	scanner := bufio.NewScanner(os.Stdin)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		torrent := process(url)
		fmt.Fprintf(w, "title: %s\t Score: %d\t\n", torrent.Name, torrent.Score())

	}
	w.Flush()
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
