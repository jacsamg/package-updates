package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type OutdatedDependency struct {
	Current  string `json:"current"`
	Wanted   string `json:"wanted"`
	Latest   string `json:"latest"`
	Location string `json:"location"`
	Type     string `json:"type"`
}

type OutdatedDependencyWithName struct {
	Name     string `json:"name"`
	Current  string `json:"current"`
	Wanted   string `json:"wanted"`
	Latest   string `json:"latest"`
	Location string `json:"location"`
	Type     string `json:"type"`
}

type OutdatedDependencyWithIndex struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	Current  string `json:"current"`
	Wanted   string `json:"wanted"`
	Latest   string `json:"latest"`
	Location string `json:"location"`
	Type     string `json:"type"`
}

var UPDATES_FILE_NAME = "package-updates.json"

func checkError(err error) {
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
}

func execNpmOutdatedCmd() []byte {
	var outBuffer, errBuffer bytes.Buffer

	cmd := exec.Command("npm", "outdated", "--json", "--long")
	cmd.Stdout = &outBuffer
	cmd.Stderr = &errBuffer
	err := cmd.Run()
	outStr := outBuffer.String()
	errStr := errBuffer.String()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if outStr == "" {
				fmt.Printf("ERROR: npm outdated command failed with exit code %d\n", exitError.ExitCode())
				fmt.Printf("ERROR: npm outdated command output: %s\n", errStr)
				os.Exit(1)
			}
		} else {
			checkError(errors.New(errStr))
		}
	}

	return outBuffer.Bytes()
}

func convertNpmOutdatedOutputToMap(rawOutput []byte) map[string]OutdatedDependency {
	var outdated map[string]OutdatedDependency

	jsonUnmarshalError := json.Unmarshal(rawOutput, &outdated)
	checkError(jsonUnmarshalError)

	return outdated
}

func convertDependencysToArray(dependencyMap map[string]OutdatedDependency) []OutdatedDependencyWithIndex {
	var itemsWithName []OutdatedDependencyWithName
	var itemsWithIndex []OutdatedDependencyWithIndex

	for pkg, details := range dependencyMap {
		itemsWithName = append(itemsWithName, OutdatedDependencyWithName{
			Name:     pkg,
			Current:  details.Current,
			Wanted:   details.Wanted,
			Latest:   details.Latest,
			Location: details.Location,
			Type:     details.Type,
		})
	}

	sort.Slice(itemsWithName, func(i, j int) bool {
		return itemsWithName[i].Name < itemsWithName[j].Name
	})

	for index, details := range itemsWithName {
		itemsWithIndex = append(itemsWithIndex, OutdatedDependencyWithIndex{
			Id:       index,
			Name:     details.Name,
			Current:  details.Current,
			Wanted:   details.Wanted,
			Latest:   details.Latest,
			Location: details.Location,
			Type:     details.Type,
		})
	}

	return itemsWithIndex
}

func displayDependencysToUpdate(dependencyArr []OutdatedDependencyWithIndex) {
	for index, details := range dependencyArr {
		indexStr := strconv.Itoa(index)

		if index < 10 {
			fmt.Printf("[0%s] %s (%s => %s) \n", indexStr, details.Name, details.Current, details.Latest)
		} else {
			fmt.Printf("[%s] %s (%s => %s) \n", indexStr, details.Name, details.Current, details.Latest)
		}
	}

	fmt.Println()
}

func checkProcess() {
	output := execNpmOutdatedCmd()
	dependencyMap := convertNpmOutdatedOutputToMap(output)
	dependencyMapLength := len(dependencyMap)

	if dependencyMapLength < 0 {
		fmt.Printf("No dependencys to update found \n")
		os.Exit(0)
	}

	fmt.Printf("%s dependencys to update found \n", strconv.Itoa(dependencyMapLength))
	fmt.Printf("To update, run the following command: \n")
	fmt.Printf("npm-updates --update [dependency-id] \n")
	fmt.Println()

	dependencyArr := convertDependencysToArray(dependencyMap)

	displayDependencysToUpdate(dependencyArr)

	jsonBytes, err := json.MarshalIndent(dependencyArr, "", "  ")
	checkError(err)

	file, err := os.Create(UPDATES_FILE_NAME)
	checkError(err)

	defer file.Close()

	_, err = file.Write(jsonBytes)
	checkError(err)
}

func formatUpdateString(rawIds string) string {
	return strings.ReplaceAll(rawIds, " ", "")
}

func verifyUpdateStringFormat(formatedIds string) bool {
	re := regexp.MustCompile(`^(\d+,)*\d+$`)
	return re.MatchString(formatedIds)
}

func getJSONData() []OutdatedDependencyWithIndex {
	var dependencys []OutdatedDependencyWithIndex

	data, readFileErr := os.ReadFile(UPDATES_FILE_NAME)
	checkError(readFileErr)

	unmarshalErr := json.Unmarshal(data, &dependencys)
	checkError(unmarshalErr)

	return dependencys
}

func removeJSONData() {
	err := os.Remove(UPDATES_FILE_NAME)
	checkError(err)
}

func getArrayOfDependencyIds(idsToUpdate string) []int {
	idsStr := strings.Split(idsToUpdate, ",")
	var ids []int

	for _, match := range idsStr {
		id, err := strconv.Atoi(match)
		checkError(err)
		ids = append(ids, id)
	}

	return ids
}

func execNpmUninstallCdm(dependency OutdatedDependencyWithIndex) {
	var outBuffer, errBuffer bytes.Buffer

	cmd := exec.Command("npm", "uninstall", dependency.Name, "--force")
	cmd.Stdout = &outBuffer
	cmd.Stderr = &errBuffer
	err := cmd.Run()
	outStr := outBuffer.String()
	errStr := errBuffer.String()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if outStr == "" {
				fmt.Printf("ERROR: npm uninstall command failed with exit code %d\n", exitError.ExitCode())
				fmt.Printf("ERROR: npm uninstall command output: %s\n", errStr)
				os.Exit(1)
			}
		} else {
			checkError(errors.New(errStr))
		}
	}
}

func execNpmInstallCdm(dependency OutdatedDependencyWithIndex) {
	var outBuffer, errBuffer bytes.Buffer
	dependencyType := ""

	if dependency.Type == "devDependencies" {
		dependencyType = "--save-dev"
	} else if dependency.Type == "dependencies" {
		dependencyType = "--save"
	}

	cmd := exec.Command("npm", "install", dependency.Name+"@"+dependency.Latest, dependencyType, "--force")
	cmd.Stdout = &outBuffer
	cmd.Stderr = &errBuffer
	err := cmd.Run()
	outStr := outBuffer.String()
	errStr := errBuffer.String()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if outStr == "" {
				fmt.Printf("ERROR: npm install command failed with exit code %d\n", exitError.ExitCode())
				fmt.Printf("ERROR: npm install command output: %s\n", errStr)
				os.Exit(1)
			}
		} else {
			checkError(errors.New(errStr))
		}
	}
}

func updateProcess(idsToUpdate string) {
	ids := getArrayOfDependencyIds(idsToUpdate)
	dependencys := getJSONData()

	for _, id := range ids {
		dependencyToUpdate := dependencys[id]

		fmt.Printf("Updating '%s' from %s to %s \n", dependencyToUpdate.Name, dependencyToUpdate.Current, dependencyToUpdate.Latest)
		execNpmUninstallCdm(dependencyToUpdate)
		execNpmInstallCdm(dependencyToUpdate)
		fmt.Printf("Dependency '%s' updated successfully \n", dependencyToUpdate.Name)
		fmt.Println()
	}

	removeJSONData()
	fmt.Println("Done!")
}

func main() {
	check := flag.Bool("check", false, "Check for updates")
	update := flag.String("update", "", "Install updates (e.g. --update 1,2,3)")

	flag.Parse()

	if !*check && *update == "" {
		fmt.Println("WARNING: Is necesary specify one of the arguments (--check or --update)")
		os.Exit(0)
	}

	if *check && *update != "" {
		fmt.Println("WARNING: Only one argument can be specified (--check or --update)")
		os.Exit(0)
	}

	if *check {
		fmt.Println("Checking for updates...")
		fmt.Println()
		checkProcess()
		os.Exit(0)
	}

	if *update != "" {
		idsToUpdate := formatUpdateString(*update)

		if !verifyUpdateStringFormat(idsToUpdate) {
			fmt.Println("ERROR: The update string must be a comma separated list of IDs (e.g. 1,2,3)")
			os.Exit(1)
		}

		fmt.Println("Updating...")
		fmt.Println()
		updateProcess(idsToUpdate)
		os.Exit(1)
	}
}
