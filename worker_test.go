package jobq

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"time"
)

type MyWorker struct {
	client *http.Client
}

func NewMyWorker(timeout time.Duration) (*MyWorker, error) {
	w := &MyWorker{
		client: &http.Client{
			Timeout: timeout,
		},
	}
	return w, nil
}

func (w *MyWorker) Do(ctx context.Context, v interface{}) error {
	p, ok := v.(*Payload)
	if !ok {
		return errors.New("unprocessable payload")
	}
	uri := p.URL + p.Params.Encode()
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil
	}
	resp, err := w.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil
	}
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return err
	}
	err = resp.Body.Close()
	if err != nil {
		return nil
	}
	log.Printf("Do : resp body [%s]", buf.String())
	return nil
}

type Payload struct {
	URL    string
	Params url.Values
}
