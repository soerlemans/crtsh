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

// Arguments Struct:
type Arguments struct {
	Query      string `arg:"-q,--query" help:"Domain query to get subdomains of."`
	InputFile  string `arg:"-f,--file" help:"Path to file containing queries."`
	Recurse    bool   `arg:"-r,--recurse" help:"Recursively search for subdomains." default:"false"`
	Wildcard   bool   `arg:"-w,--wildcard" help:"Include wildcard in output." default:"false"`
	Json       bool   `arg:"-j,--json" help:"Include wildcard in output." default:"false"`
	OutputFile string `arg:"-o,--output" help:"Write to output file instead of terminal." default:""`
}

// Arguments Methods:
func (Arguments) Version() string {
	return fmt.Sprintf("Version: %s", VERSION)
}

func (this Arguments) getQueries() ([]string, error) {
	var err error = nil
	var queries []string = make([]string, 0, DOMAINS_PREALLOC_DEFAULT_SIZE)

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

// DomainQueue Struct:
type DomainQueue struct {
	Domains chan string
}

// DomainQueue Methods:
func (this DomainQueue) push(t_domain string) {
	this.Domains <- t_domain
}

func (this DomainQueue) pop() string {
	domain := <-this.Domains

	return domain
}

func (this DomainQueue) empty() bool {
	return len(this.Domains) == 0
}

// DomainQueue Factory:
func newDomainQueue(t_size int) DomainQueue {
	return DomainQueue{make(chan string, t_size)}
}

// DefaultOutputWriter Struct:
type DefaultOutputWriter struct {
	OutputFile string
	FileHandle *os.File
}

// DefaultOutputWriter Methods:
func (this DefaultOutputWriter) shouldWriteToFile() bool {
	return len(this.OutputFile) > 0
}

func (this DefaultOutputWriter) write(t_str string) error {
	writeToFile := this.shouldWriteToFile()
	fileHandle := this.FileHandle

	// Write results of fetching.
	if writeToFile {
		fileHandle.WriteString(t_str)
	} else {
		// Write results to intended endpoint.
		fmt.Println(t_str)
	}

	return nil
}

// DefaultOutputWriter Factory:
func newDefaultOutputWriter(t_filepath string) (DefaultOutputWriter, error) {
	DefaultOutputWriter := DefaultOutputWriter{OutputFile: t_filepath, FileHandle: nil}

	if DefaultOutputWriter.shouldWriteToFile() {
		var err error
		DefaultOutputWriter.FileHandle, err = os.Create(DefaultOutputWriter.OutputFile)

		if err != nil {
			return DefaultOutputWriter, err
		}
	}

	return DefaultOutputWriter, nil
}

// Globals:
const (
	CRTSH_BASE_URL                = "https://crt.sh/?q=%s&output=json"
	VERSION                       = "1.0"
	DOMAINS_PREALLOC_DEFAULT_SIZE = 300
	NEWLINE                       = "\n"
	WILDCARD                      = "*"
	WILDCARD_ENCODE               = "%25"
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

func appendQuery(t_queries []string, t_query string) ([]string, error) {
	// TODO: Perform query validation.

	return append(t_queries, t_query), nil
}

func initArgs() (DomainQueue, error) {
	queue := newDomainQueue(DOMAINS_PREALLOC_DEFAULT_SIZE)

	// Parse and handle arguments.
	arg.MustParse(&args)

	queries, err := args.getQueries()
	if err != nil {
		return queue, err
	}

	if len(queries) == 0 {
		err := errors.New("No queries provided, use either -q or -f!")
		return queue, err
	}

	for _, domain := range queries {
		queue.push(domain)
	}

	return queue, err
}

func extractDomains(t_json []map[string]interface{}) []string {
	var domains []string

	// Loop through and extract domains from JSON.
	for _, elem := range t_json {
		for key, value := range elem {
			if key == "name_value" {
				elems := strings.Split(value.(string), "\n")

				domains = append(domains, elems...)
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

// Add the domains we just received containing a WILDCARD, to the DomainQueue.
func appendWildcards(t_queue *DomainQueue, t_domains []string) {
	// Handle recursive enum.
	for _, domain := range t_domains {
		wildcard := strings.Contains(domain, WILDCARD)

		if wildcard {
			t_queue.push(domain)
		}
	}
}

func crtsh(t_queue DomainQueue) error {
	// These should be swappable in the future.
	// Depending on the flags passed (for example JsonOutputWriter).
	writer, err := newDefaultOutputWriter(args.OutputFile)
	if err != nil {
		return err
	}

	for !t_queue.empty() {
		query := t_queue.pop()
		domains := fetch(query)

		// Append wildcards to DomainQueue
		if args.Recurse {
			appendWildcards(&t_queue, domains)
		}

		// Write received results.
		for _, domain := range domains {
			allowWildcard := args.Wildcard
			containsWildcard := strings.Contains(domain, WILDCARD)

			// Skip writing wildcards if this is disabled.
			if !allowWildcard && containsWildcard {
				continue
			}

			writer.write(domain)
		}
	}

	return nil
}

func main() {
	// First parse and set Arguments struct.
	queue, err := initArgs()
	if err != nil {
		Fail(err)
	}

	// Perform subdomain fetching.
	err = crtsh(queue)
	if err != nil {
		Fail(err)
	}
}
