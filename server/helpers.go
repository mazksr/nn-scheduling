package main

import (
	"io"
	"net/http"
)

func makeInference(r *http.Request, url string) (*http.Response, error) {
	request, err := http.NewRequest(r.Method, url, r.Body)
	if err != nil {
		return &http.Response{}, err
	}
	request.Header = r.Header.Clone()
	request.Host = ""

	client := &http.Client{}
	return client.Do(request)
}

func sendResult(w http.ResponseWriter, res *http.Response) error {
	for key, val := range res.Header {
		for _, v := range val {
			w.Header().Add(key, v)
		}
	}

	w.WriteHeader(res.StatusCode)
	_, err := io.Copy(w, res.Body)

	return err
}

func forward(w http.ResponseWriter, r *http.Request, url string) error {
	res, err := makeInference(r, url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	err = sendResult(w, res)

	return err
}
