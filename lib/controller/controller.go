package controller

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.alm.europe.cloudcenter.corp/ccc-paas/kustomize-plugins/lib/git"
	"github.alm.europe.cloudcenter.corp/ccc-paas/kustomize-plugins/lib/vault"
	"gopkg.in/ini.v1"
)

const (
	// NameGitFileConf default git file conf
	NameGitFileConf string = "cluster.ini"
)

var (
	currentGitFileConf string

	regexVault          *regexp.Regexp
	regexGit            *regexp.Regexp
	regexCommentLine    *regexp.Regexp
	regexIndentModifier *regexp.Regexp
	regexSelect         *regexp.Regexp
	regexDict           *regexp.Regexp
	regexDefault        *regexp.Regexp

	vaultConf vault.Configuration
	gitConf   *ini.File
)

func init() {
	// Compile generic regexes
	regexVault = regexp.MustCompile(vault.GenericRegex)
	regexGit = regexp.MustCompile(git.GenericRegex)
	regexCommentLine = regexp.MustCompile(`^\s*#.*`)

	// Line modifier indentN
	regexIndentModifier = regexp.MustCompile(`\bindent(\d+)\b`)

	// Data modifier select(regex)
	regexSelect = regexp.MustCompile(`select\s*\(\s*["']?(.+?)["']?\s*\)`)

	// Data modifier dict(key)
	regexDict = regexp.MustCompile(`dict\s*\(\s*["']?(.+?)["']?\s*\)`)

	// Data modifier default(key)
	regexDefault = regexp.MustCompile(`default\s*\(\s*["']?(.+?)["']?\s*\)`)
}

func getVaultEnvironmentVariables() error {
	vaultHost := os.Getenv("VAULT_HOST")
	vaultToken := os.Getenv("VAULT_TOKEN")

	if len(vaultHost) == 0 {
		return fmt.Errorf("[ERROR] VAULT_HOST environment variable not found")
	}

	if len(vaultToken) == 0 {
		return fmt.Errorf("[ERROR] VAULT_TOKEN environment variable not found")
	}

	vaultConf.VaultHost = vaultHost
	vaultConf.VaultToken = vaultToken

	return nil
}

func replaceLineFromVault(line string) (string, error) {
	var output string

	// Find all matches
	res := regexVault.FindAllStringSubmatch(line, -1)

	// Get data from all matches
	for i := range res {
		pathSecret := res[i][1]
		key := res[i][2]
		modifier := res[i][3]

		secret, err := vaultConf.GetSecret(pathSecret, key)

		if err != nil {
			if _, ok := err.(*vault.NotFoundError); !ok {
				return line, err
			}
		}

		secret = processDataModifiers(secret, modifier)

		// Create new regex to replace only the part that corresponds to
		// the variable found
		regexTmp := regexp.MustCompile(fmt.Sprintf(vault.SpecificRegex, pathSecret, key))

		// Replace only the part of the line was found
		line = regexTmp.ReplaceAllString(line, secret)
		line = processLineModifiers(line, modifier)
	}

	output += line

	return output, nil
}

func replaceLineFromGit(line string, gitConf *ini.File) (string, error) {
	var output string

	// Find all matches
	res := regexGit.FindAllStringSubmatch(line, -1)

	// Get data from all matches
	for i := range res {
		key := res[i][1]
		modifier := res[i][2]

		dat := gitConf.Section("").Key(key).String()

		//if len(dat) == 0 {
		//	continue
		//}

		// Check if the value if from Vault
		if isLineFromVault(dat) {
			// Data from Vault, get the value
			dataTmp, err := processLineFromVault(dat)

			if err != nil {
				return line, err
			}

			dat = dataTmp
		}

		dat = processDataModifiers(dat, modifier)

		// Create new regex to replace only the part that corresponds to
		// the variable found (specific regex)
		regexTmp := regexp.MustCompile(fmt.Sprintf(git.SpecificRegex, key))

		line = regexTmp.ReplaceAllString(line, dat)
		line = processLineModifiers(line, modifier)
	}

	output += line

	return output, nil
}

func processDataModifiers(dat string, modifier string) string {
	// Remove all spaces if any
	modifier = strings.ReplaceAll(modifier, " ", "")

	if len(modifier) > 0 {
		// Get all modifiers
		modifiers := strings.Split(modifier, "|")

		for j := range modifiers {
			m := modifiers[j]

			// Remove start and end spaces
			m = strings.TrimSpace(m)

			if len(m) > 0 {
				if m == "base64" {
					dat = encodingBase64(dat)
				} else if regexSelect.MatchString(m) {
					dat = selectData(dat, m)
				} else if regexDict.MatchString(m) {
					dat = selectDictData(dat, m)
				} else if regexDefault.MatchString(m) {
					dat = defaultValue(m)
				}
			}
		}
	}

	return dat
}

func processLineModifiers(line string, modifier string) string {
	// Remove all spaces if any
	//modifier = strings.ReplaceAll(modifier, " ", "")

	if len(modifier) > 0 {
		// Get all modifiers
		modifiers := strings.Split(modifier, "|")

		for j := range modifiers {
			m := modifiers[j]

			// Remove start and end spaces
			m = strings.TrimSpace(m)

			if len(m) > 0 {
				if regexIndentModifier.MatchString(m) {
					// Get the n spaces
					res := regexIndentModifier.FindAllStringSubmatch(m, -1)
					n, _ := strconv.Atoi(res[0][1])
					line = indent(line, n)
				}
			}
		}
	}

	return line
}

func encodingBase64(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

// Indent n spaces from begining
func indent(line string, n int) string {
	var output string

	re := regexp.MustCompile(`^\s+`)

	scanner := bufio.NewScanner(strings.NewReader(line))

	for scanner.Scan() {
		line := scanner.Text()

		line = re.ReplaceAllString(line, "")

		output += fmt.Sprintf("%s%s\n", strings.Repeat(" ", n), line)
	}

	if len(output) > 1 {
		// Delete last carriage return
		output = output[:len(output)-1]
	}

	return output
}

func selectData(dat string, modifier string) string {
	// Get regex
	// Notes: Remember regex is between ()
	// Example: select(^one$)
	res := regexSelect.FindAllStringSubmatch(modifier, -1)
	re := regexp.MustCompile(res[0][1])

	d := strings.Split(dat, ",")

	for i := range d {
		tmp := strings.TrimSpace(d[i])

		if re.MatchString(tmp) {
			return tmp
		}
	}

	return dat
}

func selectDictData(dat string, modifier string) string {
	// Get dict key
	// Notes: Remember key is between ()
	// Example: dict(subneta)
	res := regexDict.FindAllStringSubmatch(modifier, -1)
	keySelected := res[0][1]

	// Notes: The data is key01=value01,key02=value02....
	d := strings.Split(dat, ",")

	for i := range d {
		tmp := strings.TrimSpace(d[i])

		// Get key, value
		keyValue := strings.Split(tmp, "=")

		k := strings.TrimSpace(keyValue[0])
		v := strings.TrimSpace(keyValue[1])

		if k == keySelected {
			return v
		}
	}

	return dat
}

func defaultValue(modifier string) string {
	// Get default value
	// Notes: Remember default value is between ()
	// Example: default(foo)
	res := regexDefault.FindAllStringSubmatch(modifier, -1)
	defaultValueSelected := res[0][1]

	return defaultValueSelected
}

func checkIfFileExists(file string) error {
	info, err := os.Stat(file)

	if os.IsNotExist(err) {
		return err
	}

	if info.IsDir() {
		return fmt.Errorf("File \"%s\" is a directory", file)
	}

	if info.Size() == 0 {
		return fmt.Errorf("File \"%s\" is empty", file)
	}

	return nil
}

// isCommentLine check if line starts with a comment #
func isCommentLine(line string) bool {
	return regexCommentLine.MatchString(line)
}

// IsLineFromVault check if line is a Vault line
func isLineFromVault(line string) bool {
	return regexVault.MatchString(line)
}

// IsLineFromGit check if line is a Git line
func isLineFromGit(line string) bool {
	return regexGit.MatchString(line)
}

// processLineFromVault process a vault line
func processLineFromVault(line string) (string, error) {
	if len(vaultConf.VaultHost) == 0 || len(vaultConf.VaultToken) == 0 {
		// Get Vault environment variables
		if err := getVaultEnvironmentVariables(); err != nil {
			return "", err
		}
	}

	line, err := replaceLineFromVault(line)

	if err != nil {
		return "", err
	}

	return line, nil
}

// ProcessLineFromGit process a git line
func processLineFromGit(line string, gitFileConf string) (string, error) {
	if gitConf == nil {
		// Load git file configuration
		cfg, err := git.LoadFileConf(gitFileConf)

		if err != nil {
			return "", err
		}

		gitConf = cfg
	}

	line, err := replaceLineFromGit(line, gitConf)

	if err != nil {
		return "", err
	}

	return line, nil
}

// ProcessAllLines from bufio.Scanner
func ProcessAllLines(sc *bufio.Scanner, gitFileConf string) (string, error) {
	var output string

	currentGitFileConf = gitFileConf

	for sc.Scan() {
		// GET the line string
		line := sc.Text()

		if isCommentLine(line) {
			output += line
		} else if isLineFromVault(line) {
			lineTmp, err := processLineFromVault(line)

			if err != nil {
				return "", err
			}

			output += lineTmp
		} else if isLineFromGit(line) {
			lineTmp, err := processLineFromGit(line, gitFileConf)

			if err != nil {
				return "", err
			}

			output += lineTmp
		} else {
			output += line
		}

		output += "\n"
	}

	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("Scan file error: %v", err)
	}

	return output, nil
}

// LogError log a error in a file
func LogError(namePlugin, msg string) {
	file, err := os.OpenFile(fmt.Sprintf("/tmp/%s.log", namePlugin), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)

	if err != nil {
		fmt.Println(err)
		return
	}

	defer file.Close()

	log.SetOutput(file)
	log.Println(msg)
}

// DeleteErrorLog delete log file error
func DeleteErrorLog(namePlugin string) {
	filename := fmt.Sprintf("/tmp/%s.log", namePlugin)

	fileinfo, err := os.Stat(filename)

	if !os.IsNotExist(err) {
		now := time.Now()
		modtime := fileinfo.ModTime()

		diff := now.Sub(modtime)

		fmt.Printf("%v\n", diff)

		if h := diff.Hours(); h > 24 {
			// Delete file log if it is more than 24 hours old
			os.Remove(filename)
		}
	}
}
