package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.alm.europe.cloudcenter.corp/ccc-paas/kustomize-plugins/lib/controller"
	"gopkg.in/yaml.v2"
)

type configuration struct {
	GitFileConf string `yaml:"gitFileConf,omitempty"`
}

var (
	conf configuration
)

func main() {
	me := filepath.Base(os.Args[0])

	if len(os.Args) != 2 {
		// Notes: The file yaml configuration is provided by kustomize
		msg := fmt.Sprintf("Usage: %s file-conf.yaml", me)

		fmt.Println(msg)
		controller.LogError(me, msg)

		os.Exit(1)
	}

	// File YAML conf provided for kustomize
	fileConf := os.Args[1]

	dat, err := ioutil.ReadFile(fileConf)

	if err != nil {
		msg := fmt.Sprintf("[ERROR] Open configuration file error %s: %v", fileConf, err)

		fmt.Println(msg)
		controller.LogError(me, msg)

		os.Exit(1)
	}

	err = yaml.Unmarshal(dat, &conf)

	if err != nil {
		msg := fmt.Sprintf("[ERROR] Unmarshal configuration file error: %v", err)

		fmt.Println(msg)
		controller.LogError(me, msg)

		os.Exit(1)
	}

	if len(conf.GitFileConf) == 0 {
		conf.GitFileConf = controller.NameGitFileConf
	}

	// Scanner for stdin
	scanner := bufio.NewScanner(os.Stdin)

	// Replace data from stdin
	output, err := controller.ProcessAllLines(scanner, conf.GitFileConf)

	if err != nil {
		msg := fmt.Sprintf("[ERROR] %v", err)

		fmt.Println(msg)
		controller.LogError(me, msg)

		os.Exit(1)
	}

	// Print result to stdout
	fmt.Print(output)

	controller.DeleteErrorLog(me)

	os.Exit(0)
}
