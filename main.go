package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
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
	pokedex  map[string]pokeapi.Pokemon
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
		"catch": {
			name:        "catch",
			description: "Catch a Pokemon",
			callback:    commandCatch,
		},
		"inspect": {
			name:        "inspect",
			description: "View a Pokemon's details if it has been registered in Godex",
			callback:    commandInspect,
		},
		"godex": {
			name:        "godex",
			description: "View a list of Pokemon registered in your Godex",
			callback:    commandGodex,
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

func getPokemon(c *config, pkmn string) (pokeapi.Pokemon, error) {
	var body []byte
	query := fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%v/", pkmn)
	val, ok := c.cache.Get(query)
	if ok {
		body = val
	} else {
		resp, err := http.Get(query)
		if err != nil {
			return pokeapi.Pokemon{}, err
		}
		defer resp.Body.Close()
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return pokeapi.Pokemon{}, err
		}
		c.cache.Add(query, body)
	}
	//translate response to struct defined in pokeapi
	var pokeData pokeapi.Pokemon
	err := json.Unmarshal(body, &pokeData)

	if err != nil {
		return pokeapi.Pokemon{}, err
	}

	return pokeData, nil
}

func commandGodex(c *config, args ...string) error {
	if len(c.pokedex) == 0 {
		fmt.Println("Your Godex is empty")
		return errors.New("pokedex is empty")
	}
	formattedOutput := "Your Godex:"
	for k := range c.pokedex {
		formattedOutput += fmt.Sprintf("\n  -%v", k)
	}
	fmt.Println(formattedOutput)
	return nil
}

func commandInspect(c *config, args ...string) error {
	if len(args) < 1 {
		fmt.Println("Please provide the name of a Pokemon")
		return errors.New("not enough arguments")
	}
	if len(args) > 1 {
		fmt.Println("Only one Pokemon may be accepted")
		return errors.New("too many arguments")
	}

	pkmnName := args[0]
	pkmnData, ok := c.pokedex[pkmnName]

	if !ok {
		fmt.Printf("%v has not been registered in Godex\n", pkmnName)
		return errors.New("pkmn not registered")
	}

	statData := make([]interface{}, 6)
	for i, f := range pkmnData.Stats {
		statData[i] = f.BaseStat
	}

	formattedBasicData := fmt.Sprintf(
		`Name: %v
Height: %v
Weight: %v`, pkmnData.Name, pkmnData.Height, pkmnData.Weight)

	formattedStatData := fmt.Sprintf(
		`Stats:
  -hp: %v
  -attack: %v
  -defense: %v
  -special-attack: %v
  -special-defense: %v
  -speed: %v`, statData...)

	formattedTypeData := "Types:"
	for _, f := range pkmnData.Types {
		formattedTypeData += fmt.Sprintf("\n  -%v", f.Type.Name)
	}

	fmt.Println(formattedBasicData)
	fmt.Println(formattedStatData)
	fmt.Println(formattedTypeData)

	/*
		formattedData := fmt.Sprintf(`Name: %v
		Height: %v
		Weight: %v
		Stats:
			-hp: %v
			-attack: %v
			-defense: %v
			-special-attack: %v
			-special-defense: %v
			-speed: %v
		Types:
			-%v
			-%v`, pkmnData.Name, pkmnData.Height, pkmnData.Weight, statData...)
	*/
	return nil
}

func commandCatch(c *config, args ...string) error {
	if len(args) < 1 {
		fmt.Println("Please provide a Pokemon to catch")
		return errors.New("not enough arguments")
	}
	if len(args) > 1 {
		fmt.Println("Only one Pokemon may be accepted")
		return errors.New("too many arguments")
	}
	pkmnName := args[0]
	pkmnData, err := getPokemon(c, pkmnName)

	if err != nil {
		fmt.Println("Invalid Pokemon")
		return err
	}

	fmt.Printf("Throwing a Pokeball at %v...\n", pkmnData.Name)

	catchChance := pkmnData.BaseExperience
	catchRoll := rand.Intn(620)
	if catchRoll > catchChance {
		fmt.Printf("%v was caught!\n", pkmnData.Name)
		c.pokedex[pkmnData.Name] = pkmnData
	} else {
		fmt.Printf("%v fled!\n", pkmnData.Name)
	}
	return nil
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

	for _, pokemon := range locData.PokemonEncounters {
		fmt.Println(pokemon.Pokemon.Name)
	}
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
		pokedex:  make(map[string]pokeapi.Pokemon),
	}
	fmt.Println("Starting Godex...")
	scanner := bufio.NewScanner(os.Stdin)
	initializeInput()
	for {
		if scanner.Scan() {
			input := scanner.Text()
			words := strings.Fields(input)
			if len(words) == 0 {
				continue
			}
			command := words[0]
			arguments := words[1:]

			if cmd, ok := getCommands()[command]; ok {
				cmd.callback(&c, arguments...)
			} else {
				fmt.Println("Invalid command. Use `help` for a list of commands.")
			}

			initializeInput()
		}
	}
}
