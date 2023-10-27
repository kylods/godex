package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/kylods/godex/internal/pokeapi"
	"github.com/kylods/godex/internal/pokecache"
)

type cliCommand struct {
	name        string
	description string
	callback    func(*config, ...string) error
}

type config struct {
	locIndex int
	cache    pokecache.Cache
}

func getCommands() map[string]cliCommand {
	return map[string]cliCommand{
		"help": {
			name:        "help",
			description: "Displays a help message",
			callback:    commandHelp,
		},
		"exit": {
			name:        "exit",
			description: "Exit the Godex",
			callback:    commandExit,
		},
		"map": {
			name:        "map",
			description: "Tabs through pages of locations",
			callback:    commandMap,
		},
		"mapb": {
			name:        "mapb",
			description: "Tabs to a previous location page",
			callback:    commandMapB,
		},
		"explore": {
			name:        "explore",
			description: "Explores an area for Pokemon",
			callback:    commandExplore,
		},
	}
}

func getMapPage(c *config) error {
	page := c.locIndex
	//do 20 times
	for i := 0; i < 20; i++ {
		index := ((page - 1) * 20) + i + 1
		locData, err := getMap(c, fmt.Sprint(index))
		if err != nil {
			return err
		}
		fmt.Println(locData.Name)
	}
	return nil
}

func getMap(c *config, id string) (pokeapi.LocationArea, error) {
	var body []byte
	query := fmt.Sprintf("https://pokeapi.co/api/v2/location-area/%v/", id)
	val, ok := c.cache.Get(query)
	if ok {
		body = val
	} else {
		resp, err := http.Get(query)
		if err != nil {
			return pokeapi.LocationArea{}, err
		}
		defer resp.Body.Close()
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return pokeapi.LocationArea{}, err
		}
		c.cache.Add(query, body)
	}
	//translate response to struct defined in pokeapi
	var locData pokeapi.LocationArea
	err := json.Unmarshal(body, &locData)

	if err != nil {
		return pokeapi.LocationArea{}, err
	}

	return locData, nil
}

func commandExplore(c *config, args ...string) error {
	if len(args) < 1 {
		fmt.Println("Please provide a location ID")
		return errors.New("not enough arguments")
	}
	if len(args) > 1 {
		fmt.Println("Only one location may be accepted")
		return errors.New("too many arguments")
	}
	id := args[0]
	locData, err := getMap(c, id)

	if err != nil {
		fmt.Println("Invalid Location ID")
		return err
	}

	fmt.Println(locData.Name)
	return nil
}

func commandMap(c *config, args ...string) error {
	c.locIndex++
	getMapPage(c)

	return nil
}

func commandMapB(c *config, args ...string) error {
	if c.locIndex < 2 {
		fmt.Println("No previous page to display")
		return errors.New("c.locIndex must be 2 or greater")
	}
	c.locIndex--
	getMapPage(c)
	return nil
}

func commandExit(c *config, args ...string) error {
	fmt.Println("Exiting Godex...")
	os.Exit(0)
	return nil
}

func commandHelp(c *config, args ...string) error {
	for _, cmd := range getCommands() {
		fmt.Println(cmd.name, ": ", cmd.description)
	}
	return nil
}

func initializeInput() {
	fmt.Println("")
	fmt.Print("Godex > ")
}

func main() {
	c := config{
		locIndex: 0,
		cache:    *pokecache.NewCache(300 * time.Second),
	}
	fmt.Println("Starting Godex...")
	scanner := bufio.NewScanner(os.Stdin)
	initializeInput()
	for {
		if scanner.Scan() {
			input := scanner.Text()

			if cmd, ok := getCommands()[input]; ok {
				cmd.callback(&c)
			}

			initializeInput()
		}
	}
}
