package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Brian-Nervi/PokedexCLI/internal/pokecache"
)

func main() {
	Scanner := bufio.NewScanner(os.Stdin)
	cfg := &config{
		cache:         pokecache.NewCache(5 * time.Second),
		caughtPokemon: map[string]Pokemon{},
	}
	for {
		fmt.Print("Pokedex > ")
		Scanner.Scan()                           // waits for input
		input := Scanner.Text()                  //captures input
		cfg.history = append(cfg.history, input) //adds the input to the history
		cfg.historyIndex = len(cfg.history)      //resets to max index number
		inputSlice := cleanInput(input)          // separate the input into individuals strings in a slice
		if len(inputSlice) == 0 {
			continue
		}
		if command, exists := getCommands()[inputSlice[0]]; exists {
			err := command.callback(cfg, inputSlice[1:])
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func cleanInput(input string) []string {
	input = strings.ToLower(input)
	result := strings.Fields(input)
	return result
}

type cliCommand struct {
	name        string
	description string
	callback    func(*config, []string) error
}

func getCommands() map[string]cliCommand {
	return map[string]cliCommand{
		"exit": {
			name:        "exit",
			description: "Exit the Pokedex",
			callback:    commandExit,
		},
		"help": {
			name:        "help",
			description: "Displays a help message",
			callback:    commandHelp,
		},
		"map": {
			name:        "map",
			description: "Shows the names of 20 location areas, each subsequent use of the command will show the next 20",
			callback:    commandMap,
		},
		"mapb": {
			name:        "mapb",
			description: "Displays the previous 20 location areas, each subsequent use of the command will show the previous 20",
			callback:    commandMapB,
		},
		"explore": {
			name:        "explore",
			description: "Shows a list of all the pokemon in a location",
			callback:    commandExplore,
		},
		"catch": {
			name:        "catch",
			description: "Attempt to catch the chosen pokemon",
			callback:    commandCatch,
		},
		"inspect": {
			name:        "inspect",
			description: "Get information about the cosen pokemon",
			callback:    commandInspect,
		},
		"pokedex": {
			name:        "pokedex",
			description: "View your caught pokemon",
			callback:    commandPokedex,
		},
	}
}

type config struct {
	nextUrl       *string
	previousUrl   *string
	cache         pokecache.Cache
	caughtPokemon map[string]Pokemon
	history       []string
	historyIndex  int
}

type LocationAreaResponse struct {
	Count    int            `json:"count"`
	Next     *string        `json:"next"`
	Previous *string        `json:"previous"`
	Results  []LocationArea `json:"results"`
}

type LocationArea struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

type Pokemon struct {
	Name           string         `json:"name"`
	Height         int            `json:"height"`
	Weight         int            `json:"weight"`
	Stats          []PokemonStats `json:"stats"`
	Types          []PokemonTypes `json:"types"`
	BaseExperience int            `json:"base_experience"`
}

type PokemonStats struct {
	BaseStat int `json:"base_stat"`
	Stat     struct {
		Name string `json:"name"`
	} `json:"stat"`
}

type PokemonTypes struct {
	Type struct {
		Name string `json:"name"`
	} `json:"type"`
}

func commandExit(cfg *config, args []string) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func commandHelp(cfg *config, args []string) error {
	fmt.Println()
	fmt.Println("Welcome to the Pokedex!")
	fmt.Println("Usage:")
	fmt.Println()
	for _, cmd := range getCommands() {
		fmt.Printf("%s: %s\n", cmd.name, cmd.description)
	}
	fmt.Println()
	return nil
}

func commandMap(cfg *config, args []string) error {
	Url := "https://pokeapi.co/api/v2/location-area"
	var err error
	var res *http.Response
	if cfg.nextUrl != nil {
		Url = *cfg.nextUrl
	}
	body, ok := cfg.cache.Get(Url)
	if !ok {
		res, err = http.Get(Url)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		body, err = io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		cfg.cache.Add(Url, body)
	}

	var data LocationAreaResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}
	for _, area := range data.Results {
		fmt.Println(area.Name)
	}
	cfg.nextUrl = data.Next
	cfg.previousUrl = data.Previous

	return nil
}

func commandMapB(cfg *config, args []string) error {

	if cfg.previousUrl == nil {
		fmt.Println("you're on the first page")
		return nil
	}
	Url := cfg.previousUrl
	var res *http.Response
	var err error
	body, ok := cfg.cache.Get(*Url)
	if !ok {
		res, err = http.Get(*Url)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		body, err = io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		cfg.cache.Add(*Url, body)
	}

	var data LocationAreaResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}
	for _, area := range data.Results {
		fmt.Println(area.Name)
	}
	cfg.nextUrl = data.Next
	cfg.previousUrl = data.Previous

	return nil
}

type ExploreResponse struct {
	Name              string             `json:"name"`
	PokemonEncounters []PokemonEncounter `json:"pokemon_encounters"`
}

type PokemonEncounter struct {
	Pokemon Pokemon `json:"pokemon"`
}

func commandExplore(cfg *config, args []string) error {
	if len(args) == 0 {
		return errors.New("please use an area or name id")
	}
	locationName := args[0]
	url := "https://pokeapi.co/api/v2/location-area/" + locationName

	var res *http.Response
	var err error
	body, ok := cfg.cache.Get(url)
	if !ok {
		res, err = http.Get(url)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		body, err = io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		cfg.cache.Add(url, body)
	}
	var data ExploreResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}
	fmt.Printf("Now Exploring %s\n", locationName)
	for _, encounter := range data.PokemonEncounters {
		fmt.Println(encounter.Pokemon.Name)
	}

	return nil
}

func commandCatch(cfg *config, args []string) error {
	if len(args) == 0 {
		return errors.New("please select a pokemon to catch")
	}
	pokemonName := args[0]
	url := "https://pokeapi.co/api/v2/pokemon/" + pokemonName

	var res *http.Response
	var err error
	body, ok := cfg.cache.Get(url)
	if !ok {
		res, err = http.Get(url)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		body, err = io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		cfg.cache.Add(url, body)
	}
	var data Pokemon
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}
	fmt.Printf("Throwing a Pokeball at %s...\n", pokemonName)
	catchR := rand.IntN(data.BaseExperience)
	if catchR > 40 {
		fmt.Printf("%s escaped!\n", pokemonName)
	} else {
		fmt.Printf("%s was caught!\nYou may now inspect it with the inspect command.\n", pokemonName)
		cfg.caughtPokemon[pokemonName] = data
	}
	return nil
}

func commandInspect(cfg *config, args []string) error {
	pokemon, ok := cfg.caughtPokemon[args[0]]
	if !ok {
		return errors.New("you have not caught that pokemon")
	}
	fmt.Printf("Name: %s\nHeight: %d\nWeight: %d\n", pokemon.Name, pokemon.Height, pokemon.Weight)
	fmt.Println("Stats:")
	for _, stat := range pokemon.Stats {
		fmt.Printf("  -%s:%d\n", stat.Stat.Name, stat.BaseStat)
	}
	fmt.Println("Types:")
	for _, types := range pokemon.Types {
		fmt.Printf("  - %s\n", types.Type.Name)
	}
	return nil
}

func commandPokedex(cfg *config, args []string) error {
	fmt.Println("Your Pokedex:")
	for _, pokemon := range cfg.caughtPokemon {
		fmt.Printf("  - %s\n", pokemon.Name)
	}
	return nil
}
