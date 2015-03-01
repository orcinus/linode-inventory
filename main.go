package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/awilliams/linode"

	"code.google.com/p/gcfg"
)

const configName = "linode-inventory.ini"

var args struct {
	version bool
	ini     string
}

func init() {
	flag.BoolVar(&args.version, "v", false, "Print version")
	flag.StringVar(&args.ini, "f", configName, "Specify config file")
}

var config *configuration
var linodeClient *linode.Client

const usage = "usage: %s [flag]\n"

func main() {
	flag.Parse()
	var err error
	config, err = getConfig(args.ini)

	if err != nil {
		if args.ini != configName {
			fatal(err)
			return
		} else {
			flag.PrintDefaults()
			return
		}
	}

	if args.version {
		fmt.Printf("%s v%s\n", appName, appVersion)
		return
	} else {
		linodeClient = linode.NewClient(config.APIKey)
		inv := newInventory(linodes())
		inventoryJSON, err := inv.toJSON()
		if err != nil {
			fatal(err)
			return
		}
		os.Stdout.Write(inventoryJSON)
		return
	}

	fmt.Fprintf(os.Stderr, usage, appName)
	flag.PrintDefaults()
}

type configuration struct {
	APIKey       string `gcfg:"api-key"`
	DisplayGroup string `gcfg:"display-group"`
}

// returns true if displayGroup should be included in result set
func (c *configuration) filterDisplayGroup(displayGroup string) bool {
	if c.DisplayGroup == "" {
		return true
	}
	return c.DisplayGroup == displayGroup
}

func getConfig(filename string) (*configuration, error) {
	// first check directory where the executable is located
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return nil, err
	}
	path := dir + "/" + filename
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// fallback to working directory. This is usefull when using `go run`
		path = filename
	}

	var config struct {
		Linode configuration
	}

	err = gcfg.ReadFileInto(&config, path)
	if err != nil {
		return nil, err
	}

	return &config.Linode, nil
}

type linodeWithIPs struct {
	node linode.Linode
	ips  []linode.LinodeIP
}

func linodes() map[int]*linodeWithIPs {
	nodes, err := linodeClient.LinodeList()
	if err != nil {
		fatal(err)
	}

	m := make(map[int]*linodeWithIPs, len(nodes))
	ids := make([]int, 0, len(nodes))
	for _, n := range nodes {
		if !config.filterDisplayGroup(n.DisplayGroup) {
			continue
		}
		v := &linodeWithIPs{node: n}
		m[n.ID] = v
		ids = append(ids, n.ID)
	}

	ipMap, err := linodeClient.LinodeIPList(ids)
	if err != nil {
		fatal(err)
	}
	for nodeID, ips := range ipMap {
		m[nodeID].ips = ips
	}

	return m
}

func fatal(v interface{}) {
	fmt.Fprintln(os.Stderr, v)
	os.Exit(1)
}
