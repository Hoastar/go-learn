package main

import (
	"fmt"
	"net/http"

	"golang.org/x/sync/errgroup"
)

/*func main() {
	o := &sync.Once{}
	for i := 0; i < 10; i++ {
		o.Do(func() {
			fmt.Println("only once")
		})
	}
}
*/

/*func main() {
	c := sync.NewCond(&sync.Mutex{})
	for i := 0; i < 10; i++ {
		go listen(c, i)
	}

	time.Sleep(1 * time.Second)
	go broadcast(c)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

}

func broadcast(c *sync.Cond) {
	c.L.Lock()
	c.Broadcast()
	c.L.Unlock()
}

func listen(c *sync.Cond, i int) {
	c.L.Lock()
	c.Wait()
	fmt.Printf("listen: %v\n", i)
	c.L.Unlock()
}
*/
func main() {
	var g errgroup.Group
	var urls = []string{
		"http://www.google.org",
		"http://www.google.com/",
		"http://www.somestupidname.com/",
	}

	for i := range urls {
		url := urls[i]

		g.Go(func() error {
			resp, err := http.Get(url)
			if err == nil {
				resp.Body.Close()
			}
			return err
		})
	}

	if err := g.Wait(); err == nil {
		fmt.Println("Successfully fetched all RULS.")
	}
}
