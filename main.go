package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/Devatoria/go-nsenter"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/marpaia/graphite-golang"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	dockerHost   string
	graphiteHost string
	graphitePort int
)

func init() {
	flag.StringVar(&dockerHost, "docker", "127.0.0.1:2375", "docker host")
	flag.StringVar(&graphiteHost, "host", "", "graphite host")
	flag.IntVar(&graphitePort, "port", 2003, "graphite port")
	flag.Parse()
}

type ContainerNetstat struct {
	gh          *graphite.Graphite
	pid         int
	app         string
	proc        string
	no          int
	ESTABLISHED int
	SYN_SENT    int
	SYN_RECV    int
	FIN_WAIT1   int
	FIN_WAIT2   int
	TIME_WAIT   int
	CLOSE       int
	CLOSE_WAIT  int
	LAST_ACK    int
	LISTEN      int
	CLOSING     int
}

func (cn *ContainerNetstat) send() {
	cn.gh.Prefix = "lain_cloud."
	err := cn.gh.SimpleSend("app."+cn.app+".proc."+cn.proc+"."+strconv.Itoa(cn.no)+".tcp_connections.ESTABLISHED", strconv.Itoa(cn.ESTABLISHED))
	if err != nil {
		log.Println(err)
	}
	err = cn.gh.SimpleSend("app."+cn.app+".proc."+cn.proc+"."+strconv.Itoa(cn.no)+".tcp_connections.SYN_SENT", strconv.Itoa(cn.SYN_SENT))
	if err != nil {
		log.Println(err)
	}
	err = cn.gh.SimpleSend("app."+cn.app+".proc."+cn.proc+"."+strconv.Itoa(cn.no)+".tcp_connections.SYN_RECV", strconv.Itoa(cn.SYN_RECV))
	if err != nil {
		log.Println(err)
	}
	err = cn.gh.SimpleSend("app."+cn.app+".proc."+cn.proc+"."+strconv.Itoa(cn.no)+".tcp_connections.FIN_WAIT1", strconv.Itoa(cn.FIN_WAIT1))
	if err != nil {
		log.Println(err)
	}
	err = cn.gh.SimpleSend("app."+cn.app+".proc."+cn.proc+"."+strconv.Itoa(cn.no)+".tcp_connections.FIN_WAIT2", strconv.Itoa(cn.FIN_WAIT2))
	if err != nil {
		log.Println(err)
	}
	err = cn.gh.SimpleSend("app."+cn.app+".proc."+cn.proc+"."+strconv.Itoa(cn.no)+".tcp_connections.TIME_WAIT", strconv.Itoa(cn.TIME_WAIT))
	if err != nil {
		log.Println(err)
	}
	err = cn.gh.SimpleSend("app."+cn.app+".proc."+cn.proc+"."+strconv.Itoa(cn.no)+".tcp_connections.CLOSE", strconv.Itoa(cn.CLOSE))
	if err != nil {
		log.Println(err)
	}
	err = cn.gh.SimpleSend("app."+cn.app+".proc."+cn.proc+"."+strconv.Itoa(cn.no)+".tcp_connections.CLOSE_WAIT", strconv.Itoa(cn.CLOSE_WAIT))
	if err != nil {
		log.Println(err)
	}
	err = cn.gh.SimpleSend("app."+cn.app+".proc."+cn.proc+"."+strconv.Itoa(cn.no)+".tcp_connections.LAST_ACK", strconv.Itoa(cn.LAST_ACK))
	if err != nil {
		log.Println(err)
	}
	err = cn.gh.SimpleSend("app."+cn.app+".proc."+cn.proc+"."+strconv.Itoa(cn.no)+".tcp_connections.LISTEN", strconv.Itoa(cn.LISTEN))
	if err != nil {
		log.Println(err)
	}
	err = cn.gh.SimpleSend("app."+cn.app+".proc."+cn.proc+"."+strconv.Itoa(cn.no)+".tcp_connections.CLOSING", strconv.Itoa(cn.CLOSING))
	if err != nil {
		log.Println(err)
	}
}

func removeEmpty(array []string) []string {
	var newArray []string
	for _, i := range array {
		if i != " " && i != "" {
			newArray = append(newArray, i)
		}
	}
	return newArray
}

func (cn *ContainerNetstat) netstat() {
	config := &nsenter.Config{
		Target: cn.pid,
		Net:    true,
	}
	stdout, stderr, err := config.Execute("netstat", "-nt")
	if err != nil {
		fmt.Println(stderr)
		return
	}

	lines := strings.Split(string(stdout), "\n")
	for i := 1; i < len(lines)-1; i++ {
		lineArray := removeEmpty(strings.Split(strings.TrimSpace(lines[i]), " "))
		switch lineArray[5] {
		case "ESTABLISHED":
			cn.ESTABLISHED += 1
		case "SYN_SENT":
			cn.SYN_SENT += 1
		case "SYN_RECV":
			cn.SYN_RECV += 1
		case "FIN_WAIT1":
			cn.FIN_WAIT1 += 1
		case "FIN_WAIT2":
			cn.FIN_WAIT2 += 1
		case "TIME_WAIT":
			cn.TIME_WAIT += 1
		case "CLOSE":
			cn.CLOSE += 1
		case "CLOSE_WAIT":
			cn.CLOSE_WAIT += 1
		case "LAST_ACK":
			cn.LAST_ACK += 1
		case "LISTEN":
			cn.LISTEN += 1
		case "CLOSING":
			cn.CLOSING += 1
		}
	}
	cn.send()
}

func main() {
	for {
		dockerClient, err := client.NewClient("tcp://"+dockerHost, "1.24", nil, nil)
		if err != nil {
			fmt.Println(err)
			continue
		}
		gh, err := graphite.NewGraphite(graphiteHost, graphitePort)
		if err != nil {
			log.Println(err)
			return
		}
		containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
		if err != nil {
			fmt.Println(err)
			continue
		}
		var wg sync.WaitGroup
		wg.Add(len(containers))
		for _, container := range containers {
			id := container.ID
			go func() {
				defer wg.Done()
				json, err := dockerClient.ContainerInspect(context.Background(), id)
				if err != nil {
					fmt.Println(err)
					return
				}
				names := strings.Split(json.Name, ".")
				if len(names) == 4 {
					no, err := strconv.Atoi(strings.Split(names[3], "-")[1][1:])
					if err != nil {
						fmt.Println(err)
						return
					}
					if no > 0 {
						cn := ContainerNetstat{
							gh:   gh,
							pid:  json.State.Pid,
							app:  names[0][1:],
							proc: names[2],
							no:   no,
						}
						cn.netstat()
					}
				}
			}()
		}
		wg.Wait()
		time.Sleep(time.Minute)
	}
}
