package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

type result struct {
	path     string
	snippets string
	err      error
}

// Regexp pattern requires formatting with user input
var target_pattern = ".*%v\\(.*\\)|.*%v\\.delay|.*%v\\.apply_async"

// Compiled Regexps
var python_source_re = regexp.MustCompile(`[a-z]*\.py$`)
var decorator_re = regexp.MustCompile(`^@`)
var function_def_re = regexp.MustCompile(`^\s*def\s.+:$`)

// Declaring command-line flags
var exclude_patterns string

func init() {
	flag.StringVar(&exclude_patterns, "exclude_patterns", "", "A comma-separated list of files to be exlcuded")
}

func walkFiles(done <-chan struct{}, root string, exclude_patterns string) (<-chan string, <-chan error) {
	paths := make(chan string)
	errc := make(chan error, 1)

	go func() {
		defer close(paths)
		errc <- filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Checking to see if the file ends in .py
			is_python_source := python_source_re.MatchString(path)

			if err != nil {
				return err
			}

			if !is_python_source {
				return nil
			}

			if len(exclude_patterns) > 0 {
				for _, pattern := range strings.Split(exclude_patterns, ",") {
					if strings.Contains(path, pattern) {
						return nil
					}
				}
			}

			select {
			case paths <- path:
			case <-done:
				return errors.New("Walk canceled")
			}

			return nil
		})
	}()

	return paths, errc
}

func searcher(done <-chan struct{}, target_re *regexp.Regexp, paths <-chan string, results chan<- result) {
	for path := range paths {
		f, err := os.Open(path)

		var snippets_buffer bytes.Buffer
		var buffer bytes.Buffer

		scanner := bufio.NewScanner(f)
		is_target_stub := false
		for scanner.Scan() {
			line := scanner.Text()
			is_decorated := decorator_re.MatchString(line)
			is_def := function_def_re.MatchString(line)
			if is_def || is_decorated {
				if is_target_stub {
					is_target_stub = false
					snippets_buffer.WriteString(buffer.String())
				}

				buffer.Reset()
			}
			is_target_call := target_re.MatchString(line)
			// When true, we'll add this stub to snippets when we hit the next def
			if is_target_call {
				is_target_stub = true
			}
			buffer.WriteString(line)
			buffer.WriteString("\n")
		}
		if is_target_stub {
			snippets_buffer.WriteString(buffer.String())
			buffer.Reset()
		}

		snippets := snippets_buffer.String()

		if len(snippets) == 0 {
			continue
		}

		result := result{
			path,
			snippets,
			err,
		}

		select {
		case results <- result:
		case <-done:
			return
		}
	}
}

func main() {
	runtime.GOMAXPROCS(2)
	flag.Parse()
	args := flag.Args()

	// TODO: This arg is required, validate that shit
	var target string
	if len(args) > 0 {
		target = args[0]
	}

	formatted_target_pattern := fmt.Sprintf(target_pattern, target)
	target_re := regexp.MustCompile(formatted_target_pattern)

	pwd, _ := os.Getwd()

	// Set up channel to alert searchers we're done
	done := make(chan struct{})
	defer close(done)

	paths, errc := walkFiles(done, pwd, exclude_patterns)

	results := make(chan result)

	var wait_group sync.WaitGroup
	// It's possible that spreading out this work across goroutines inherently isn't performant, but it's also possible that I'm doing this wrong
	// So 1 for now!
	const numSearchers = 8
	wait_group.Add(numSearchers)

	for i := 0; i < numSearchers; i++ {
		go func() {
			searcher(done, target_re, paths, results)
			wait_group.Done()
		}()
	}

	go func() {
		wait_group.Wait()
		close(results)
	}()

	for result := range results {
		if result.err != nil {
			fmt.Println(result.err)
			return
		}
		fmt.Println(result.path)
		fmt.Println(result.snippets)
	}

	if err := <-errc; err != nil {
		fmt.Println(err)
		return
	}

}
