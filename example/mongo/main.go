package main

import (
	"flag"
	"log"
	"time"

	"gopkg.in/mgo.v2"

	"github.com/jimmy-go/jobq"
)

var (
	maxWs = flag.Int("max-workers", 3, "Number of workers.")
	maxQs = flag.Int("max-queue", 10, "Number of queue works.")
	tasks = flag.Int("tasks", 20, "Number of tasks.")

	hosts  = flag.String("host", "", "Mongo host.")
	dbName = flag.String("database", "", "Mongo database.")
	u      = flag.String("username", "", "Mongo username.")
	p      = flag.String("password", "", "Mongo password.")
)

// Post struct.
type Post struct {
	Link string `bson:"link"`
}

func main() {
	flag.Parse()
	log.SetFlags(log.Lshortfile)
	log.Printf("workers [%d]", *maxWs)
	log.Printf("queue len [%d]", *maxQs)
	log.Printf("tasks [%d]", *tasks)
	log.Printf("hosts [%v]", *hosts)
	log.Printf("dbName [%v]", *dbName)
	log.Printf("u [%v]", *u)
	log.Printf("p [%v]", *p)

	errc := make(chan error)
	go func() {
		for err := range errc {
			if err != nil {
				log.Printf("main : error channel : err [%s]", err)
			}
		}
	}()

	// q := make(chan jobq.Job, *maxQs)
	q := make(chan jobq.Job)
	d := jobq.NewDispatcher(*maxWs, q, errc)
	d.Run()

	di := &mgo.DialInfo{
		Addrs:    []string{*hosts},
		Timeout:  1 * time.Second,
		Database: *dbName,
		Username: *u,
		Password: *p,
	}
	sess, err := mgo.DialWithInfo(di)
	if err != nil {
		panic(err)
	}
	sess.SetMode(mgo.Monotonic, true)

	for i := 0; i < *tasks; i++ {
		func(index int) {
			// declare some task.
			task := func(e chan error) {
				now := time.Now()
				var us []*Post
				err := sess.DB(*dbName).C("posts").Find(nil).Limit(100).All(&us)
				log.Printf("main : task [%d] users len [%v] done! [%s]", index, len(us), time.Since(now))
				e <- err
			}
			q <- task
		}(i)
	}

	// select {}
	time.Sleep(1 * time.Minute)
	for {
		log.Println("sleep 1 minute!")
		time.Sleep(1 * time.Minute)
	}
}
