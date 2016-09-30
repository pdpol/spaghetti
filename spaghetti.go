package main

import (
	//"flag"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

func walkFiles(done <-chan struct{}, root string) (<-chan string, <-chan error) {
	paths := make(chan string)
	errc := make(chan error, 1)

	go func() {
		defer close(paths)
		errc <- filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.Mode().IsDir() {
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

func searcher(done <-chan struct{}, target string, paths <-chan string, c chan<- []string) {
	for path := range paths {
		data := MethodSearch(target, path)

		select {
		case c <- data:
		case <-done:
			return
		}
	}
}

func MethodSearch(methodName string, path string) (results []string) {
	//f, err := os.Open(path)

	/*if err != nil {
		return err
	}*/

	result := []string{methodName, path}
	return result
}

func main() {
	/*args := os.Args[1:]

	if len(args) > 0 {
		fmt.Println("Got some args: ", args)
	} else {
		fmt.Println("No args here!")
	}*/

	pwd, _ := os.Getwd()

	done := make(chan struct{})
	defer close(done)

	paths, errc := walkFiles(done, pwd)
	target := "test"

	c := make(chan []string)

	var wg sync.WaitGroup
	const numSearchers = 10
	wg.Add(numSearchers)

	for i := 0; i < numSearchers; i++ {
		go func() {
			searcher(done, target, paths, c)
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(c)
	}()

	for dirList := range c {
		fmt.Println(dirList)
	}

	if err := <-errc; err != nil {
		fmt.Println(err)
		return
	}

}
