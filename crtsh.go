package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alexflint/go-arg"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// Structs:
// Arguments Struct:
type Arguments struct {
	Query      string `arg:"-q,--query" help:"Domain query to get subdomains of."`
	InputFile  string `arg:"-f,--file" help:"Path to file containing queries."`
	Recurse    bool   `arg:"-r,--recurse" help:"Recursively search for subdomains." default:"false"`
	Wildcard   bool   `arg:"-w,--wildcard" help:"Include wildcard in output." default:"false"`
	Json       bool   `arg:"-j,--json" help:"Include wildcard in output." default:"false"`
	OutputFile string `arg:"-o,--output" help:"Write to output file instead of terminal." default:""`
}

// Methods:
func (Arguments) Version() string {
	return fmt.Sprintf("Version: %s", VERSION)
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
		var file *os.File

		file, err = os.Open(this.InputFile)
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
const (
	CRTSH_BASE_URL   = "https://crt.sh/?q=%s&output=json"
	VERSION          = "1.0"
	QUERIES_PREALLOC = 100
	NEWLINE          = "\n"
	WILDCARD         = "*"
	WILDCARD_ENCODE  = "%25"
)

var args Arguments

// Contains subdomains containing a wildcard (*).
var wildcardSubdomains []string

// Functions:
func Fail(t_err error) {
	log.Fatal(t_err)

	os.Exit(1)
}

func Log(t_err error) {
	log.Println(t_err)
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

func extractDomains(t_json []map[string]interface{}) []string {
	var domains []string

	// Loop through and extract domains from JSON.
	for _, elem := range t_json {
		for key, value := range elem {
			if key == "name_value" {
				elems := strings.Split(value.(string), "\n")

				fmt.Printf("%s: %s\n", key, elems)
				for _, elem := range elems {
					if strings.Contains(elem, WILDCARD) {
						wildcardSubdomains = append(wildcardSubdomains, elem)
					} else {
						domains = append(domains, elem)
					}
				}

			}
		}
	}

	return domains
}

func fetch(t_query string) []string {
	var domains []string

	encodeWildcard := args.Wildcard || args.Recurse

	// Encode wildcard if necessary.
	if encodeWildcard {
		t_query = strings.ReplaceAll(t_query, WILDCARD, WILDCARD_ENCODE)
	}

	// Make the request.
	url := fmt.Sprintf(CRTSH_BASE_URL, t_query)
	resp, err := http.Get(url)
	if err != nil {
		Log(err)
		return domains
	}

	// Deal with JSON in the body.
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	var jsonData []map[string]interface{}
	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		Log(err)
		return domains
	}

	// Loop through domains
	domains = extractDomains(jsonData)

	return domains
}

func crtsh(t_queries []string) error {
	var err error = nil

	outputFile := args.OutputFile
	writeToFile := (len(outputFile) > 0)

	var fileHandle *os.File
	if writeToFile {
		fileHandle, err = os.Create(outputFile)
		if err != nil {
			return err
		}
	}

	for _, query := range t_queries {
		domains := fetch(query)

		for _, domain := range domains {
			// Write results of fetching.
			if writeToFile {
				fileHandle.WriteString(domain)
			} else {
				// Write results to intended endpoint.
				fmt.Println(domain)
			}
		}
	}

	return err
}

func printWildcards() {
	if args.Wildcard {
		for _, elem := range wildcardSubdomains {
			// TODO: Fix.
		}
	}
}

func main() {
	// First parse and set Arguments struct.
	queries, err := initArgs()
	if err != nil {
		Fail(err)
	}

	// Perform subdomain fetching.
	err = crtsh(queries)
	if err != nil {
		Fail(err)
	}

	printWildcards()
}
