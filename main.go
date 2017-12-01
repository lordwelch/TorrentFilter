package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alexflint/go-arg"
)

var (
	current_torrents SeriesTorrent
	unselectedDir    string
)

func main() {
	var (
		torrentName string
		torrentPath string
		args        struct {
			RES     string   `arg:"help:Resolution preference [480/720/1080]"`
			RELEASE []string `arg:"-r,help:Release group preference order."`
			TAGS    []string `arg:"-t, help:Tags to prefer -t internal would choose an internal over another"`
			Series  []string `arg:"required,positional,help:TV series to download"`
			NEW     bool     `arg:"-n,help:Only modify new torrents"`
			PATH    string   `arg:"-P,help:Path to torrent files"`
		}
	)

	arg.MustParse(&args)
	if len(args.PATH) < 1 {
		args.PATH, _ = os.Getwd()
	}
	unselectedDir = filepath.Clean(args.PATH + "/unselected/")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		torrentName = filepath.Base(url)
		torrentPath = filepath.Join(unselectedDir, torrentName)
		cmd := exec.Command("wget", url, "-o", torrentPath)
		if cmd.Run() != nil {
			fmt.Println("url failed: ", url)
			continue
		}
		process(torrentPath)
	}
}

func process(torrentFile string) *SceneVideoTorrent {
	var (
		mt *MetaTorrent       = new(MetaTorrent)
		vt *SceneVideoTorrent = new(SceneVideoTorrent)
	)
	f, _ := os.OpenFile(torrentFile, os.O_RDONLY, 755)
	mt.ReadFile(f)
	//fmt.Printf("%+v\n", mt)
	vt.Torrent = NewTorrent(*mt)
	vt.Parse(vt.Name)
	fmt.Printf("%v\n", *vt)
	return vt
}
