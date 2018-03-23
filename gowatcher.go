package main

import (
  "fmt"
  "os/exec"
  "strings"
  "bufio"
  "os"
  "os/signal"
  "syscall"
  "reflect"
  "path/filepath"
  "log"

  "github.com/fsnotify/fsnotify"
)

//
var watcher *fsnotify.Watcher

// main
func main() {

  // creates a new file watcher
  setpath := os.Args[1]
  abspath, _ := filepath.Abs(setpath)
  procname := os.Args[2]
  watcher, _ = fsnotify.NewWatcher()
  defer watcher.Close()

  // starting at the root of the project, walk each file/directory searching for
  // directories
  fmt.Println("Watching: "+abspath)
  if err := filepath.Walk(abspath, watchDir); err != nil {
    fmt.Println("ERROR", err)
  }

  run  := "pmgo start " + setpath + " " + procname + " > /dev/null"
  kill := "pmgo stop " + procname + " > /dev/null && pmgo delete " + procname + " > /dev/null"

  cmd := exec.Command("sh", "-c", kill + " && sleep 2 && " + run)
  stdout, _ := cmd.CombinedOutput()
  fmt.Println(string(stdout))

  //
  done := make(chan bool)
  c := make(chan os.Signal, 2)
  signal.Notify(c, os.Interrupt, syscall.SIGTERM)
  //
  go func() {
    for {
      select {
      // watch for events
      case event := <-watcher.Events:
        if strings.Contains(event.Name, ".go") && event.Op == 2{
          fmt.Printf("UPDATED: %s\n", event.Name)
          cmd2 := exec.Command("sh", "-c", "go test")
          stdout2, _ := cmd2.CombinedOutput()
          fmt.Println(string(stdout2))

          cmd := exec.Command("sh", "-c", kill + " && sleep 2 && " + run)
          stdout, _ := cmd.CombinedOutput()
          fmt.Println(string(stdout))
        }

        // watch for errors
      case err := <-watcher.Errors:
        fmt.Println("ERROR", err)
      }

    }
    }()

  go func() {
      <-c
      fmt.Println("\nExiting...")
      cmd := exec.Command("sh", "-c", kill)
      stdout, _ := cmd.CombinedOutput()
      fmt.Println(string(stdout))
      done<-true
  }()

  <-done
}

func in_array(val interface{}, array interface{}) (exists bool, index int) {
    exists = false
    index = -1

    switch reflect.TypeOf(array).Kind() {
    case reflect.Slice:
        s := reflect.ValueOf(array)

        for i := 0; i < s.Len(); i++ {
            if reflect.DeepEqual(val, s.Index(i).Interface()) == true {
                index = i
                exists = true
                return
            }
        }
    }

    return
}

// watchDir gets run as a walk func, searching for directories to add watchers to
func watchDir(path string, fi os.FileInfo, err error) error {

  // since fsnotify can watch all the files in a directory, watchers only need
  // to be added to each nested directory
  ignoreFile, err := os.Open("./.gowatcher")
  ignores := []string{".."}

  if err != nil{
    log.Println("Error", err.Error())
    ignoreFile.Close()
  }else{
    scanner := bufio.NewScanner(ignoreFile)
    for scanner.Scan() {
      ignores = append(ignores, scanner.Text())
    }

    if err := scanner.Err(); err != nil {
        fmt.Println("Error", err.Error())
    }
  }

  if len(ignores) > 0 {
    if isIgnore,_ := in_array(fi.Name(), ignores); isIgnore {
      return filepath.SkipDir
    }

    if fi.Mode().IsDir() {
      return watcher.Add(path)
    }
  }else{
    if fi.Mode().IsDir() {
      return watcher.Add(path)
    }
  }

  return nil
}
