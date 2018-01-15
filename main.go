package main

import (
	"context"
	"flag"
	"github.com/Devatoria/go-nsenter"
	"github.com/cyberdelia/go-metrics-graphite"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/rcrowley/go-metrics"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	graphiteAddress string
)

func init() {
	flag.StringVar(&graphiteAddress, "graphite", "", "graphite host")
	flag.Parse()
}

type ContainerNetstat struct {
	addr        *net.TCPAddr
	pid         int
	app         string
	proc        string
	no          int
	ESTABLISHED int64
	SYN_SENT    int64
	SYN_RECV    int64
	FIN_WAIT1   int64
	FIN_WAIT2   int64
	TIME_WAIT   int64
	CLOSE       int64
	CLOSE_WAIT  int64
	LAST_ACK    int64
	LISTEN      int64
	CLOSING     int64
}

func (cn *ContainerNetstat) send() {
	r := metrics.NewRegistry()
	prefix := "app." + cn.app + ".proc." + cn.proc + "." + strconv.Itoa(cn.no) + ".tcp_connections."
	metrics.GetOrRegisterGauge(prefix+"ESTABLISHED", r).Update(cn.ESTABLISHED)
	metrics.GetOrRegisterGauge(prefix+"SYN_SENT", r).Update(cn.SYN_SENT)
	metrics.GetOrRegisterGauge(prefix+"SYN_RECV", r).Update(cn.SYN_RECV)
	metrics.GetOrRegisterGauge(prefix+"FIN_WAIT1", r).Update(cn.FIN_WAIT1)
	metrics.GetOrRegisterGauge(prefix+"FIN_WAIT2", r).Update(cn.FIN_WAIT2)
	metrics.GetOrRegisterGauge(prefix+"TIME_WAIT", r).Update(cn.TIME_WAIT)
	metrics.GetOrRegisterGauge(prefix+"CLOSE", r).Update(cn.CLOSE)
	metrics.GetOrRegisterGauge(prefix+"CLOSE_WAIT", r).Update(cn.CLOSE_WAIT)
	metrics.GetOrRegisterGauge(prefix+"LAST_ACK", r).Update(cn.LAST_ACK)
	metrics.GetOrRegisterGauge(prefix+"LISTEN", r).Update(cn.LISTEN)
	metrics.GetOrRegisterGauge(prefix+"CLOSING", r).Update(cn.CLOSING)
	c := graphite.Config{
		Addr:          cn.addr,
		Registry:      r,
		FlushInterval: time.Minute,
		DurationUnit:  time.Millisecond,
		Percentiles:   []float64{0.5, 0.75, 0.99, 0.999},
		Prefix:        "lain_cloud.",
	}
	err := graphite.Once(c)
	if err != nil {
		log.Println(err)
		return
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
	}
	stdout, stderr, err := config.Execute("netstat", "-nt")
	if err != nil {
		log.Println(stderr)
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
	addr, err := net.ResolveTCPAddr("tcp", graphiteAddress)
	if err != nil {
		log.Fatal(err)
	}
	for range time.Tick(time.Minute) {
		dockerClient, err := client.NewClient(client.DefaultDockerHost, "1.24", nil, nil)
		if err != nil {
			log.Println(err)
			continue
		}
		containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
		if err != nil {
			log.Println(err)
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
					log.Println(err)
					return
				}
				names := strings.Split(json.Name, ".")
				if len(names) == 4 {
					no, err := strconv.Atoi(strings.Split(names[3], "-")[1][1:])
					if err != nil {
						log.Println(err)
						return
					}
					if no > 0 {
						cn := ContainerNetstat{
							addr: addr,
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
	}
}
