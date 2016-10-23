package main

import (
	//"flag"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

type result struct {
	path     string
	snippets string
	err      error
}

func walkFiles(done <-chan struct{}, root string) (<-chan string, <-chan error) {
	paths := make(chan string)
	errc := make(chan error, 1)

	go func() {
		defer close(paths)
		errc <- filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			is_python_source, err := regexp.MatchString(`[a-z]*\.py$`, path)

			if !is_python_source {
				return nil
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

func searcher(id int, done <-chan struct{}, target string, paths <-chan string, c chan<- result) {
	for path := range paths {
		f, err := os.Open(path)

		var snippets_buffer bytes.Buffer
		var buffer bytes.Buffer

		scanner := bufio.NewScanner(f)
		is_target_stub := false
		for scanner.Scan() {
			line := scanner.Text()
			// TODO: need to compare line with regexp for decorator
			if strings.Contains(line, "def") && is_target_stub {
				is_target_stub = false
				fmt.Printf("I'm searcher %d and I found a target in %s \n", id, path)
				buffer.WriteString("\n")
				snippets_buffer.WriteString(buffer.String())
				buffer.Reset()
			}
			// When true, we'll add this stub to snippets when we hit the next def
			if strings.Contains(line, target) {
				is_target_stub = true
			}
			buffer.WriteString(line)
			buffer.WriteString("\n")
		}
		if is_target_stub {
			fmt.Printf("I'm searcher %d and I found a target in %s \n", id, path)
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
		case c <- result:
		case <-done:
			return
		}
	}
}

func main() {
	args := os.Args[1:]

	var target string
	if len(args) > 0 {
		target = args[0]
	}

	pwd, _ := os.Getwd()

	done := make(chan struct{})
	defer close(done)

	paths, errc := walkFiles(done, pwd)

	c := make(chan result)

	var wg sync.WaitGroup
	const numSearchers = 10
	wg.Add(numSearchers)

	for i := 0; i < numSearchers; i++ {
		go func() {
			searcher(i, done, target, paths, c)
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(c)
	}()

	for result := range c {
		fmt.Println(result.path)
		fmt.Println(result.snippets)
	}

	if err := <-errc; err != nil {
		fmt.Println(err)
		return
	}

}
