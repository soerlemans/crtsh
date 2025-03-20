package main

import (
	//	"encoding/json"
	"bufio"
	"errors"
	"fmt"
	"github.com/alexflint/go-arg"
	"log"
	"os"
	// "net/http"
)

// Structs:
// Arguments Struct:
type Arguments struct {
	Query       string `arg:"-q,--query" help:"Domain query to get subdomains of."`
	InputFile  string `arg:"-f,--file" help:"Path to file containing queries."`
	Recurse     bool   `arg:"-r,--recurse" help:"Recursively search for subdomains." default:"false"`
	Wildcard    bool   `arg:"-w,--wildcard" help:"Include wildcard in output." default:"false"`
	Json bool   `arg:"-j,--json" help:"Include wildcard in output." default:"false"`
	OutputFile string `arg:"-o,--output" help:"Write to output file instead of terminal." default:"crtsh.txt"`
}

// Methods:
func (Arguments) Version() string {
	return "Version: 1.0"
}

func (this Arguments) getQueries() ([]string, error) {
	var err error = nil
	var queries []string = make([]string, 0, QUERIES_PREALLOC)

	if len(args.Query) > 0 {
		queries, err = appendQuery(queries, this.Query)
		if err != nil {
			goto ret
		}
	}

	if len(this.InputFile) > 0 {
		file, err := os.Open(this.InputFile)
		if err != nil {
			goto ret
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			queries, err = appendQuery(queries, scanner.Text())
			if err != nil {
				goto ret
			}
		}

	}

ret:
	return queries, err
}

// Globals:
const CRTSH_BASE_URL = "https://crt.sh/?q=%s&output=json"
const VERSION = "1.0"
const QUERIES_PREALLOC = 100

var args Arguments

// Functions:
func Fail(t_err error) {
	log.Fatal(t_err)

	os.Exit(1)
}

func Print() {

}

func appendQuery(t_queries []string, t_query string) ([]string, error) {
	// TODO: Perform query validation.

	return append(t_queries, t_query), nil
}

func initArgs() ([]string, error) {
	var err error = nil
	var queries []string

	// Parse and handle arguments.
	arg.MustParse(&args)

	queries, err = args.getQueries()
	if err != nil {
		goto ret
	}

	if len(queries) == 0 {
		err = errors.New("No queries provided, use either -q or -f!")
		goto ret
	}

ret:
	return queries, err
}

func fetch(t_queries []string) {
	for _, query := range t_queries {
		fmt.Println("Hypothetical query: ", query)
	}
}

func main() {
	// First parse and set Arguments struct.
	queries, err := initArgs()
	if err != nil {
		Fail(err)
	}

	// Perform subdomain fetching.
	fetch(queries)
}
